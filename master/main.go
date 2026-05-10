package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Connect to MySQL
	if err := connectMySQL(); err != nil {
		log.Fatalf("[MASTER] MySQL connection failed: %v", err)
	}
	log.Println("[MASTER] Connected to MySQL")

	mux := http.NewServeMux()
	setupRoutes(mux)

	log.Printf("[MASTER] Running on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
