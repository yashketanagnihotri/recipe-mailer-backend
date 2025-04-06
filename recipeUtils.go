package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
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

	// Decode input ingredients
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
		"model": "gpt-4o-mini",
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

	resp, err := http.DefaultClient.Do(openaiReq)
	if err != nil {
		http.Error(w, "Failed to call OpenAI API", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("OpenAI API error %d: %s", resp.StatusCode, string(bodyBytes))
		http.Error(w, "OpenAI API returned an error", http.StatusInternalServerError)
		return
	}

	body, _ := io.ReadAll(resp.Body)
	var aiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &aiResp); err != nil || len(aiResp.Choices) == 0 {
		log.Printf("Failed to parse OpenAI response: %v\nBody: %s", err, string(body))
		http.Error(w, "Failed to parse OpenAI response", http.StatusInternalServerError)
		return
	}

	// Extract and clean the JSON content
	decoded := extractJSON(aiResp.Choices[0].Message.Content)

	// Unmarshal into recipe struct
	var recipes []Recipe
	if err := json.Unmarshal([]byte(decoded), &recipes); err != nil {
		log.Printf("Failed to decode recipes: %v\nContent: %s", err, decoded)
		http.Error(w, "Failed to decode recipes. Check OpenAI output format.", http.StatusInternalServerError)
		return
	}

	// Send final response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recipes)
}

// Regex cleaner to extract JSON inside triple backticks or fallback
func extractJSON(raw string) string {
	re := regexp.MustCompile("(?s)```(?:json)?(.*?)```")
	matches := re.FindStringSubmatch(raw)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}
	return strings.TrimSpace(raw)
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

