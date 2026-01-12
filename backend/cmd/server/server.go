// Package main is the entry point for the GoSvelteKit backend server.
package main

import (
	"fmt"
	"os"

	"gosveltekit/internal/auth"
	gormadapter "gosveltekit/internal/auth/adapter/gorm"
	"gosveltekit/internal/config"
	"gosveltekit/internal/email"
	"gosveltekit/internal/handlers"
	"gosveltekit/internal/logger"
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
		// Initialize logger with defaults before config is loaded
		logger.Init("info", "text")
		logger.Error("Falha ao carregar as configurações", "error", err)
		os.Exit(1)
	}

	// Initialize logger with config
	logLevel := cfg.Log.Level
	if logLevel == "" {
		logLevel = "info"
	}
	logFormat := cfg.Log.Format
	if logFormat == "" {
		logFormat = "text"
	}
	logger.Init(logLevel, logFormat)

	logger.Info("Iniciando servidor", "port", cfg.Server.Port)

	dbDSN := cfg.Database.DSN

	// Connect to SQLite
	db, err := gorm.Open(sqlite.Open(dbDSN), &gorm.Config{})
	if err != nil {
		logger.Error("Falha ao conectar ao banco de dados", "error", err, "dsn", dbDSN)
		os.Exit(1)
	}
	logger.Info("Conectado ao banco de dados", "dsn", dbDSN)

	// Migrate tables (including new Session table)
	if err := db.AutoMigrate(&models.User{}, &models.Session{}); err != nil {
		logger.Error("Falha ao executar migrações", "error", err)
		os.Exit(1)
	}
	logger.Info("Migrações executadas com sucesso")

	// Create admin user if not exists
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("Falha ao gerar hash da senha do admin", "error", err)
	}

	result := db.Where(models.User{Username: "admin"}).FirstOrCreate(&models.User{
		Username:     "admin",
		Email:        "onyx.views5004@eagereverest.com",
		DisplayName:  "Administrator",
		PasswordHash: string(passwordHash),
		Role:         "admin",
	})
	if result.Error != nil {
		logger.Error("Falha ao criar usuário admin", "error", result.Error)
	}
	logger.Info("Usuário admin verificado", "rows_affected", result.RowsAffected)

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
	port := ":8080"
	if cfg.Server.Port != 0 {
		port = fmt.Sprintf(":%d", cfg.Server.Port)
	}
	logger.Info("Servidor iniciado", "port", port)
	if err := r.Run(port); err != nil {
		logger.Error("Erro ao iniciar servidor", "error", err, "port", port)
		os.Exit(1)
	}
}
