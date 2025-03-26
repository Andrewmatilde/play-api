package meta

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestResource is a test implementation of BaseResource
type TestResource struct {
	BaseResource
	Name string `json:"name" gorm:"not null"`
}

func (r *TestResource) BeforeCreate(tx *gorm.DB) error {
	r.Kind = "TestResource"
	r.APIVersion = "v1"
	return r.BaseResource.BeforeCreate(tx)
}

func (r *TestResource) BeforeUpdate(tx *gorm.DB) error {
	r.Kind = "TestResource"
	r.APIVersion = "v1"
	return r.BaseResource.BeforeUpdate(tx)
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// Migrate test resource table
	err = db.AutoMigrate(&TestResource{})
	assert.NoError(t, err)

	// Verify table was created
	var tables []string
	err = db.Raw("SELECT name FROM sqlite_master WHERE type='table'").Scan(&tables).Error
	assert.NoError(t, err)
	assert.Contains(t, tables, "test_resources")

	return db
}

// cleanupTestDB closes the database connection
func cleanupTestDB(t *testing.T, db *gorm.DB) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	err = sqlDB.Close()
	assert.NoError(t, err)
}

func TestBaseResource_Creation(t *testing.T) {
	db := setupTestDB(t)

	// Test resource creation
	resource := &TestResource{
		Name: "test",
	}

	err := db.Create(resource).Error
	assert.NoError(t, err)

	// Verify fields
	assert.NotEmpty(t, resource.ID)
	assert.NotEmpty(t, resource.CreatedAt)
	assert.NotEmpty(t, resource.UpdatedAt)
	assert.Equal(t, "test", resource.Name)
	assert.Equal(t, "TestResource", resource.Kind)
	assert.Equal(t, "v1", resource.APIVersion)
	assert.NotEmpty(t, resource.UID)
	assert.Equal(t, 1, resource.ResourceVersion)
}

func TestBaseResource_Status(t *testing.T) {
	resource := &TestResource{}

	// Test status setting
	resource.SetStatus("Active", "Resource is active", "Created")
	assert.Equal(t, "Active", resource.Status.Phase)
	assert.Equal(t, "Resource is active", resource.Status.Message)
	assert.Equal(t, "Created", resource.Status.Reason)
	assert.NotEmpty(t, resource.Status.LastTransitionTime)
}

func TestBaseResource_Validation(t *testing.T) {
	resource := &TestResource{}

	// Test validation without required fields
	err := resource.Validate()
	assert.Error(t, err)

	// Test validation with required fields
	resource.Kind = "TestResource"
	resource.APIVersion = "v1"
	err = resource.Validate()
	assert.NoError(t, err)
}

func TestBaseResource_Events(t *testing.T) {
	db := setupTestDB(t)

	resource := &TestResource{
		Name: "test",
	}

	// Test BeforeCreate
	err := db.Create(resource).Error
	assert.NoError(t, err)
	assert.Equal(t, "Pending", resource.Status.Phase)

	// Test BeforeUpdate
	resource.Name = "updated"
	err = db.Save(resource).Error
	assert.NoError(t, err)
	assert.Equal(t, "Pending", resource.Status.Phase)
	assert.Equal(t, 2, resource.ResourceVersion)

	// Test BeforeDelete
	err = db.Delete(resource).Error
	assert.NoError(t, err)
}

func TestBaseResource_Metadata(t *testing.T) {
	resource := &TestResource{}

	// Test metadata operations
	resource.SetMetadata("key1", "value1")
	value, exists := resource.GetMetadata("key1")
	assert.True(t, exists)
	assert.Equal(t, "value1", value)

	// Test non-existent key
	_, exists = resource.GetMetadata("nonexistent")
	assert.False(t, exists)

	// Test metadata deletion
	resource.DeleteMetadata("key1")
	_, exists = resource.GetMetadata("key1")
	assert.False(t, exists)
}

func TestBaseResource_Timestamps(t *testing.T) {
	db := setupTestDB(t)

	resource := &TestResource{
		Name: "test",
	}

	// Create resource
	err := db.Create(resource).Error
	assert.NoError(t, err)

	// Verify timestamps
	assert.NotEmpty(t, resource.CreatedAt)
	assert.NotEmpty(t, resource.UpdatedAt)

	// Update resource
	time.Sleep(time.Millisecond) // Ensure time difference
	resource.Name = "updated"
	err = db.Save(resource).Error
	assert.NoError(t, err)

	// Verify UpdatedAt changed
	assert.NotEqual(t, resource.CreatedAt, resource.UpdatedAt)
}
