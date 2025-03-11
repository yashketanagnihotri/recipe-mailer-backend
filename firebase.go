package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/option"
)



type Recipe struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Ingredients []string `json:"ingredients"`
	Instructions []string `json:"instructions"`
}
// Firestore client
var firestoreClient *firestore.Client

// Initialize Firebase Firestore
func initFirebase() {
	ctx := context.Background()
	sa := option.WithCredentialsFile("serviceAccountKey.json")

	client, err := firestore.NewClient(ctx, "recipes-app-e2ba2", sa)
	if err != nil {
		log.Fatalf("Failed to initialize Firestore: %v", err)
	}

	firestoreClient = client
	fmt.Println("Connected to Firestore")
}

// Firestore client initialization
func initFirestore() (*firestore.Client, error) {
	ctx := context.Background()
	sa := option.WithCredentialsFile("serviceAccountKey.json") // Update this with your actual credentials file
	client, err := firestore.NewClient(ctx, "recipes-app-e2ba2", sa)
	if err != nil {
		return nil, err
	}
	return client, nil
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

	client, err := initFirestore()
	if err != nil {
		http.Error(w, "Failed to initialize Firestore", http.StatusInternalServerError)
		log.Println("Firestore initialization error:", err)
		return
	}
	defer client.Close()

	ctx := context.Background()
	batch := client.Batch()
	collectionRef := client.Collection("recipes")

	for _, recipe := range recipes {
		// Ensure that each recipe is unique and not a reference to the same object
		newRecipe := Recipe{
			Title:        recipe.Title,
			Description:  recipe.Description,
			Ingredients:  append([]string{}, recipe.Ingredients...), // Copy slice to avoid reference issues
			Instructions: append([]string{}, recipe.Instructions...),
		}

		docRef := collectionRef.NewDoc()
		batch.Set(docRef, newRecipe)
	}

	_, err = batch.Commit(ctx)
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
