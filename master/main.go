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

	// Load any saved databases from disk
	loadAllDBs()
	log.Println("[MASTER] Databases loaded from disk")

	mux := http.NewServeMux()
	setupRoutes(mux)

	log.Printf("[MASTER] Running on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
