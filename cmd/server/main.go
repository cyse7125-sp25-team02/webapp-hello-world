// cmd/server/main.go
package main

import (
	"log"
	"net/http"
	"webapp-hello-world/internal/config"
	"webapp-hello-world/internal/database"
	"webapp-hello-world/internal/handler"
)

func main() {
	cfg := config.NewConfig()

	db, err := database.NewMySQLConnection(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	healthHandler := handler.NewHealthHandler(db)
	http.Handle("/healthz", healthHandler)

	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
