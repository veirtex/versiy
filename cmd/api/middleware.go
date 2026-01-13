package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"

	"github.com/google/uuid"
)

type deviceIDKeyType string

const deviceIDKey deviceIDKeyType = "device_id"

func (app *application) fixedSizeWindow(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		deviceID := fmt.Sprintf("%s:%s", host, getValFromContext(r.Context()))
		ctx := r.Context()

		req, err := app.store.Users.IncrUser(ctx, deviceID, app.cfg.rateLimiting.duration)
		if err != nil {
			app.internalServerError(w, err)
			return
		}

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
		cookie, err := r.Cookie("device_id")

		needNew := false
		var deviceID string

		switch err {
		case http.ErrNoCookie:
			needNew = true
		case nil:
			if _, err := uuid.Parse(cookie.Value); err != nil {
				needNew = true
			} else {
				if cookie.Value == "" {
					needNew = true
				} else {
					deviceID = cookie.Value
				}
			}
		default:
			next.ServeHTTP(w, r)
			return
		}

		if needNew {
			deviceID = uuid.NewString()
			http.SetCookie(w, &http.Cookie{
				Name:     "device_id",
				Value:    deviceID,
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
				Secure:   r.TLS != nil,
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
