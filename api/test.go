package handler

import (
	"encoding/json"
	"net/http"
)

// Handler is a simple test handler for Vercel
func Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"status": "ok",
		"message": "PipCal is working!",
		"path": r.URL.Path,
		"method": r.Method,
	}

	json.NewEncoder(w).Encode(response)
}
