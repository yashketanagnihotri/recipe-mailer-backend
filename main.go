package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	initFirebase()

	// Wrap handlers with CORS middleware
	http.HandleFunc("/send-email", withCORS(sendEmailHandler))
	http.HandleFunc("/add-recipe", withCORS(addRecipesHandler))
	http.HandleFunc("/send-single-email", withCORS(sendSingleEmailHandler)) 

	fmt.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// Middleware to handle CORS
func withCORS(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Handle preflight (OPTIONS) requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		handler(w, r)
	}
}
