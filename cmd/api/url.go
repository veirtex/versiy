package main

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"
	"versiy/internal/database"
	"versiy/internal/util"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

func (app *application) StoreURL(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OriginalURL string `json:"original_url"`
	}
	if err := decodeJSON(r, &req); err != nil {
		app.badRequest(w, err)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), time.Second*5)
	defer cancel()

	defualtExpiry := time.Now().Add(time.Hour * 24 * 30)

	originalURL, err := util.NormalizeURL(req.OriginalURL)
	if err != nil {
		app.badRequest(w, err)
		return
	}

	short_code, err := app.store.URL.Store(ctx, database.URLInsert{
		OriginalURL: originalURL,
		ExpiresAt:   &defualtExpiry,
	}, app.cfg.secret)
	if err != nil {
		app.badRequest(w, err)
		return
	}

	if err := encodeJSON(w, map[string]string{
		"url": app.cfg.defaultLink + short_code,
	}, http.StatusCreated); err != nil {
		app.internalServerError(w, err)
		return
	}
}

func (app *application) GetURL(w http.ResponseWriter, r *http.Request) {
	shortCode := chi.URLParam(r, "code")
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*5)
	defer cancel()

	cachedURL, err := app.store.URL.CheckCached(ctx, shortCode)
	if cachedURL != "" {
		if err := app.store.URL.UpdateClicks(ctx, shortCode); err != nil {
			app.internalServerError(w, err)
			return
		}
		http.Redirect(w, r, cachedURL, http.StatusFound)
		return
	}

	originalURL, err := app.store.URL.Get(ctx, shortCode)

	if err != nil {
		switch err {
		case pgx.ErrNoRows:
			app.notFoundError(w)
		default:
			app.badRequest(w, err)
		}
		return
	}

	u, err := url.Parse(originalURL)
	if err != nil || !u.IsAbs() {
		app.badRequest(w, errors.New("invalid redirect url"))
		return
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		app.badRequest(w, errors.New("invalid redirect scheme"))
		return
	}

	if err := app.store.URL.LastTimeAccessed(ctx, shortCode); err != nil {
		switch err {
		case pgx.ErrNoRows:
			app.notFoundError(w)
		default:
			app.internalServerError(w, err)
		}
		return
	}

	if err := app.store.URL.UpdateClicks(ctx, shortCode); err != nil {
		app.internalServerError(w, err)
		return
	}

	if err = app.store.URL.CacheResult(ctx, shortCode, originalURL, app.cfg.redisConfig.defualtTTL); err != nil {
		app.internalServerError(w, err)
		return
	}

	http.Redirect(w, r, originalURL, http.StatusFound)
}
