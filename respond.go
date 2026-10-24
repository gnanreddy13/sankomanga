package main

import (
	"encoding/json"
	"log"
	"net/http"
)

// errorResponse is the single JSON shape used for every error: {"error": "..."}.
type errorResponse struct {
	Error string `json:"error"`
}

// writeJSON sends v as a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	// Encode first, before writing anything. If encoding failed halfway through
	// writing, the client would get a 200 status with a broken body.
	body, err := json.Marshal(v)
	if err != nil {
		log.Printf("error: encoding JSON response: %v", err)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal error"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	w.Write(body)
}

// writeError sends an error in the standard {"error": "..."} shape.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}
