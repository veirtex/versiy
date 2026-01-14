package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"versiy/internal/database"
)

func setupTestApp(t *testing.T) (*application, pgxmock.PgxPoolIface) {
	s := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	mockDB, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	require.NoError(t, err)

	return &application{
		store: database.NewStorage(nil, client),
		cfg: config{
			secret:      "test-secret",
			defaultLink: "https://api.versiy.cc/",
			redisConfig: redisConfig{
				defaultTTL: time.Hour,
			},
			rateLimiting: rateLimitConfig{
				size:     10,
				duration: 15 * time.Second,
			},
		},
		mut: &sync.Mutex{},
	}, mockDB
}

func TestEndpoint_Health(t *testing.T) {
	t.Run("returns 200 OK", func(t *testing.T) {
		app := &application{env: "test"}

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		app.health(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "test", response["env"])
	})

	t.Run("accepts POST requests", func(t *testing.T) {
		app := &application{}

		req := httptest.NewRequest("POST", "/health", nil)
		w := httptest.NewRecorder()

		app.health(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestEndpoint_StoreURL(t *testing.T) {
	app, _ := setupTestApp(t)

	t.Run("returns 400 for missing URL", func(t *testing.T) {
		payload := map[string]string{}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		app.StoreURL(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 400 for empty URL", func(t *testing.T) {
		payload := map[string]string{"original_url": ""}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		app.StoreURL(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 400 for invalid URL format", func(t *testing.T) {
		payload := map[string]string{"original_url": "not-a-valid-url"}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		app.StoreURL(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 400 for unsupported scheme", func(t *testing.T) {
		payload := map[string]string{"original_url": "ftp://example.com"}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		app.StoreURL(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 400 for invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", strings.NewReader("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		app.StoreURL(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 400 for empty JSON body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		app.StoreURL(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("rejects unknown fields in JSON", func(t *testing.T) {
		payload := map[string]interface{}{
			"original_url":  "https://example.com",
			"unknown_field": "value",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		app.StoreURL(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestEndpoint_GetURL(t *testing.T) {
	app, mockDB := setupTestApp(t)
	defer mockDB.Close()

	t.Run("returns 404 for non-existent code", func(t *testing.T) {
		mockDB.ExpectQuery(`SELECT original_url FROM urls`).
			WithArgs("nonexistent").
			WillReturnError(pgx.ErrNoRows)

		r := app.mount()
		req := httptest.NewRequest("GET", "/nonexistent", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("returns 400 for invalid URL in cache", func(t *testing.T) {
		ctx := context.Background()
		err := app.store.URL.CacheResult(ctx, "testcode", "not a url", time.Hour)
		require.NoError(t, err)

		r := app.mount()
		req := httptest.NewRequest("GET", "/testcode", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 400 for javascript: scheme", func(t *testing.T) {
		ctx := context.Background()
		err := app.store.URL.CacheResult(ctx, "testcode", "javascript:alert('xss')", time.Hour)
		require.NoError(t, err)

		r := app.mount()
		req := httptest.NewRequest("GET", "/testcode", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 400 for ftp: scheme", func(t *testing.T) {
		ctx := context.Background()
		err := app.store.URL.CacheResult(ctx, "testcode", "ftp://example.com", time.Hour)
		require.NoError(t, err)

		r := app.mount()
		req := httptest.NewRequest("GET", "/testcode", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("redirects to cached URL", func(t *testing.T) {
		ctx := context.Background()
		err := app.store.URL.CacheResult(ctx, "testcode", "https://example.com", time.Hour)
		require.NoError(t, err)

		r := app.mount()
		req := httptest.NewRequest("GET", "/testcode", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, "https://example.com", w.Header().Get("Location"))
	})
}

func TestEndpoint_RateLimiting(t *testing.T) {
	t.Run("allows requests within limit", func(t *testing.T) {
		app, _ := setupTestApp(t)
		app.cfg.rateLimiting.size = 5

		r := app.mount()

		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("POST", "/", strings.NewReader(`{"original_url":"https://example.com"}`))
			req.Header.Set("Content-Type", "application/json")
			req.RemoteAddr = "192.168.1.1:1234"
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.NotEqual(t, http.StatusTooManyRequests, w.Code)
		}
	})

	t.Run("blocks requests exceeding limit", func(t *testing.T) {
		app, _ := setupTestApp(t)
		app.cfg.rateLimiting.size = 2

		r := app.mount()

		allowedCount := 0
		blockedCount := 0

		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("POST", "/", strings.NewReader(`{"original_url":"https://example.com"}`))
			req.Header.Set("Content-Type", "application/json")
			req.RemoteAddr = "192.168.1.1:1234"
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code == http.StatusTooManyRequests {
				blockedCount++
				assert.NotEmpty(t, w.Header().Get("Retry-After"))
			} else {
				allowedCount++
			}
		}

		assert.Equal(t, 2, allowedCount)
		assert.Equal(t, 3, blockedCount)
	})

	t.Run("rate limits by IP address", func(t *testing.T) {
		app, _ := setupTestApp(t)
		app.cfg.rateLimiting.size = 2

		r := app.mount()

		req1 := httptest.NewRequest("POST", "/", strings.NewReader(`{"original_url":"https://example.com"}`))
		req1.Header.Set("Content-Type", "application/json")
		req1.RemoteAddr = "192.168.1.1:1234"
		w1 := httptest.NewRecorder()
		r.ServeHTTP(w1, req1)

		req2 := httptest.NewRequest("POST", "/", strings.NewReader(`{"original_url":"https://example.com"}`))
		req2.Header.Set("Content-Type", "application/json")
		req2.RemoteAddr = "192.168.1.1:1234"
		w2 := httptest.NewRecorder()
		r.ServeHTTP(w2, req2)

		req3 := httptest.NewRequest("POST", "/", strings.NewReader(`{"original_url":"https://example.com"}`))
		req3.Header.Set("Content-Type", "application/json")
		req3.RemoteAddr = "192.168.1.2:5678"
		w3 := httptest.NewRecorder()
		r.ServeHTTP(w3, req3)

		req4 := httptest.NewRequest("POST", "/", strings.NewReader(`{"original_url":"https://example.com"}`))
		req4.Header.Set("Content-Type", "application/json")
		req4.RemoteAddr = "192.168.1.1:1234"
		w4 := httptest.NewRecorder()
		r.ServeHTTP(w4, req4)

		assert.Equal(t, http.StatusTooManyRequests, w4.Code)
	})
}

func TestEndpoint_Cookies(t *testing.T) {
	t.Run("sets device_id cookie on first request", func(t *testing.T) {
		app, _ := setupTestApp(t)

		r := app.mount()
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		cookies := w.Result().Cookies()
		deviceCookie := findCookie(cookies, "device_id")

		require.NotNil(t, deviceCookie)
		assert.Equal(t, "/", deviceCookie.Path)

		_, err := uuid.Parse(deviceCookie.Value)
		assert.NoError(t, err)
	})

	t.Run("preserves existing valid cookie", func(t *testing.T) {
		app, _ := setupTestApp(t)
		existingID := uuid.New().String()

		r := app.mount()
		req := httptest.NewRequest("GET", "/health", nil)
		req.AddCookie(&http.Cookie{Name: "device_id", Value: existingID})
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		cookies := w.Result().Cookies()
		deviceCookie := findCookie(cookies, "device_id")

		assert.Nil(t, deviceCookie)
	})

	t.Run("recreates invalid cookie", func(t *testing.T) {
		app, _ := setupTestApp(t)

		r := app.mount()
		req := httptest.NewRequest("GET", "/health", nil)
		req.AddCookie(&http.Cookie{Name: "device_id", Value: "not-a-uuid"})
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		cookies := w.Result().Cookies()
		deviceCookie := findCookie(cookies, "device_id")

		require.NotNil(t, deviceCookie)

		_, err := uuid.Parse(deviceCookie.Value)
		assert.NoError(t, err)
	})
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}
