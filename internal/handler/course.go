package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"webapp-hello-world/internal/model"

	"github.com/google/uuid"
)

type CourseHandler struct {
	db *sql.DB
}

func NewCourseHandler(db *sql.DB) *CourseHandler {
	return &CourseHandler{db: db}
}

func (h *CourseHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Course handler hit:", r.Method, r.URL.Path)
	w.Header().Set("Content-Type", "application/json")

	// Allow requests without authentication
	if r.Method == http.MethodGet {
		h.GetCourseByID(w, r)
		return
	}

	// Perform basic authentication
	username, password, hasAuth := r.BasicAuth()
	if !hasAuth {
		w.Header().Set("WWW-Authenticate", `Basic realm="Course Authentication Required"`)
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Authentication required"})
		return
	}

	// Authenticate the user
	user, err := model.AuthenticateUser(h.db, username, password)
	if err != nil {
		w.Header().Set("WWW-Authenticate", `Basic realm="Invalid Credentials"`)
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid username or password"})
		return
	}

	// Check if the user has admin privileges
	if user.Role != "admin" {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Insufficient permissions"})
		return
	}

	switch r.Method {
	case http.MethodPost:
		h.createCourse(w, r, user)
	case http.MethodDelete:
		h.DeleteCourseByID(w, r)
	case http.MethodPatch:
		h.PatchCourse(w, r, user)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
	}
}

func (h *CourseHandler) createCourse(w http.ResponseWriter, r *http.Request, user *model.User) {
	var req model.CreateCourseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	// Validate the request data
	if err := req.Validate(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Create the course in the database
	course, err := model.CreateCourse(h.db, req, user.ID)
	if err != nil {
		if strings.Contains(err.Error(), "foreign key constraint") {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid instructor_id"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create course"})
		return
	}

	// Return the created course
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(course)
}

func (h *CourseHandler) GetCourseByID(w http.ResponseWriter, r *http.Request) {
	// Extract the course ID from query parameters
	courseIDStr := r.URL.Query().Get("id")
	if courseIDStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Course ID is required"})
		return
	}

	// Parse the course ID into a UUID
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid course ID format"})
		return
	}

	// Retrieve the course from the database
	course, err := model.GetCourseByID(h.db, courseID)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Course not found"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to retrieve course"})
		return
	}

	// Return the course details as JSON
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(course)
}

func (h *CourseHandler) DeleteCourseByID(w http.ResponseWriter, r *http.Request) {
	// Extract course ID from query parameters
	courseIDStr := r.URL.Query().Get("id")
	if courseIDStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Course ID is required"})
		return
	}

	// Parse the course ID as a UUID
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid course ID format"})
		return
	}

	// Delete the course from the database
	err = model.DeleteCourseByID(h.db, courseID)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Course not found"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to delete course"})
		return
	}

	// Return success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Course deleted successfully"})
}

// UpdateCourse handles the PATCH request to update a course.
func (h *CourseHandler) PatchCourse(w http.ResponseWriter, r *http.Request, user *model.User) {
	// Extract course ID from query parameters
	courseIDStr := r.URL.Query().Get("id")
	if courseIDStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Course ID is required"})
		return
	}

	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid course ID format"})
		return
	}

	// Parse request body
	var req model.UpdateCourseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Update the course
	updatedCourse, err := model.UpdateCourse(h.db, courseID, req, user.ID)
	if err != nil {
		if strings.Contains(err.Error(), "foreign key constraint") {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid user_id or instructor_id"})
			return
		}
		if err.Error() == "course not found" {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Course not found"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to update course"})
		return
	}

	// Return the updated course
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updatedCourse)
}
