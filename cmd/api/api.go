package main

import (
	"net/http"
	"sync"
	"time"
	"versiy/internal/database"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type application struct {
	store database.Storage
	cfg   config
	env   string
	mut   *sync.Mutex
}

type config struct {
	addr           string
	secret         string
	defaultLink    string
	postgresConfig postgreSQLConfig
	redisConfig    redisConfig
	rateLimiting   rateLimitConfig
}

type postgreSQLConfig struct {
	addr           string
	max_idle_conns int
}

type redisConfig struct {
	addr         string
	pswd         string
	defualtTTL   time.Duration
	dialTimeout  time.Duration
	readTimeout  time.Duration
	writeTimeout time.Duration
	poolTimeout  time.Duration
}

type rateLimitConfig struct {
	size     int
	duration time.Duration
}

func (app *application) mount() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(app.handleCookies)
	r.Use(app.securityHeaders)

	r.Get("/health", app.health)

	r.Group(func(r chi.Router) {
		r.Use(app.fixedSizeWindow)
		r.Post("/", app.StoreURL)
	})

	r.Get("/{code}", app.GetURL)

	return r
}

func (app *application) run(r *chi.Mux) {
	if err := http.ListenAndServe(app.cfg.addr, r); err != nil {
		panic(err)
	}
}

func (app *application) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Content Security Policy - restricts sources of content
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'none'; object-src 'none'; base-uri 'self'; frame-ancestors 'none';")

		// X-Content-Type-Options - prevents MIME sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// X-Frame-Options - prevents clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// X-XSS-Protection - XSS filter (legacy, but still useful)
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Strict-Transport-Security - enforces HTTPS
		if r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}

		// Referrer-Policy - controls referrer information
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions-Policy - restricts browser features
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		// Content-Type - enforce JSON for API responses
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		next.ServeHTTP(w, r)
	})
}
