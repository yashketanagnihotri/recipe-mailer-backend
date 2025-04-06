package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

type IngredientsRequest struct {
	Ingredients []string `json:"ingredients"`
}

func generateRecipesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Decode the input ingredients
	var req IngredientsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if len(req.Ingredients) == 0 || len(req.Ingredients) > 10 {
		http.Error(w, "Please provide between 1 and 10 ingredients", http.StatusBadRequest)
		return
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		http.Error(w, "OpenAI API key not configured", http.StatusInternalServerError)
		return
	}

	// Construct prompt
	prompt := "Generate 5 healthy recipes using ONLY the following ingredients: " +
		strings.Join(req.Ingredients, ", ") + ". " +
		"Each recipe should be returned as a JSON object with fields: title, description, ingredients (as an array of strings), and instructions (as an array of strings). Return an array of 5 such recipes in JSON."

	// Prepare OpenAI API request
	payload := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.7,
	}
	payloadBytes, _ := json.Marshal(payload)

	reqBody := bytes.NewBuffer(payloadBytes)
	openaiReq, _ := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", reqBody)
	openaiReq.Header.Set("Authorization", "Bearer "+apiKey)
	openaiReq.Header.Set("Content-Type", "application/json")

	// Make the request
	resp, err := http.DefaultClient.Do(openaiReq)
	if err != nil {
		http.Error(w, "Failed to call OpenAI API", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Parse OpenAI response
	body, _ := ioutil.ReadAll(resp.Body)
	var aiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &aiResp); err != nil || len(aiResp.Choices) == 0 {
		http.Error(w, "Failed to parse OpenAI response", http.StatusInternalServerError)
		return
	}

	// Extract JSON from the message content
	var recipes []Recipe
	decoded := aiResp.Choices[0].Message.Content
	decoder := json.NewDecoder(strings.NewReader(decoded))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&recipes); err != nil {
		http.Error(w, "Failed to decode recipes. Check OpenAI output format.", http.StatusInternalServerError)
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recipes)
}


// get a list of all recipes from Firestore
func getAllRecipesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	recipes, err := getRecipesFromFirestore()
	if err != nil {
		http.Error(w, "Failed to fetch recipes", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recipes)
}

