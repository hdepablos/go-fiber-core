package menu

import (
	"context"
	"go-fiber-core/internal/dtos/connect"
	"go-fiber-core/internal/repositories/menu"
	"go-fiber-core/internal/models"
	"go-fiber-core/internal/dtos/requests"
	"errors"
)

// Interfaz del servicio
type MenuWriterService interface {
	AddBulkUsers(ctx context.Context, menuIDs, userIDs []uint64) error
	BulkRemoveUsers(ctx context.Context, menuIDs, userIDs []uint64) error
	CreateMenu(ctx context.Context, req requests.CreateMenuRequest) (*models.Menu, error)
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

	// Uso el writer exacto como tu repository
	db := s.conn.ConnectGormWrite

	return s.repo.AddBulkUsers(ctx, db, menuIDs, userIDs)
}

func (s *menuWriterService) BulkRemoveUsers(
	ctx context.Context,
	menuIDs []uint64,
	userIDs []uint64,
) error {

	db := s.conn.ConnectGormWrite

	return s.repo.BulkRemoveUsers(ctx, db, menuIDs, userIDs)
}

func (s *menuWriterService) CreateMenu(
	ctx context.Context,
	req requests.CreateMenuRequest,
) (*models.Menu, error) {

	db := s.conn.ConnectGormWrite

	// 1️⃣ Si viene parent_id, validar que exista
	if req.ParentID != nil {
		parent, err := s.repo.GetByID(ctx, db, *req.ParentID)
		if err != nil {
			return nil, errors.New("el menú padre no existe")
		}

		// (opcional pero recomendado)
		if parent.DeletedAt.Valid {
			return nil, errors.New("el menú padre está eliminado")
		}
	}

	menu := &models.Menu{
		ItemType:   req.ItemType,
		ItemName:   req.ItemName,
		ToPath:     req.ToPath,
		Icon:       req.Icon,
		ParentID:   req.ParentID,
		OrderIndex: req.OrderIndex,
		IsActive:   true,
	}

	if err := s.repo.Create(ctx, db, menu); err != nil {
		return nil, err
	}

	return menu, nil
}
