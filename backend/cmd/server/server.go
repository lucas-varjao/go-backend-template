// Package main is the entry point for the GoSvelteKit backend server.
package main

import (
	"fmt"
	"log"

	"gosveltekit/internal/auth"
	gormadapter "gosveltekit/internal/auth/adapter/gorm"
	"gosveltekit/internal/config"
	"gosveltekit/internal/email"
	"gosveltekit/internal/handlers"
	"gosveltekit/internal/models"
	"gosveltekit/internal/router"
	"gosveltekit/internal/service"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		panic("Falha ao carregar as configurações")
	}

	dbDSN := cfg.Database.DSN

	// Connect to SQLite
	db, err := gorm.Open(sqlite.Open(dbDSN), &gorm.Config{})
	if err != nil {
		panic("Falha ao conectar ao banco de dados")
	}

	// Migrate tables (including new Session table)
	db.AutoMigrate(&models.User{}, &models.Session{})

	// Create admin user if not exists
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println(err)
	}

	result := db.Where(models.User{Username: "admin"}).FirstOrCreate(&models.User{
		Username:     "admin",
		Email:        "onyx.views5004@eagereverest.com",
		DisplayName:  "Administrator",
		PasswordHash: string(passwordHash),
		Role:         "admin",
	})
	if result.Error != nil {
		fmt.Println(result.Error)
	}
	fmt.Printf("Admin user ready - rows affected: %d\n", result.RowsAffected)

	// Initialize adapters
	userAdapter := gormadapter.NewUserAdapter(db)
	sessionAdapter := gormadapter.NewSessionAdapter(db)

	// Initialize auth manager with default config
	authConfig := auth.DefaultAuthConfig()
	authManager := auth.NewAuthManager(userAdapter, sessionAdapter, authConfig)

	// Initialize services
	emailService := email.NewEmailService(cfg)
	authService := service.NewAuthService(authManager, userAdapter, emailService)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)

	// Setup router
	r := router.SetupRouter(authHandler, authManager)

	// Start server
	log.Println("Starting server on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Erro ao iniciar servidor: %v", err)
	}
}
