package internal

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ListResponse represents a paginated list response
type ListResponse[T any] struct {
	Items []T   `json:"items"`
	Total int64 `json:"total"`
	Page  int   `json:"page"`
	Size  int   `json:"size"`
}

// RegisterResource registers CRUD routes for a resource
func RegisterResource[T any](router *gin.Engine, db *gorm.DB, path string) {
	dao := NewDAO[T](db)

	// Auto-migrate the resource
	if err := dao.AutoMigrate(); err != nil {
		panic(err)
	}

	// Create routes group
	group := router.Group(path)
	{
		// Create resource
		group.POST("", func(c *gin.Context) {
			var obj T
			if err := c.ShouldBindJSON(&obj); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			// Use transaction for create operation
			if err := dao.Transaction(func(tx *gorm.DB) error {
				return tx.Create(&obj).Error
			}); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusCreated, obj)
		})

		// Get resource by ID
		group.GET("/:id", func(c *gin.Context) {
			id, err := strconv.ParseUint(c.Param("id"), 10, 32)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
				return
			}

			obj, err := dao.Get(uint(id))
			if err != nil {
				if err == gorm.ErrRecordNotFound {
					c.JSON(http.StatusNotFound, gin.H{"error": "Resource not found"})
					return
				}
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, obj)
		})

		// List all resources with pagination and filtering
		group.GET("", func(c *gin.Context) {
			// Parse pagination parameters
			page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
			pageSize, _ := strconv.Atoi(c.DefaultQuery("size", "10"))

			// Parse filters from query parameters
			filters := make(map[string]interface{})
			for key, values := range c.Request.URL.Query() {
				if key != "page" && key != "size" {
					filters[key] = values[0]
				}
			}

			items, total, err := dao.List(page, pageSize, filters)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			response := ListResponse[T]{
				Items: items,
				Total: total,
				Page:  page,
				Size:  pageSize,
			}
			c.JSON(http.StatusOK, response)
		})

		// Update resource
		group.PUT("/:id", func(c *gin.Context) {
			id, err := strconv.ParseUint(c.Param("id"), 10, 32)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
				return
			}

			var obj T
			if err := db.First(&obj, id).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					c.JSON(http.StatusNotFound, gin.H{"error": "Resource not found"})
					return
				}
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			if err := c.ShouldBindJSON(&obj); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			// Use transaction for update operation
			if err := dao.Transaction(func(tx *gorm.DB) error {
				return tx.Save(&obj).Error
			}); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, obj)
		})

		// Delete resource
		group.DELETE("/:id", func(c *gin.Context) {
			id, err := strconv.ParseUint(c.Param("id"), 10, 32)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
				return
			}

			// Use transaction for delete operation
			if err := dao.Transaction(func(tx *gorm.DB) error {
				return tx.Delete(new(T), id).Error
			}); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusNoContent, nil)
		})
	}
}
