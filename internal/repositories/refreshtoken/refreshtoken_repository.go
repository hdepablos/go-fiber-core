package refreshtoken

import (
	"context"
	"go-fiber-core/internal/models"

	"gorm.io/gorm"
)

// --- INTERFACES SEGREGADAS POR ROL ---

type RefreshTokenReader interface {
	GetByToken(ctx context.Context, db *gorm.DB, token string) (*models.RefreshToken, error)
}
type RefreshTokenWriter interface {
	Create(ctx context.Context, db *gorm.DB, token *models.RefreshToken) error
	DeleteByUserID(ctx context.Context, db *gorm.DB, userID uint64) error
}
type RefreshTokenRepository interface {
	RefreshTokenReader
	RefreshTokenWriter
}

// --- STRUCTS Y CONSTRUCTORES GRANULARES ---

type RefreshTokenReaderRepo struct{}

func NewRefreshTokenReaderRepo() RefreshTokenReader { return &RefreshTokenReaderRepo{} }

type RefreshTokenWriterRepo struct{}

func NewRefreshTokenWriterRepo() RefreshTokenWriter { return &RefreshTokenWriterRepo{} }

// --- STRUCT Y CONSTRUCTOR COMPUESTO ---

type refreshTokenRepository struct {
	RefreshTokenReader
	RefreshTokenWriter
}

func NewRefreshTokenRepository(r RefreshTokenReader, w RefreshTokenWriter) RefreshTokenRepository {
	return &refreshTokenRepository{r, w}
}

// --- IMPLEMENTACIONES DE MÉTODOS ---

func (r *RefreshTokenWriterRepo) Create(ctx context.Context, db *gorm.DB, token *models.RefreshToken) error {
	return db.WithContext(ctx).Create(token).Error
}
func (r *RefreshTokenWriterRepo) DeleteByUserID(ctx context.Context, db *gorm.DB, userID uint64) error {
	return db.WithContext(ctx).Where("user_id = ?", userID).Delete(&models.RefreshToken{}).Error
}
func (r *RefreshTokenReaderRepo) GetByToken(ctx context.Context, db *gorm.DB, token string) (*models.RefreshToken, error) {
	var refreshToken models.RefreshToken
	err := db.WithContext(ctx).Where("token = ?", token).First(&refreshToken).Error
	if err != nil {
		return nil, err
	}
	return &refreshToken, nil
}
