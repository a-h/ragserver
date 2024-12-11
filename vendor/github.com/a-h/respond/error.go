package respond

import (
	"encoding/json"
	"net/http"
)

func WithError(w http.ResponseWriter, msg string, status int) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Error{
		Message:    msg,
		StatusCode: status,
	})
}

type Error struct {
	Message    string `json:"message"`
	StatusCode int    `json:"statusCode"`
}
