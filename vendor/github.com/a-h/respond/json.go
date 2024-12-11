package respond

import (
	"encoding/json"
	"net/http"
)

func WithJSON(w http.ResponseWriter, v any, status int) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		WithError(w, "failed to encode", http.StatusInternalServerError)
	}
}
