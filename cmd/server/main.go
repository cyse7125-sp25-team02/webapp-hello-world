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

	db, err := database.NewPostgresConnection(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create a new ServeMux
	mux := http.NewServeMux()

	// Register handlers
	healthHandler := handler.NewHealthHandler(db)
	mux.Handle("/healthz", healthHandler)

	userHandler := handler.NewUserHandler(db)
	mux.Handle("/v1/user", userHandler)

	instructorHandler := handler.NewInstructorHandler(db)
	mux.Handle("/v1/instructor", instructorHandler)

	log.Println("Server starting on :3000")
	if err := http.ListenAndServe(":3000", mux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
