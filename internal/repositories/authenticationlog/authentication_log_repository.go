package authenticationlog

import (
	"context"
	"go-fiber-core/internal/models"

	"gorm.io/gorm"
)

type AuthenticationLogWriter interface {
	Create(ctx context.Context, db *gorm.DB, log *models.AuthenticationLog) error
}

type AuthenticationLogRepository interface {
	AuthenticationLogWriter
}

type authenticationLogWriterRepo struct{}

func NewAuthenticationLogWriterRepo() AuthenticationLogWriter { return &authenticationLogWriterRepo{} }

func (w *authenticationLogWriterRepo) Create(ctx context.Context, db *gorm.DB, log *models.AuthenticationLog) error {
	return db.WithContext(ctx).Create(log).Error
}

type authenticationLogRepository struct {
	AuthenticationLogWriter
}

func NewAuthenticationLogRepository(w AuthenticationLogWriter) AuthenticationLogRepository {
	return &authenticationLogRepository{AuthenticationLogWriter: w}
}
