package rol

import (
	"context"
	"go-fiber-core/internal/dtos/connect"
	"go-fiber-core/internal/models"
	rolRepo "go-fiber-core/internal/repositories/rol"
)

type RolWriterService interface {
	Create(ctx context.Context, role *models.Role) error
	Update(ctx context.Context, id uint, updatedRoleData *models.Role) (*models.Role, error)
	SoftDelete(ctx context.Context, id uint) error
	HardDelete(ctx context.Context, id uint) error
}

type rolWriterService struct {
	conn       *connect.ConnectDTO
	roleWriter rolRepo.RolWriter
	roleReader rolRepo.RolReader
}

func NewRolWriterService(
	conn *connect.ConnectDTO,
	writer rolRepo.RolWriter,
	reader rolRepo.RolReader,
) RolWriterService {
	return &rolWriterService{
		conn:       conn,
		roleWriter: writer,
		roleReader: reader,
	}
}

func (s *rolWriterService) Create(ctx context.Context, role *models.Role) error {
	return s.roleWriter.Create(ctx, s.conn.ConnectGormWrite, role)
}

func (s *rolWriterService) Update(ctx context.Context, id uint, updatedRoleData *models.Role) (*models.Role, error) {
	existingRole, err := s.roleReader.GetByID(ctx, s.conn.ConnectGormWrite, id)
	if err != nil {
		return nil, err
	}

	existingRole.Name = updatedRoleData.Name
	existingRole.IsActive = updatedRoleData.IsActive

	if err := s.roleWriter.Update(ctx, s.conn.ConnectGormWrite, existingRole); err != nil {
		return nil, err
	}
	return existingRole, nil
}

func (s *rolWriterService) SoftDelete(ctx context.Context, id uint) error {
	return s.roleWriter.SoftDelete(ctx, s.conn.ConnectGormWrite, id)
}

func (s *rolWriterService) HardDelete(ctx context.Context, id uint) error {
	return s.roleWriter.HardDelete(ctx, s.conn.ConnectGormWrite, id)
}
