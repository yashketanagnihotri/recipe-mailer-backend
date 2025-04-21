package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"google.golang.org/api/iterator"
	"gopkg.in/gomail.v2"
)

type EmailRequest struct {
	Receivers []string `json:"receivers"`
}

type MealPreference struct {
	Email     string `json:"email"`
	Breakfast bool   `json:"breakfast"`
	Lunch     bool   `json:"lunch"`
	Dinner    bool   `json:"dinner"`
}

func registerMealPreferenceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var pref MealPreference
	if err := json.NewDecoder(r.Body).Decode(&pref); err != nil || pref.Email == "" {
		http.Error(w, "Invalid input data", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	_, err := firestoreClient.Collection("meal_preferences").Doc(pref.Email).Set(ctx, map[string]interface{}{
		"email":     pref.Email,
		"breakfast": pref.Breakfast,
		"lunch":     pref.Lunch,
		"dinner":    pref.Dinner,
	})
	if err != nil {
		log.Println("Failed to store meal preference:", err)
		http.Error(w, "Failed to store data", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Meal preferences registered for: " + pref.Email))
}

// Picks a random recipe from Firestore
func getRandomRecipe() (Recipe, error) {
	recipes, err := getRecipesFromFirestore()
	if err != nil {
		log.Println("Error fetching recipes:", err)
		return Recipe{}, err
	}

	rand.Seed(time.Now().UnixNano())
	return recipes[rand.Intn(len(recipes))], nil
}

// Stores new emails in Firestore and returns all stored emails
func storeAndGetEmails(receivers []string) ([]string, error) {
	ctx := context.Background()
	bulkWriter := firestoreClient.BulkWriter(ctx)

	// Retrieve all stored emails
	var storedEmailsMap = make(map[string]bool)
	iter := firestoreClient.Collection("email_recipients").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read emails from Firestore: %v", err)
		}
		email := doc.Data()["email"].(string)
		storedEmailsMap[email] = true
	}

	// Add new emails if they don't already exist
	for _, email := range receivers {
		if _, exists := storedEmailsMap[email]; !exists {
			docRef := firestoreClient.Collection("email_recipients").Doc(email)
			bulkWriter.Set(docRef, map[string]interface{}{
				"email": email,
			})
			storedEmailsMap[email] = true
		}
	}

	bulkWriter.End()

	// Convert map to slice
	var allEmails []string
	for email := range storedEmailsMap {
		allEmails = append(allEmails, email)
	}

	return allEmails, nil
}

// Generates a visually attractive animated HTML email template
func generateEmailBody(recipe Recipe) string {
	return fmt.Sprintf(`
		<!DOCTYPE html>
		<html lang="en">
		<head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<title>Delicious Recipe</title>
			<link href="https://fonts.googleapis.com/css2?family=Pacifico&family=Roboto:wght@300;400;700&display=swap" rel="stylesheet">
			<style>
				body { margin: 0; font-family: 'Roboto', sans-serif; background: linear-gradient(to right, #ff758c, #ff7eb3); color: #fff; text-align: center; padding: 20px; }
				.container { background: rgba(255, 255, 255, 0.15); padding: 20px; border-radius: 15px; box-shadow: 0px 10px 25px rgba(0, 0, 0, 0.2); max-width: 600px; margin: auto; animation: fadeIn 1s ease-in-out; }
				h1 { font-family: 'Pacifico', cursive; font-size: 28px; color: #ffe600; text-shadow: 2px 2px 5px rgba(0, 0, 0, 0.2); }
				p { font-size: 16px; line-height: 1.6; }
				ul, ol { text-align: left; display: inline-block; background: rgba(255, 255, 255, 0.2); padding: 15px; border-radius: 10px; width: 80%%; animation: slideUp 1s ease-in-out; }
				li { margin-bottom: 5px; }
				.footer { margin-top: 20px; font-size: 12px; opacity: 0.8; }
				@keyframes fadeIn { from { opacity: 0; transform: scale(0.9); } to { opacity: 1; transform: scale(1); } }
				@keyframes slideUp { from { transform: translateY(20px); opacity: 0; } to { transform: translateY(0); opacity: 1; } }
				.emoji { font-size: 50px; animation: bounce 1.5s infinite; }
				@keyframes bounce { 0%%, 100%% { transform: translateY(0); } 50%% { transform: translateY(-10px); } }
			</style>
		</head>
		<body>
			<div class="container">
				<div class="emoji">🍽️</div>
				<h1>%s</h1>
				<p><strong>%s</strong></p>
				<h3>🥕 Ingredients</h3>
				<ul>%s</ul>
				<h3>🔥 Instructions</h3>
				<ol>%s</ol>
			</div>
			<p class="footer">Bon Appétit! 🎉 | Stay Hungry, Stay Happy!</p>
		</body>
		</html>
	`, recipe.Title, recipe.Description, formatList(recipe.Ingredients), formatList(recipe.Instructions))
}

// Formats a list into HTML list items
func formatList(items []string) string {
	html := ""
	for _, item := range items {
		html += fmt.Sprintf("<li>%s</li>", item)
	}
	return html
}

// Sends email using Gmail SMTP
func sendEmail(to []string, body string) error {
	senderEmail := "yashagni1992@gmail.com"
	password := "eydewcznbacbgvqy"

	d := gomail.NewDialer("smtp.gmail.com", 587, senderEmail, password)

	for _, recipient := range to {
		m := gomail.NewMessage()
		m.SetHeader("From", senderEmail)
		m.SetHeader("To", recipient)
		m.SetHeader("Subject", "🍽️ Your Random Recipe for Today!")
		m.SetBody("text/html", body)

		if err := d.DialAndSend(m); err != nil {
			log.Println("Failed to send email to:", recipient, "Error:", err)
			return err
		}
	}

	return nil
}

// Handles API requests
func sendEmailHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var emailReq EmailRequest
	err := json.NewDecoder(r.Body).Decode(&emailReq)
	if err != nil {
		// Log error but continue to fetch stored emails
		log.Println("Invalid request body, fetching stored emails:", err)
		emailReq.Receivers = []string{}
	}

	// Store new emails (if any) and get all recipients
	allEmails, err := storeAndGetEmails(emailReq.Receivers)
	if err != nil {
		http.Error(w, "Failed to process emails", http.StatusInternalServerError)
		return
	}

	// If no emails are found in Firestore, return an error
	if len(allEmails) == 0 {
		http.Error(w, "No recipients found", http.StatusNotFound)
		return
	}

	// Fetch a random recipe
	recipe, err := getRandomRecipe()
	if err != nil {
		http.Error(w, "Failed to fetch recipe", http.StatusInternalServerError)
		return
	}

	// Generate email body and send emails
	emailBody := generateEmailBody(recipe)
	if err := sendEmail(allEmails, emailBody); err != nil {
		http.Error(w, "Failed to send email", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Emails sent successfully!"))
}

func sendSingleEmailHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Email string `json:"email"`
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.Email == "" {
		http.Error(w, "Invalid email input", http.StatusBadRequest)
		return
	}

	// Get a random recipe
	recipe, err := getRandomRecipe()
	if err != nil {
		http.Error(w, "Failed to fetch recipe", http.StatusInternalServerError)
		return
	}

	// Generate email body
	body := generateEmailBody(recipe)

	// Send to single email
	if err := sendEmail([]string{req.Email}, body); err != nil {
		http.Error(w, "Failed to send email", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Recipe sent to " + req.Email))
}

func checkPreferencesAndSend(meal string) {
	ctx := context.Background()
	iter := firestoreClient.Collection("meal_preferences").Documents(ctx)

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Println("Error reading meal preferences:", err)
			break
		}

		data := doc.Data()
		email := data["email"].(string)
		shouldSend := data[meal].(bool)

		if shouldSend {
			recipe, err := getRandomRecipe()
			if err != nil {
				log.Println("Failed to get recipe for", email, ":", err)
				continue
			}
			body := generateEmailBody(recipe)
			sendEmail([]string{email}, body)
		}
	}
}

// These are related to the scheduling of daily tasks
func scheduleDailyTasks() {
	go scheduleTask("breakfast", "07:30")
	go scheduleTask("lunch", "12:30")
	go scheduleTask("dinner", "18:30")
}

func scheduleTask(meal string, targetTime string) {
	for {
		next, _ := nextOccurrenceIST(targetTime)
		duration := time.Until(next)
		log.Printf("Next %s task scheduled at: %v (in %v)", meal, next, duration)

		time.Sleep(duration)

		checkPreferencesAndSend(meal)

		// Wait 1 minute to prevent back-to-back triggering in edge cases
		time.Sleep(time.Minute)
	}
}

func nextOccurrenceIST(timeStr string) (time.Time, error) {
	// Parse desired hour and minute (e.g. "07:30") in UTC just for extracting hour/min
	targetHourMin, err := time.Parse("15:04", timeStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid time format: %w", err)
	}

	// Get IST location
	istLoc, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to load IST location: %w", err)
	}

	now := time.Now().In(istLoc)

	// Create today's target time in IST
	next := time.Date(
		now.Year(), now.Month(), now.Day(),
		targetHourMin.Hour(), targetHourMin.Minute(), 0, 0,
		istLoc,
	)

	// If it's already past that time today, schedule for tomorrow
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}

	return next, nil
}
