package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"example.com/edtech-verification/internal/enrollment"
)

func main() {
	workflow := enrollment.Workflow{Sender: &enrollment.Client{APIKey: os.Getenv("INFRAI_API_KEY"), MaxRetries: 3}}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /signup", func(w http.ResponseWriter, r *http.Request) {
		var signup enrollment.Signup
		if err := json.NewDecoder(r.Body).Decode(&signup); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		outcome, err := workflow.Start(r.Context(), signup)
		if err != nil {
			status := http.StatusBadGateway
			var apiErr *enrollment.InfraiError
			switch {
			case errors.Is(err, enrollment.ErrCourseNotReady), errors.Is(err, enrollment.ErrDeadlinePassed):
				status = http.StatusUnprocessableEntity
			case errors.As(err, &apiErr) && apiErr.HTTPStatus >= 400 && apiErr.HTTPStatus < 500:
				status = apiErr.HTTPStatus
			}
			writeJSON(w, status, outcome)
			return
		}
		writeJSON(w, http.StatusAccepted, outcome)
	})

	server := &http.Server{Addr: ":8080", Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Printf("enrollment verifier listening on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode response: %v", err)
	}
}
