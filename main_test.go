package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"my-embedded-api/apiv1"
	"my-embedded-api/internal"
	"my-embedded-api/meta"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Server represents the test server
type Server struct {
	server *httptest.Server
	db     *gorm.DB
}

func (s *Server) URL() string {
	return s.server.URL
}

func (s *Server) Close() {
	s.server.Close()
	sqlDB, err := s.db.DB()
	if err != nil {
		return
	}
	sqlDB.Close()
}

func setupTestServer(t *testing.T) (*Server, *gorm.DB) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Create a temporary directory for the test database
	tmpDir, err := os.MkdirTemp("", "testdb")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Migrate the database schema
	err = db.AutoMigrate(&apiv1.User{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Verify that the users table was created
	var tables []string
	err = db.Raw("SELECT name FROM sqlite_master WHERE type='table'").Scan(&tables).Error
	if err != nil {
		t.Fatalf("Failed to verify tables: %v", err)
	}

	if !contains(tables, "users") {
		t.Fatal("Required table 'users' was not created")
	}

	// Register routes
	routerObj := internal.NewRouter[apiv1.User](router, db)
	routerObj.Register("/api/v1/users")

	server := &Server{
		server: httptest.NewServer(router),
		db:     db,
	}
	return server, db
}

func cleanupTestServer(t *testing.T, server *Server, db *gorm.DB) {
	server.Close()
	sqlDB, err := db.DB()
	if err != nil {
		t.Logf("Failed to get underlying *sql.DB: %v", err)
		return
	}
	if err := sqlDB.Close(); err != nil {
		t.Logf("Failed to close database connection: %v", err)
	}
}

func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

func TestUserAPI(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()

	// Test user creation
	user := apiv1.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}

	body, err := json.Marshal(user)
	assert.NoError(t, err)

	resp, err := http.Post(server.URL()+"/api/v1/users", "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var created apiv1.User
	err = json.NewDecoder(resp.Body).Decode(&created)
	assert.NoError(t, err)
	assert.NotEmpty(t, created.ID)
	resp.Body.Close()

	// Test user retrieval
	resp, err = http.Get(fmt.Sprintf("%s/api/v1/users/%d", server.URL(), created.ID))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var found apiv1.User
	err = json.NewDecoder(resp.Body).Decode(&found)
	assert.NoError(t, err)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, user.Username, found.Username)
	assert.Equal(t, user.Email, found.Email)
	resp.Body.Close()

	// Test user update
	found.Email = "updated@example.com"
	body, err = json.Marshal(found)
	assert.NoError(t, err)

	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/api/v1/users/%d", server.URL(), found.ID), bytes.NewBuffer(body))
	assert.NoError(t, err)
	resp, err = http.DefaultClient.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Test user deletion
	req, err = http.NewRequest("DELETE", fmt.Sprintf("%s/api/v1/users/%d", server.URL(), found.ID), nil)
	assert.NoError(t, err)
	resp, err = http.DefaultClient.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Verify deletion
	resp, err = http.Get(fmt.Sprintf("%s/api/v1/users/%d", server.URL(), found.ID))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()
}

func TestServer_Startup(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL() + "/api/v1/users")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

func TestServer_UserOperations(t *testing.T) {
	server, db := setupTestServer(t)
	defer cleanupTestServer(t, server, db)

	// Test user creation
	user := apiv1.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
		BaseResource: meta.BaseResource{
			TypeMeta: meta.TypeMeta{
				Kind:       "User",
				APIVersion: "v1",
			},
		},
	}

	body, err := json.Marshal(user)
	assert.NoError(t, err)

	resp, err := http.Post(server.URL()+"/api/v1/users", "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var created apiv1.User
	err = json.NewDecoder(resp.Body).Decode(&created)
	assert.NoError(t, err)
	assert.NotEmpty(t, created.ID)

	// Test user retrieval
	resp, err = http.Get(server.URL() + fmt.Sprintf("/api/v1/users/%d", created.ID))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var found apiv1.User
	err = json.NewDecoder(resp.Body).Decode(&found)
	assert.NoError(t, err)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, user.Username, found.Username)
	assert.Equal(t, user.Email, found.Email)

	// Test user update
	found.Email = "updated@example.com"
	body, err = json.Marshal(found)
	assert.NoError(t, err)

	req, err := http.NewRequest("PUT", server.URL()+fmt.Sprintf("/api/v1/users/%d", found.ID), bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err = http.DefaultClient.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Test user deletion
	req, err = http.NewRequest("DELETE", server.URL()+fmt.Sprintf("/api/v1/users/%d", found.ID), nil)
	assert.NoError(t, err)

	resp, err = http.DefaultClient.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify deletion
	resp, err = http.Get(server.URL() + fmt.Sprintf("/api/v1/users/%d", found.ID))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestServer_ConcurrentRequests(t *testing.T) {
	server, db := setupTestServer(t)
	defer cleanupTestServer(t, server, db)

	// Create a test user
	user := apiv1.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
		BaseResource: meta.BaseResource{
			TypeMeta: meta.TypeMeta{
				Kind:       "User",
				APIVersion: "v1",
			},
		},
	}

	body, err := json.Marshal(user)
	assert.NoError(t, err)

	resp, err := http.Post(server.URL()+"/api/v1/users", "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var created apiv1.User
	err = json.NewDecoder(resp.Body).Decode(&created)
	assert.NoError(t, err)
	assert.NotEmpty(t, created.ID)

	// Test concurrent requests
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get(server.URL() + fmt.Sprintf("/api/v1/users/%d", created.ID))
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			resp.Body.Close()
		}()
	}
	wg.Wait()
}

func TestServer_ErrorHandling(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()

	// Test invalid user creation (missing required fields)
	user := apiv1.User{
		// Missing required fields
		Email: "invalid-email", // Invalid email format
	}

	body, err := json.Marshal(user)
	assert.NoError(t, err)

	resp, err := http.Post(server.URL()+"/api/v1/users", "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	resp.Body.Close()

	// Test invalid user ID
	resp, err = http.Get(server.URL() + "/api/v1/users/invalid")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	resp.Body.Close()
}

func TestServer_GracefulShutdown(t *testing.T) {
	// Create temporary database file
	tmpDB := "test.db"
	defer os.Remove(tmpDB)

	// Initialize database
	db, err := gorm.Open(sqlite.Open(tmpDB), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&apiv1.User{})
	assert.NoError(t, err)

	// Initialize router
	router := gin.Default()
	internal.RegisterResource[apiv1.User](router, db, "/api/v1/users")

	// Create server
	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Errorf("Failed to start server: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Test server is running
	resp, err := http.Get("http://localhost:8080/api/v1/users")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Shutdown server
	if err := srv.Shutdown(nil); err != nil {
		t.Errorf("Failed to shutdown server: %v", err)
	}

	// Verify server is down
	_, err = http.Get("http://localhost:8080/api/v1/users")
	assert.Error(t, err)
}
