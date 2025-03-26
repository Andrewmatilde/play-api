package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"my-embedded-api/apiv1"
	"my-embedded-api/meta"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func setupTestRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	db := setupTestDB(t)

	// Register routes
	routerObj := NewRouter[apiv1.User](router, db)
	routerObj.Register("/api/v1/users")

	return router, db
}

func TestRouter_CRUD(t *testing.T) {
	r, db := setupTestRouter(t)
	defer cleanupTestDB(t, db)

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

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/users", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var created apiv1.User
	err = json.NewDecoder(w.Body).Decode(&created)
	assert.NoError(t, err)
	assert.NotEmpty(t, created.ID)

	// Test user retrieval
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", fmt.Sprintf("/api/v1/users/%d", created.ID), nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var found apiv1.User
	err = json.NewDecoder(w.Body).Decode(&found)
	assert.NoError(t, err)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, user.Username, found.Username)
	assert.Equal(t, user.Email, found.Email)

	// Test user update
	found.Email = "updated@example.com"
	body, err = json.Marshal(found)
	assert.NoError(t, err)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", fmt.Sprintf("/api/v1/users/%d", found.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test user deletion
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", fmt.Sprintf("/api/v1/users/%d", found.ID), nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)

	// Verify deletion
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", fmt.Sprintf("/api/v1/users/%d", found.ID), nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRouter_Create(t *testing.T) {
	router, db := setupTestRouter(t)

	// Create test user
	user := &apiv1.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}
	body, _ := json.Marshal(user)
	req := httptest.NewRequest("POST", "/api/v1/users", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	// Verify user was created
	var found apiv1.User
	err := db.First(&found).Error
	assert.NoError(t, err)
	assert.Equal(t, user.Username, found.Username)
	assert.Equal(t, user.Email, found.Email)
}

func TestRouter_Get(t *testing.T) {
	router, db := setupTestRouter(t)

	// Create test user
	user := &apiv1.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}
	err := db.Create(user).Error
	assert.NoError(t, err)

	// Test getting user
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/users/%d", user.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response apiv1.User
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, user.Username, response.Username)
	assert.Equal(t, user.Email, response.Email)
}

func TestRouter_Update(t *testing.T) {
	router, db := setupTestRouter(t)

	// Create test user
	user := &apiv1.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}
	err := db.Create(user).Error
	assert.NoError(t, err)

	// Update user
	user.Email = "updated@example.com"
	body, _ := json.Marshal(user)
	req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/users/%d", user.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify update
	var found apiv1.User
	err = db.First(&found, user.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "updated@example.com", found.Email)
}

func TestRouter_Delete(t *testing.T) {
	router, db := setupTestRouter(t)

	// Create test user
	user := &apiv1.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}
	err := db.Create(user).Error
	assert.NoError(t, err)

	// Delete user
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/users/%d", user.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	// Verify deletion
	var found apiv1.User
	err = db.First(&found, user.ID).Error
	assert.Error(t, err)
}

func TestRouter_List(t *testing.T) {
	router, db := setupTestRouter(t)

	// Create test users
	users := []apiv1.User{
		{Username: "user1", Email: "user1@example.com", Password: "pass1"},
		{Username: "user2", Email: "user2@example.com", Password: "pass2"},
		{Username: "user3", Email: "user3@example.com", Password: "pass3"},
	}

	for _, user := range users {
		err := db.Create(&user).Error
		assert.NoError(t, err)
	}

	// Test listing users
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []apiv1.User
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response, 3)
}

func TestRouter_Validation(t *testing.T) {
	router, _ := setupTestRouter(t)

	// Test invalid user (missing required fields)
	user := &apiv1.User{
		Email: "test@example.com",
		// Missing username and password
	}
	body, _ := json.Marshal(user)
	req := httptest.NewRequest("POST", "/api/v1/users", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRouter_Pagination(t *testing.T) {
	router, db := setupTestRouter(t)

	// Create test users
	users := []apiv1.User{
		{Username: "user1", Email: "user1@example.com", Password: "pass1"},
		{Username: "user2", Email: "user2@example.com", Password: "pass2"},
		{Username: "user3", Email: "user3@example.com", Password: "pass3"},
	}

	for _, user := range users {
		err := db.Create(&user).Error
		assert.NoError(t, err)
	}

	// Test pagination
	req := httptest.NewRequest("GET", "/api/v1/users?page=1&size=2", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []apiv1.User
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response, 2)
}

func TestRouter_Concurrent(t *testing.T) {
	router, db := setupTestRouter(t)

	// Create test users
	users := []apiv1.User{
		{Username: "user1", Email: "user1@example.com", Password: "pass1"},
		{Username: "user2", Email: "user2@example.com", Password: "pass2"},
		{Username: "user3", Email: "user3@example.com", Password: "pass3"},
	}

	// Create users sequentially first
	for _, user := range users {
		user.Kind = "User"
		user.APIVersion = "v1"
		err := db.Create(&user).Error
		assert.NoError(t, err)
	}

	// Test concurrent requests
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/api/v1/users", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
