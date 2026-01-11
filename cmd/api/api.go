package main

import (
	"net/http"
	"time"
	"versiy/internal/database"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type application struct {
	store database.Storage
	cfg   config
}

type config struct {
	addr           string
	secret         string
	defaultLink    string
	postgresConfig postgreSQLConfig
	redisConfig    redisConfig
}

type postgreSQLConfig struct {
	addr           string
	max_idle_conns int
}

type redisConfig struct {
	addr       string
	defualtTTL int
}

func (app *application) mount() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/health", app.health)
	r.Post("/", app.StoreURL)
	r.Get("/{code}", app.GetURL)

	return r
}

func (app *application) run(r *chi.Mux) {
	if err := http.ListenAndServe(app.cfg.addr, r); err != nil {
		panic(err)
	}
}
