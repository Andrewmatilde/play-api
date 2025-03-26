package apiv1

import (
	"testing"
	"time"

	"my-embedded-api/meta"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// Migrate user table
	err = db.AutoMigrate(&User{})
	assert.NoError(t, err)

	// Verify table was created
	var tables []string
	err = db.Raw("SELECT name FROM sqlite_master WHERE type='table'").Scan(&tables).Error
	assert.NoError(t, err)
	assert.Contains(t, tables, "users")

	return db
}

// cleanupTestDB closes the database connection
func cleanupTestDB(t *testing.T, db *gorm.DB) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	err = sqlDB.Close()
	assert.NoError(t, err)
}

func TestUser_Creation(t *testing.T) {
	db := setupTestDB(t)

	// Test user creation
	user := &User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}
	user.BaseResource.TypeMeta.Kind = "User"
	user.BaseResource.TypeMeta.APIVersion = "v1"

	err := db.Create(user).Error
	assert.NoError(t, err)

	// Verify fields
	assert.NotEmpty(t, user.ID)
	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, "test@example.com", user.Email)
	assert.NotEmpty(t, user.Password) // Password should be hashed
	assert.Equal(t, "User", user.BaseResource.TypeMeta.Kind)
	assert.Equal(t, "v1", user.BaseResource.TypeMeta.APIVersion)
}

func TestUser_Validate(t *testing.T) {
	tests := []struct {
		name    string
		user    User
		wantErr bool
	}{
		{
			name: "valid user",
			user: User{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
				BaseResource: meta.BaseResource{
					TypeMeta: meta.TypeMeta{
						Kind:       "User",
						APIVersion: "v1",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing username",
			user: User{
				Email:    "test@example.com",
				Password: "password123",
				BaseResource: meta.BaseResource{
					TypeMeta: meta.TypeMeta{
						Kind:       "User",
						APIVersion: "v1",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "missing email",
			user: User{
				Username: "testuser",
				Password: "password123",
				BaseResource: meta.BaseResource{
					TypeMeta: meta.TypeMeta{
						Kind:       "User",
						APIVersion: "v1",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "missing password",
			user: User{
				Username: "testuser",
				Email:    "test@example.com",
				BaseResource: meta.BaseResource{
					TypeMeta: meta.TypeMeta{
						Kind:       "User",
						APIVersion: "v1",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid email format",
			user: User{
				Username: "testuser",
				Email:    "invalid-email",
				Password: "password123",
				BaseResource: meta.BaseResource{
					TypeMeta: meta.TypeMeta{
						Kind:       "User",
						APIVersion: "v1",
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.user.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUser_BeforeCreate(t *testing.T) {
	user := User{
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

	err := user.BeforeCreate(nil)
	assert.NoError(t, err)

	// Verify password is hashed
	assert.NotEqual(t, "password123", user.Password)

	// Verify default status is set
	assert.Equal(t, "Active", user.Status.Phase)
	assert.Equal(t, "User created successfully", user.Status.Message)
	assert.Equal(t, "Created", user.Status.Reason)
	assert.NotEmpty(t, user.Status.LastTransitionTime)
}

func TestUser_BeforeUpdate(t *testing.T) {
	user := User{
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

	err := user.BeforeUpdate(nil)
	assert.NoError(t, err)

	// Verify password is hashed
	assert.NotEqual(t, "password123", user.Password)
}

func TestUser_ComparePassword(t *testing.T) {
	user := User{
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

	// Hash password
	err := user.BeforeCreate(nil)
	assert.NoError(t, err)

	// Test correct password
	err = user.ComparePassword("password123")
	assert.NoError(t, err)

	// Test incorrect password
	err = user.ComparePassword("wrongpassword")
	assert.Error(t, err)
}

func TestUser_Status(t *testing.T) {
	user := User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
		BaseResource: meta.BaseResource{
			TypeMeta: meta.TypeMeta{
				Kind:       "User",
				APIVersion: "v1",
			},
			ObjectMeta: meta.ObjectMeta{
				Status: meta.ResourceStatus{
					Phase:              "Active",
					Message:            "User is active",
					Reason:             "Created",
					LastTransitionTime: time.Now(),
				},
			},
		},
	}

	// Test status fields
	assert.Equal(t, "Active", user.Status.Phase)
	assert.Equal(t, "User is active", user.Status.Message)
	assert.Equal(t, "Created", user.Status.Reason)
	assert.NotEmpty(t, user.Status.LastTransitionTime)
}

func TestUser_Events(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Test user creation
	user := &User{
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

	err := db.Create(user).Error
	assert.NoError(t, err)
	assert.Equal(t, "Active", user.Status.Phase)

	// Test user update
	user.Email = "updated@example.com"
	err = db.Save(user).Error
	assert.NoError(t, err)
	assert.Equal(t, "Active", user.Status.Phase)

	// Test user deletion
	err = db.Delete(user).Error
	assert.NoError(t, err)
	assert.Equal(t, "Deleted", user.Status.Phase)
}

func TestUser_EmailValidation(t *testing.T) {
	// Test valid email
	user := &User{
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
	err := user.Validate()
	assert.NoError(t, err)

	// Test invalid email
	user.Email = "invalid-email"
	err = user.Validate()
	assert.Error(t, err)
}
