package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type requestData struct {
	requests  int
	resetTime time.Time
}

var devices map[string]*requestData

func init() {
	devices = make(map[string]*requestData)
}

type deviceIDKeyType string

const deviceIDKey deviceIDKeyType = "device_id"

func (app *application) fixedSizeWindow(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		deviceID := fmt.Sprintf("%s:%s", host, getValFromContext(r.Context()))
		now := time.Now()

		app.mut.Lock()
		device, ok := devices[deviceID]
		if !ok {
			device = &requestData{
				requests:  0,
				resetTime: time.Now().Add(app.cfg.rateLimiting.duration),
			}
			devices[deviceID] = device
		}

		if now.After(device.resetTime) {
			device.requests = 0
			device.resetTime = now.Add(app.cfg.rateLimiting.duration)
		}

		if device.requests >= app.cfg.rateLimiting.size {
			reset := device.resetTime
			app.mut.Unlock()

			w.Header().Set("x-rate-limited", strconv.Itoa(int(time.Now().Unix())))
			w.Header().Set("x-retry-After", strconv.Itoa(int(time.Until(reset).Seconds())))
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		device.requests++

		app.mut.Unlock()
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
