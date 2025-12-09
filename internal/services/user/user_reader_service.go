// internal/services/user/user_reader_service.go
package user

import (
	"context"
	"go-fiber-core/internal/dtos"
	"go-fiber-core/internal/dtos/connect"
	"go-fiber-core/internal/models"
	userRepo "go-fiber-core/internal/repositories/user"
)

// UserReaderService define la interfaz para las operaciones de lectura de usuarios.
type UserReaderService interface {
	GetByID(ctx context.Context, id uint64) (*models.User, error)
	GetAll(ctx context.Context) ([]models.User, error)
	GetAllPaginated(ctx context.Context, req dtos.PaginationRequest) (*dtos.PaginationResponse[models.User], error)
}

type userReaderService struct {
	conn          *connect.ConnectDTO
	userReader    userRepo.UserReader
	userPaginator userRepo.UserPaginator
}

// NewUserReaderService es el constructor del servicio de lectura.
func NewUserReaderService(
	conn *connect.ConnectDTO,
	reader userRepo.UserReader,
	paginator userRepo.UserPaginator,
) UserReaderService {
	return &userReaderService{
		conn:          conn,
		userReader:    reader,
		userPaginator: paginator,
	}
}

func (s *userReaderService) GetByID(ctx context.Context, id uint64) (*models.User, error) {
	return s.userReader.GetByID(ctx, s.conn.ConnectGormRead, id)
}

func (s *userReaderService) GetAll(ctx context.Context) ([]models.User, error) {
	return s.userReader.GetAll(ctx, s.conn.ConnectGormRead)
}

func (s *userReaderService) GetAllPaginated(ctx context.Context, req dtos.PaginationRequest) (*dtos.PaginationResponse[models.User], error) {
	return s.userPaginator.GetAllPaginated(ctx, s.conn.ConnectGormRead, req)
}
