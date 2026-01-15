package main

import (
	"context"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"versiy/internal/security"
)

type deviceIDKeyType string

const deviceIDKey deviceIDKeyType = "device_id"

func (app *application) fixedSizeWindow(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Extract X-Forwarded-For header for proxy/load balancer cases
		xForwardedFor := r.Header.Get("X-Forwarded-For")

		// Get device ID from context if available
		idFromCtx := getValFromContext(ctx)

		// Get rate limit identifier (prefers IP, falls back to cookie)
		rateLimitKey := security.GetRateLimitIdentifier(r.RemoteAddr, xForwardedFor, idFromCtx)

		// Increment rate limit counter
		req, err := app.store.Users.IncrUser(ctx, rateLimitKey, app.cfg.rateLimiting.duration)
		if err != nil {
			app.internalServerError(w, err)
			return
		}

		// Check if rate limit exceeded
		if req > app.cfg.rateLimiting.size {
			w.Header().Set("Retry-After", strconv.Itoa(int(app.cfg.rateLimiting.duration)))
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

func (app *application) handleCookies(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookies := r.Cookies()

		// Validate and use only first device_id cookie
		var deviceID string
		needNew := true

		for _, cookie := range cookies {
			if cookie.Name == "device_id" {
				// Validate cookie value
				if err := security.ValidateCookieValue(cookie.Value); err == nil {
					if _, err := uuid.Parse(cookie.Value); err == nil && cookie.Value != "" {
						deviceID = cookie.Value
						needNew = false
						break
					}
				}
			}
		}

		if needNew {
			deviceID = uuid.NewString()

			// Always set Secure flag for production security
			// Use SameSite=Strict for better CSRF protection
			http.SetCookie(w, &http.Cookie{
				Name:     "device_id",
				Value:    deviceID,
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
				Secure:   true,
				MaxAge:   60 * 60 * 24 * 365,
			})
		}

		r = r.WithContext(context.WithValue(r.Context(), deviceIDKey, deviceID))
		next.ServeHTTP(w, r)
	})
}

func getValFromContext(ctx context.Context) string {
	switch v := ctx.Value(deviceIDKey).(type) {
	case string:
		return v
	default:
		return ""
	}
}
