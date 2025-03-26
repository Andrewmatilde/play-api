package internal

import (
	"os"
	"path/filepath"
	"testing"

	"my-embedded-api/apiv1"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates a new in-memory SQLite database for testing
func setupTestDB(t *testing.T) *gorm.DB {
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
	err = db.AutoMigrate(
		&apiv1.User{},
		&TestModel{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Verify that the tables were created
	var tables []string
	err = db.Raw("SELECT name FROM sqlite_master WHERE type='table'").Scan(&tables).Error
	if err != nil {
		t.Fatalf("Failed to verify tables: %v", err)
	}

	requiredTables := []string{"users", "test_models"}
	for _, table := range requiredTables {
		if !contains(tables, table) {
			t.Fatalf("Required table %s was not created", table)
		}
	}

	return db
}

// cleanupTestDB closes the database connection
func cleanupTestDB(t *testing.T, db *gorm.DB) {
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
