package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/robfig/cron/v3"
)

func main() {
	initFirebase()

	// Start the cron job in a separate goroutine
	go startCronJobs()

	// Email routes
	http.HandleFunc("/send-email", withCORS(sendEmailHandler))
	http.HandleFunc("/send-single-email", withCORS(sendSingleEmailHandler))
	http.HandleFunc("/register-meal-preference", withCORS(registerMealPreferenceHandler))

	// Recipes routes
	http.HandleFunc("/add-recipe", withCORS(addRecipesHandler))
	http.HandleFunc("/get-all-recipes", withCORS(getAllRecipesHandler))
	http.HandleFunc("/generate-recipes", withCORS(generateRecipesHandler))

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

// Start Cron Jobs
func startCronJobs() {
	// Manually set IST (UTC+5:30)
	ist := time.FixedZone("IST", 5*60*60+30*60)

	c := cron.New(cron.WithLocation(ist))

	// 7:30 AM IST
	c.AddFunc("30 7 * * *", func() {
		log.Println("Cron: Sending Breakfast Recipes")
		checkPreferencesAndSend("breakfast")
	})

	// 12:30 PM IST
	c.AddFunc("30 12 * * *", func() {
		log.Println("Cron: Sending Lunch Recipes")
		checkPreferencesAndSend("lunch")
	})

	// 6:30 PM IST
	c.AddFunc("30 18 * * *", func() {
		log.Println("Cron: Sending Dinner Recipes")
		checkPreferencesAndSend("dinner")
	})

	c.Start()
	log.Println("Cron jobs scheduled.")
}
