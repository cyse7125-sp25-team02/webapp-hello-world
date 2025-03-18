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

	courseHandler := handler.NewCourseHandler(db, cfg)
	mux.Handle("POST /v1/course", http.HandlerFunc(courseHandler.CreateCourse))
	mux.Handle("GET /v1/course/{course_id}", http.HandlerFunc(courseHandler.GetCourseByID))
	mux.Handle("PATCH /v1/course/{course_id}", http.HandlerFunc(courseHandler.PatchCourse))
	mux.Handle("DELETE /v1/course/{course_id}", http.HandlerFunc(courseHandler.DeleteCourseByID))
	mux.Handle("GET /v1/course/{course_id}/trace", http.HandlerFunc(courseHandler.GetTracesByCourseID))
	mux.Handle("POST /v1/course/{course_id}/trace", http.HandlerFunc(courseHandler.HandleTraceUpload))
	mux.Handle("GET /v1/course/{course_id}/trace/{trace_id}", http.HandlerFunc(courseHandler.GetTraceByID))
	mux.Handle("DELETE /v1/course/{course_id}/trace/{trace_id}", http.HandlerFunc(courseHandler.DeleteTraceByID))

	log.Println("Server starting on :3000")
	if err := http.ListenAndServe(":3000", mux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
