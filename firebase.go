package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/firestore"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
)

type Recipe struct {
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Ingredients  []string `json:"ingredients"`
	Instructions []string `json:"instructions"`
}

// Firestore client
var firestoreClient *firestore.Client

// Initialize Firestore with credentials from environment variable
func initFirebase() {
	// Load .env file (for local development)
	_ = godotenv.Load()

	// Get Firebase credentials from environment variable
	creds := os.Getenv("FIREBASE_CREDENTIALS")
	if creds == "" {
		log.Fatal("FIREBASE_CREDENTIALS not set in environment")
	}

	ctx := context.Background()
	credBytes := []byte(creds)
	opt := option.WithCredentialsJSON(credBytes)

	client, err := firestore.NewClient(ctx, "recipes-app-e2ba2", opt)
	if err != nil {
		log.Fatalf("Failed to initialize Firestore: %v", err)
	}

	firestoreClient = client
	fmt.Println("Connected to Firestore")
}

// Handler to add multiple recipes to Firestore
func addRecipesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var recipes []Recipe
	if err := json.NewDecoder(r.Body).Decode(&recipes); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	batch := firestoreClient.Batch()
	collectionRef := firestoreClient.Collection("recipes")

	for _, recipe := range recipes {
		// Ensure that each recipe is unique
		newRecipe := Recipe{
			Title:        recipe.Title,
			Description:  recipe.Description,
			Ingredients:  append([]string{}, recipe.Ingredients...), // Copy slices
			Instructions: append([]string{}, recipe.Instructions...),
		}

		docRef := collectionRef.NewDoc()
		batch.Set(docRef, newRecipe)
	}

	_, err := batch.Commit(ctx)
	if err != nil {
		http.Error(w, "Failed to add recipes", http.StatusInternalServerError)
		log.Println("Firestore batch commit error:", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Recipes added successfully!"))
}

// Fetch recipes from Firestore
func getRecipesFromFirestore() ([]Recipe, error) {
	ctx := context.Background()
	var recipes []Recipe

	iter := firestoreClient.Collection("recipes").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err != nil {
			break
		}
		var recipe Recipe
		if err := doc.DataTo(&recipe); err != nil {
			log.Println("Error parsing document:", err)
			continue
		}
		recipes = append(recipes, recipe)
	}

	if len(recipes) == 0 {
		return nil, fmt.Errorf("no recipes found")
	}

	return recipes, nil
}

