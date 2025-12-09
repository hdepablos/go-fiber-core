package seeders

import (
	"context"
	"fmt"
	"log/slog"

	"go-fiber-core/cmd/api/di"
	"go-fiber-core/internal/models"
)

// User seeder constants
const (
	defaultUserName     = "test"
	defaultUserEmail    = "test@test.com"
	defaultUserPassword = "123456"
)

// CreateUserSeeder creates a default test user using dependency injection.
// This seeder uses the application's DI container to access the UserWriterService.
func CreateUserSeeder(configPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultSeederTimeout)
	defer cancel()

	logger := slog.Default().With("seeder", "create_user")
	logger.Info("iniciando seeder de usuario de prueba")

	// Initialize DI container
	server, cleanup, err := di.InitializeServer(configPath)
	if err != nil {
		return fmt.Errorf("inicializar dependencias: %w", err)
	}
	defer cleanup()
	logger.Debug("dependencias inicializadas correctamente")

	// Get UserWriterService from DI container
	userService := server.UserWriterService
	if userService == nil {
		return fmt.Errorf("UserWriterService no disponible en el contenedor DI")
	}
	logger.Debug("servicio de usuario obtenido desde DI container")

	// Create default test user
	user := &models.User{
		Name:     defaultUserName,
		Email:    defaultUserEmail,
		Password: defaultUserPassword,
	}

	logger.Info("creando usuario de prueba", "email", user.Email)
	if err := userService.Create(ctx, user); err != nil {
		return fmt.Errorf("crear usuario: %w", err)
	}

	logger.Info("usuario de prueba creado exitosamente",
		"name", user.Name,
		"email", user.Email,
		"id", user.ID)

	return nil
}

// CreateUserSeederWithCustomData creates a user with custom data.
// Useful for creating multiple test users with different credentials.
func CreateUserSeederWithCustomData(configPath string, name, email, password string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultSeederTimeout)
	defer cancel()

	logger := slog.Default().With("seeder", "create_user_custom")
	logger.Info("iniciando seeder de usuario personalizado", "email", email)

	server, cleanup, err := di.InitializeServer(configPath)
	if err != nil {
		return fmt.Errorf("inicializar dependencias: %w", err)
	}
	defer cleanup()

	userService := server.UserWriterService
	if userService == nil {
		return fmt.Errorf("UserWriterService no disponible en el contenedor DI")
	}

	user := &models.User{
		Name:     name,
		Email:    email,
		Password: password,
	}

	logger.Info("creando usuario personalizado", "email", user.Email)
	if err := userService.Create(ctx, user); err != nil {
		return fmt.Errorf("crear usuario: %w", err)
	}

	logger.Info("usuario personalizado creado exitosamente",
		"name", user.Name,
		"email", user.Email,
		"id", user.ID)

	return nil
}
