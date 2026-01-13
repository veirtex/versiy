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
