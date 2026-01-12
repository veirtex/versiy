package main

import (
	"errors"
	"log"
	"net/http"
)

func responseError(w http.ResponseWriter, err error, statusCode int) {
	log.Printf("error %v", err)
	if statusCode == http.StatusInternalServerError {
		encodeJSON(w, map[string]string{
			"code":  http.StatusText(statusCode),
			"error": "something went wrong",
		}, statusCode)
		return
	}
	encodeJSON(w, map[string]string{
		"code":  http.StatusText(statusCode),
		"error": err.Error(),
	}, statusCode)
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
