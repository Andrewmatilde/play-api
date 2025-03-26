package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"my-embedded-api/apiv1"
	"my-embedded-api/internal"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Config holds the application configuration
type Config struct {
	// Server configuration
	Server struct {
		Port string `default:":8080"`
	}

	// Database configuration
	Database struct {
		Path string `default:"app.db"`
	}

	// Logging configuration
	Logging struct {
		Level string `default:"info"`
	}
}

// NewConfig creates a new configuration with default values
func NewConfig() *Config {
	config := &Config{}

	// Set default values
	config.Server.Port = ":8080"
	config.Database.Path = "app.db"
	config.Logging.Level = "info"

	return config
}

func main() {
	// Load configuration
	config := NewConfig()

	// Initialize standard logger
	stdLogger := log.New(os.Stdout, "", log.LstdFlags)

	// Initialize GORM logger
	gormLogger := logger.Default.LogMode(logger.Info)

	// Initialize database with logging
	db, err := gorm.Open(sqlite.Open(config.Database.Path), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		stdLogger.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize Gin router
	router := gin.Default()

	// Add middleware
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	// Register resources
	internal.RegisterResource[apiv1.User](router, db, "/api/v1/users")

	// Create HTTP server
	srv := &http.Server{
		Addr:    config.Server.Port,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		stdLogger.Printf("Starting server on %s", config.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			stdLogger.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Graceful shutdown
	stdLogger.Println("Shutting down server...")

	// Create shutdown context with 5 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := srv.Shutdown(ctx); err != nil {
		stdLogger.Fatalf("Server forced to shutdown: %v", err)
	}

	stdLogger.Println("Server exiting")
}
