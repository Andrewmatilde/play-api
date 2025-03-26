package internal

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Validator interface for resource validation
type Validator interface {
	Validate() error
}

// Router handles HTTP routing for a resource
type Router[T any] struct {
	engine *gin.Engine
	db     *gorm.DB
	dao    *DAO[T]
}

// NewRouter creates a new router for the given resource
func NewRouter[T any](engine *gin.Engine, db *gorm.DB) *Router[T] {
	return &Router[T]{
		engine: engine,
		db:     db,
		dao:    NewDAO[T](db),
	}
}

// Register registers all CRUD routes for the resource
func (r *Router[T]) Register(path string) {
	group := r.engine.Group(path)
	{
		group.POST("", r.Create)
		group.GET("", r.List)
		group.GET("/:id", r.Get)
		group.PUT("/:id", r.Update)
		group.DELETE("/:id", r.Delete)
	}
}

// Create handles POST requests to create a new resource
func (r *Router[T]) Create(c *gin.Context) {
	var resource T
	if err := c.ShouldBindJSON(&resource); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if resource implements Validator interface
	if validator, ok := any(&resource).(Validator); ok {
		if err := validator.Validate(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	if err := r.dao.Create(&resource); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, resource)
}

// List handles GET requests to list resources
func (r *Router[T]) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("size", "10"))

	items, _, err := r.dao.List(page, pageSize, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return empty list instead of null
	if items == nil {
		items = make([]T, 0)
	}

	// Return items directly for backward compatibility
	c.JSON(http.StatusOK, items)
}

// Get handles GET requests to retrieve a resource by ID
func (r *Router[T]) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	resource, err := r.dao.Get(uint(id))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resource)
}

// Update handles PUT requests to update a resource
func (r *Router[T]) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var resource T
	if err := c.ShouldBindJSON(&resource); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := r.dao.Update(uint(id), &resource); err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resource)
}

// Delete handles DELETE requests to delete a resource
func (r *Router[T]) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := r.dao.Delete(uint(id)); err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
