package apiv1

import (
	"errors"
	"regexp"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"my-embedded-api/meta"
)

// User represents a user in the system
type User struct {
	meta.BaseResource `json:",inline"`

	// Username is the unique username for the user
	Username string `gorm:"size:100;not null;unique" json:"username" binding:"required"`

	// Email is the user's email address
	Email string `gorm:"size:100;not null;unique" json:"email" binding:"required,email"`

	// Password is the hashed password (not exposed in JSON)
	Password string `gorm:"size:100;not null" json:"password" binding:"required"`

	// FullName is the user's full name
	FullName string `gorm:"size:100" json:"fullName,omitempty"`

	// IsActive indicates whether the user account is active
	IsActive bool `gorm:"default:true" json:"isActive"`
}

// TableName specifies the table name for GORM
func (User) TableName() string {
	return "users"
}

// isHashedPassword checks if a password is already hashed
func isHashedPassword(password string) bool {
	return strings.HasPrefix(password, "$2a$") || strings.HasPrefix(password, "$2b$")
}

// Validate implements ResourceValidator interface
func (u *User) Validate() error {
	// First validate base resource
	if err := u.BaseResource.Validate(); err != nil {
		return err
	}

	// Validate username
	if u.Username == "" {
		return errors.New("username is required")
	}
	if len(u.Username) < 3 {
		return errors.New("username must be at least 3 characters long")
	}

	// Validate email
	if u.Email == "" {
		return errors.New("email is required")
	}

	// Validate email format
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(u.Email) {
		return errors.New("invalid email format")
	}

	// Validate password (only if it's not empty)
	if u.Password == "" {
		return errors.New("password is required")
	}

	return nil
}

// SetPassword hashes and sets the user's password
func (u *User) SetPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

// CheckPassword verifies if the provided password matches the user's password
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// BeforeCreate is a GORM hook that runs before creating a user
func (u *User) BeforeCreate(tx *gorm.DB) error {
	// Set TypeMeta fields
	u.Kind = "User"
	u.APIVersion = "v1"

	// Set initial status
	u.SetStatus("Active", "User created successfully", "Created")

	// Hash password if not already hashed
	if !strings.HasPrefix(u.Password, "$2a$") {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		u.Password = string(hashedPassword)
	}

	// Call parent BeforeCreate
	return u.BaseResource.BeforeCreate(tx)
}

// BeforeUpdate is a GORM hook that runs before updating a user
func (u *User) BeforeUpdate(tx *gorm.DB) error {
	// Set TypeMeta fields
	u.Kind = "User"
	u.APIVersion = "v1"

	// Update status
	u.SetStatus("Active", "User updated successfully", "Updated")

	// Hash password if not already hashed
	if !strings.HasPrefix(u.Password, "$2a$") {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		u.Password = string(hashedPassword)
	}

	// Call parent BeforeUpdate
	return u.BaseResource.BeforeUpdate(tx)
}

// BeforeDelete is a GORM hook that runs before deleting a user
func (u *User) BeforeDelete(tx *gorm.DB) error {
	// Update status
	u.SetStatus("Deleted", "User deleted successfully", "Deleted")

	// Call parent BeforeDelete
	return u.BaseResource.BeforeDelete(tx)
}

// ComparePassword compares the given password with the user's hashed password
func (u *User) ComparePassword(password string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
}
