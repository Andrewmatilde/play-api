package internal

import (
	"gorm.io/gorm"
)

// DAO provides generic database operations for resources
type DAO[T any] struct {
	db *gorm.DB
}

// NewDAO creates a new DAO instance
func NewDAO[T any](db *gorm.DB) *DAO[T] {
	return &DAO[T]{db: db}
}

// Create creates a new resource
func (d *DAO[T]) Create(resource *T) error {
	return d.db.Create(resource).Error
}

// Get retrieves a resource by ID
func (d *DAO[T]) Get(id uint) (*T, error) {
	var resource T
	err := d.db.First(&resource, id).Error
	if err != nil {
		return nil, err
	}
	return &resource, nil
}

// List retrieves all resources with pagination and filtering
func (d *DAO[T]) List(page, pageSize int, filter map[string]interface{}) ([]T, int64, error) {
	var resources []T
	var total int64

	// Create a new instance of T to get the table name
	var obj T
	query := d.db.Model(&obj)
	if filter != nil {
		query = query.Where(filter)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err = query.Offset(offset).Limit(pageSize).Find(&resources).Error
	if err != nil {
		return nil, 0, err
	}

	return resources, total, nil
}

// Update updates a resource by ID
func (d *DAO[T]) Update(id uint, resource *T) error {
	result := d.db.Model(resource).Where("id = ?", id).Updates(resource)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Delete deletes a resource by ID
func (d *DAO[T]) Delete(id uint) error {
	var resource T
	result := d.db.Delete(&resource, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// AutoMigrate performs database migration for the resource
func (d *DAO[T]) AutoMigrate() error {
	var obj T
	return d.db.AutoMigrate(&obj)
}

// Transaction executes a function within a database transaction
func (d *DAO[T]) Transaction(fc func(tx *gorm.DB) error) error {
	return d.db.Transaction(fc)
}
