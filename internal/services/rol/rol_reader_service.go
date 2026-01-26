package rol

import (
	"context"
	"go-fiber-core/internal/dtos/connect"
	"go-fiber-core/internal/models"
	rolRepo "go-fiber-core/internal/repositories/rol"
)

type RolReaderService interface {
	GetByID(ctx context.Context, id uint) (*models.Role, error)
	GetAll(ctx context.Context) ([]models.Role, error)
}

type rolReaderService struct {
	conn       *connect.ConnectDTO
	roleReader rolRepo.RolReader
}

func NewRolReaderService(conn *connect.ConnectDTO, reader rolRepo.RolReader) RolReaderService {
	return &rolReaderService{
		conn:       conn,
		roleReader: reader,
	}
}

func (s *rolReaderService) GetByID(ctx context.Context, id uint) (*models.Role, error) {
	return s.roleReader.GetByID(ctx, s.conn.ConnectGormRead, id)
}

func (s *rolReaderService) GetAll(ctx context.Context) ([]models.Role, error) {
	return s.roleReader.GetAll(ctx, s.conn.ConnectGormRead)
}
