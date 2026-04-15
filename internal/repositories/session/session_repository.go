package session

import (
	"context"
	"go-fiber-core/internal/dtos"
	"go-fiber-core/internal/models"
	"go-fiber-core/internal/services/pagination"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SessionReader define métodos de lectura para sesiones
type SessionReader interface {
	GetByID(ctx context.Context, db *gorm.DB, id uuid.UUID) (*models.Session, error)
	GetActiveSessionsByUserID(ctx context.Context, db *gorm.DB, userID uint64) ([]models.Session, error)
}

// SessionPagination define métodos de paginación para sesiones
type SessionPagination interface {
	GetActiveSessionsPaginated(ctx context.Context, db *gorm.DB, req dtos.PaginationRequest) (*dtos.PaginationResponse[models.Session], error)
}

// SessionWriter define métodos de escritura para sesiones
type SessionWriter interface {
	Create(ctx context.Context, db *gorm.DB, session *models.Session) error
	Revoke(ctx context.Context, db *gorm.DB, id uuid.UUID) error
	RevokeAllByUserID(ctx context.Context, db *gorm.DB, userID uint64) error
	RevokeAll(ctx context.Context, db *gorm.DB) error // 👈 NUEVO
}

// SessionRepository combina lectura y escritura
type SessionRepository interface {
	SessionReader
	SessionWriter
	SessionPagination
}

// --- Implementación Reader ---

type SessionReaderRepo struct{}

func NewSessionReaderRepo() SessionReader { return &SessionReaderRepo{} }

func (r *SessionReaderRepo) GetByID(ctx context.Context, db *gorm.DB, id uuid.UUID) (*models.Session, error) {
	var session models.Session
	err := db.WithContext(ctx).Preload("User").First(&session, "id = ?", id).Error
	return &session, err
}

func (r *SessionReaderRepo) GetActiveSessionsByUserID(ctx context.Context, db *gorm.DB, userID uint64) ([]models.Session, error) {
	var sessions []models.Session
	err := db.WithContext(ctx).
		Where("user_id = ? AND is_blocked = ? AND expires_at > ?", userID, false, time.Now()).
		Find(&sessions).Error
	return sessions, err
}

// --- Implementación Paginator ---

type SessionPaginationRepo struct {
	ps pagination.Service[models.Session]
}

func NewSessionPaginationRepo(ps pagination.Service[models.Session]) SessionPagination {
	return &SessionPaginationRepo{ps: ps}
}

func (r *SessionPaginationRepo) GetActiveSessionsPaginated(ctx context.Context, db *gorm.DB, req dtos.PaginationRequest) (*dtos.PaginationResponse[models.Session], error) {
	// Filtro base: solo sesiones activas (no bloqueadas y no expiradas)
	modifier := func(query *gorm.DB) *gorm.DB {
		return query.
			Preload("User"). // Cargar datos del usuario
			Where("is_blocked = ? AND expires_at > ?", false, time.Now())
	}

	return r.ps.Execute(db.WithContext(ctx), req, modifier, nil)
}

// --- Implementación Writer ---

type SessionWriterRepo struct{}

func NewSessionWriterRepo() SessionWriter { return &SessionWriterRepo{} }

func (w *SessionWriterRepo) Create(ctx context.Context, db *gorm.DB, session *models.Session) error {
	return db.WithContext(ctx).Create(session).Error
}

func (w *SessionWriterRepo) Revoke(ctx context.Context, db *gorm.DB, id uuid.UUID) error {
	return db.WithContext(ctx).
		Model(&models.Session{}).
		Where("id = ?", id).
		Update("is_blocked", true).Error
}

func (w *SessionWriterRepo) RevokeAllByUserID(ctx context.Context, db *gorm.DB, userID uint64) error {
	return db.WithContext(ctx).
		Model(&models.Session{}).
		Where("user_id = ? AND is_blocked = ?", userID, false).
		Update("is_blocked", true).Error
}

func (w *SessionWriterRepo) RevokeAll(ctx context.Context, db *gorm.DB) error {
	return db.WithContext(ctx).
		Model(&models.Session{}).
		Where("is_blocked = ?", false).
		Update("is_blocked", true).Error
}

// --- Implementación Combinada ---

type sessionRepository struct {
	SessionReader
	SessionWriter
	SessionPagination
}

func NewSessionRepository(r SessionReader, w SessionWriter, p SessionPagination) SessionRepository {
	return &sessionRepository{r, w, p}
}
