package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	loadAllDBs()
	log.Println("[SLAVE-GO] Databases loaded from disk")

	mux := http.NewServeMux()
	setupRoutes(mux)

	log.Printf("[SLAVE-GO] Running on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
