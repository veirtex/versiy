package main

import (
	"errors"
	"net/http"
)

func responseError(w http.ResponseWriter, err error, statusCode int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	if statusCode == http.StatusInternalServerError {
		encodeJSON(w, map[string]string{
			"code":  http.StatusText(statusCode),
			"error": "something went wrong",
		})
		return
	}
	encodeJSON(w, map[string]string{
		"code":  http.StatusText(statusCode),
		"error": err.Error(),
	})
}

func (app *application) badRequest(w http.ResponseWriter, err error) {
	responseError(w, err, http.StatusBadRequest)
}

func (app *application) internalServerError(w http.ResponseWriter, err error) {
	responseError(w, err, http.StatusInternalServerError)
}

func (app *application) notFoundError(w http.ResponseWriter) {
	responseError(w, errors.New("not found"), http.StatusNotFound)
}
