package internal

import "gorm.io/gorm"

// TestModel is a test model for testing DAO operations
type TestModel struct {
	gorm.Model
	Name string
}
