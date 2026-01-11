package main

import "net/http"

func (app *application) health(w http.ResponseWriter, r *http.Request) {
	if err := encodeJSON(w, map[string]string{
		"env": app.env,
	}); err != nil {
		app.internalServerError(w, err)
		return
	}
}
