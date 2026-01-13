package main

import (
	"net/http"
	"strconv"
	"time"
)

func (app *application) fixedSizeWindow(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()

		app.mut.Lock()
		defer app.mut.Unlock()

		if now.After(app.cfg.rateLimiting.resetTime) {
			app.cfg.rateLimiting.counter = 0
			app.cfg.rateLimiting.resetTime = now.Add(app.cfg.rateLimiting.duration)
		}

		if app.cfg.rateLimiting.counter >= app.cfg.rateLimiting.size {
			reset := app.cfg.rateLimiting.resetTime

			w.Header().Set("Retry-After", strconv.Itoa(int(time.Until(reset).Seconds())))
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		app.cfg.rateLimiting.counter++
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}
