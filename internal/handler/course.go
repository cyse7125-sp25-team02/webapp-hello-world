// internal/handler/course.go
package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"webapp-hello-world/internal/config"
	"webapp-hello-world/internal/model"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"

	"github.com/google/uuid"
)

type CourseHandler struct {
	db         *sql.DB
	gcsClient  *storage.Client
	bucketName string
}

func NewCourseHandler(db *sql.DB, cfg *config.Config) *CourseHandler {
	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(cfg.GCSCredentialsFile))
	if err != nil {
		log.Fatalf("Failed to create GCS client: %v", err)
	}
	return &CourseHandler{
		db:         db,
		gcsClient:  client,
		bucketName: cfg.GCSBucketName,
	}
}

func (h *CourseHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Course handler hit:", r.Method, r.URL.Path)
	w.Header().Set("Content-Type", "application/json")

	// Allow requests without authentication
	if r.Method == http.MethodGet && !strings.Contains(r.URL.Path, "/trace") {
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
	case http.MethodGet:
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/trace") {
			courseIDStr := r.PathValue("course_id")
			if courseIDStr == "" {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "Missing course_id in URL"})
				return
			}
			h.GetTracesByCourseID(w, r, courseIDStr)
			return
		}
	case http.MethodPost:
		// Handle trace upload
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/trace") {
			// Get course_id from path
			courseIDStr := r.PathValue("course_id")
			if courseIDStr == "" {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "Missing course_id in URL"})
				return
			}
			h.handleTraceUpload(w, r, user, courseIDStr)
			return
		}
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

func (h *CourseHandler) handleTraceUpload(w http.ResponseWriter, r *http.Request, user *model.User, courseIDStr string) {
	// Parse the course ID
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid course_id format"})
		return
	}

	// Parse multipart form (max 10MB)
	err = r.ParseMultipartForm(10 << 20)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to parse multipart form"})
		return
	}

	// Get the PDF file
	file, handler, err := r.FormFile("file")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "File is required"})
		return
	}
	defer file.Close()

	// Get form fields
	fileName := r.FormValue("file_name")
	if fileName == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "file_name is required"})
		return
	}

	instructorIDStr := r.FormValue("instructor_id")
	if instructorIDStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "instructor_id is required"})
		return
	}
	instructorID, err := uuid.Parse(instructorIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid instructor_id format"})
		return
	}

	var vectorID *string
	if vid := r.FormValue("vector_id"); vid != "" {
		vectorID = &vid
	}

	// Generate a unique filename for GCS to avoid conflicts
	uniqueName := fmt.Sprintf("%s-%s", uuid.New().String(), handler.Filename)
	bucketURL, err := h.uploadToGCS(file, uniqueName)
	status := "uploaded"
	if err != nil {
		status = "failed"
		bucketURL = "" // Since bucket_url is NOT NULL, use empty string
		err = model.InsertTrace(h.db, user.ID, instructorID, status, courseID, vectorID, fileName, bucketURL)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to insert trace record"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to upload file to GCS"})
		return
	}

	// Insert trace record on successful upload
	err = model.InsertTrace(h.db, user.ID, instructorID, status, courseID, vectorID, fileName, bucketURL)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to insert trace record"})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "File uploaded successfully", "bucket_url": bucketURL})
}

func (h *CourseHandler) uploadToGCS(file io.Reader, filename string) (string, error) {
	ctx := context.Background()
	bucket := h.gcsClient.Bucket(h.bucketName)
	object := bucket.Object(filename)

	w := object.NewWriter(ctx)
	if _, err := io.Copy(w, file); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}

	attrs, err := object.Attrs(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", h.bucketName, attrs.Name), nil
}

func (h *CourseHandler) GetTracesByCourseID(w http.ResponseWriter, r *http.Request, courseIDStr string) {
	// Parse the course ID
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid course_id format"})
		return
	}

	// Get traces from the database
	traces, err := model.GetTracesByCourseID(h.db, courseID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to retrieve traces"})
		return
	}

	// Return the traces as JSON
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"data": traces})
}

// internal/handler/course.go

func (h *CourseHandler) GetTraceByID(w http.ResponseWriter, r *http.Request, courseIDStr, traceIDStr string) {
	// Parse the course ID
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid course_id format"})
		return
	}

	// Parse the trace ID
	traceID, err := uuid.Parse(traceIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid trace_id format"})
		return
	}

	// Get trace from the database
	trace, err := model.GetTraceByID(h.db, courseID, traceID)
	if err != nil {
		if err.Error() == "trace not found" {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Trace not found"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to retrieve trace"})
		return
	}

	// Return the trace as JSON
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(trace)
}
