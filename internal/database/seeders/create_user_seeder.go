package seeders

import (
	"context"
	"fmt"
	"log/slog"

	"go-fiber-core/cmd/api/di"
	"go-fiber-core/internal/models"
)

type SeedUser struct {
	Name     string
	Email    string
	Password string
}

var seedUsers = []SeedUser{
	{"Admin", "admin@gmail.com", "123456"},   // Admin
	{"Coordinador", "coordinador@gmail.com", "123456"}, // Coordinador
	{"Supervisor", "supervisor@gmail.com", "123456"}, // Supervisor
	{"Operador", "operador@gmail.com", "123456"}, // Operador
}

// SOLO crea usuarios (sin roles)
func CreateUserSeeder(configPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultSeederTimeout)
	defer cancel()

	logger := slog.Default().With("seeder", "users")

	server, cleanup, err := di.InitializeServer(configPath)
	if err != nil {
		return fmt.Errorf("init DI: %w", err)
	}
	defer cleanup()

	userService := server.UserWriterService

	for _, u := range seedUsers {
		user := &models.User{
			Name:     u.Name,
			Email:    u.Email,
			Password: u.Password,
		}

		logger.Info("creando usuario", "email", u.Email)

		if err := userService.Create(ctx, user); err != nil {
			return fmt.Errorf("crear usuario %s: %w", u.Email, err)
		}
	}

	logger.Info("usuarios creados", "total", len(seedUsers))
	return nil
}