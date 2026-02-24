package menu

import (
	"context"
	"fmt"
	"go-fiber-core/internal/dtos/connect"
	"go-fiber-core/internal/repositories/menu"
)

// Interfaz del servicio
type MenuWriterService interface {
	AddBulkUsers(ctx context.Context, menuIDs, userIDs []uint64) error
	BulkRemoveUsers(ctx context.Context, menuIDs, userIDs []uint64) error
}

type menuWriterService struct {
	repo menu.MenuWriter
	conn *connect.ConnectDTO
}

func NewMenuWriterService(repo menu.MenuWriter, conn *connect.ConnectDTO) MenuWriterService {
	return &menuWriterService{
		repo: repo,
		conn: conn,
	}
}

func (s *menuWriterService) AddBulkUsers(
	ctx context.Context,
	menuIDs []uint64,
	userIDs []uint64,
) error {


	fmt.Println(" Va a procesarrrrr")

	// Uso el writer exacto como tu repository
	db := s.conn.ConnectGormWrite

	error := s.repo.AddBulkUsers(ctx, db, menuIDs, userIDs)
		if error != nil {
			fmt.Print("error general")
			fmt.Print(error.Error())
		}

	return nil
}

func (s *menuWriterService) BulkRemoveUsers(
	ctx context.Context,
	menuIDs []uint64,
	userIDs []uint64,
) error {

	db := s.conn.ConnectGormWrite

	return s.repo.BulkRemoveUsers(ctx, db, menuIDs, userIDs)
}
