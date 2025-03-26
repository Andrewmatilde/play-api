package internal

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestDAO_CRUD(t *testing.T) {
	db := setupTestDB(t)
	err := db.AutoMigrate(&TestModel{})
	assert.NoError(t, err)

	dao := NewDAO[TestModel](db)

	// Test Create
	model := &TestModel{Name: "test"}
	err = dao.Create(model)
	assert.NoError(t, err)
	assert.NotZero(t, model.ID)

	// Test Get
	found, err := dao.Get(model.ID)
	assert.NoError(t, err)
	assert.Equal(t, model.ID, found.ID)
	assert.Equal(t, model.Name, found.Name)

	// Test Update
	model.Name = "updated"
	err = dao.Update(model.ID, model)
	assert.NoError(t, err)

	// Verify update
	found, err = dao.Get(model.ID)
	assert.NoError(t, err)
	assert.Equal(t, "updated", found.Name)

	// Test Delete
	err = dao.Delete(model.ID)
	assert.NoError(t, err)

	// Verify deletion
	_, err = dao.Get(model.ID)
	assert.Error(t, err)
	assert.Equal(t, gorm.ErrRecordNotFound, err)
}

func TestDAO_List(t *testing.T) {
	db := setupTestDB(t)
	err := db.AutoMigrate(&TestModel{})
	assert.NoError(t, err)

	dao := NewDAO[TestModel](db)

	// Create test data
	for i := 0; i < 5; i++ {
		model := &TestModel{Name: fmt.Sprintf("test%d", i)}
		err := dao.Create(model)
		assert.NoError(t, err)
	}

	// Test pagination
	items, total, err := dao.List(1, 2, nil)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, items, 2)

	// Test second page
	items, total, err = dao.List(2, 2, nil)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, items, 2)

	// Test last page
	items, total, err = dao.List(3, 2, nil)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, items, 1)
}

func TestDAO_Transaction(t *testing.T) {
	db := setupTestDB(t)
	dao := NewDAO[TestModel](db)

	// Test transaction
	err := dao.Transaction(func(tx *gorm.DB) error {
		model1 := &TestModel{Name: "model1"}
		if err := tx.Create(model1).Error; err != nil {
			return err
		}

		model2 := &TestModel{Name: "model2"}
		if err := tx.Create(model2).Error; err != nil {
			return err
		}

		return nil
	})
	assert.NoError(t, err)

	// Verify both models were created
	var count int64
	err = db.Model(&TestModel{}).Count(&count).Error
	assert.NoError(t, err)
	assert.Equal(t, int64(2), count)
}
