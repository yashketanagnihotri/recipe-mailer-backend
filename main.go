package main

import (
	"fmt"
	"log"
	"net/http"
)



func main() {
	initFirebase()
	http.HandleFunc("/send-email", sendEmailHandler)
	http.HandleFunc("/add-recipe", addRecipesHandler) 

	fmt.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
