package catalog

import (
	"context"

	"go-fiber-core/internal/dtos/connect"
	"go-fiber-core/internal/models"
	catalogRepo "go-fiber-core/internal/repositories/catalog"
	"go-fiber-core/internal/services"
)

type CatalogService interface {
	GetAll(ctx context.Context) (models.AllCatalogsResponse, error)
}

type catalogService struct {
	services.TransactionManager
	catalogRepo catalogRepo.CatalogRepository
}

func NewCatalogService(
	catalogRepo catalogRepo.CatalogRepository,
	connect *connect.ConnectDTO,
) CatalogService {
	return &catalogService{
		TransactionManager: services.NewTransactionManager(connect),
		catalogRepo:        catalogRepo,
	}
}

func (s *catalogService) GetAll(ctx context.Context) (models.AllCatalogsResponse, error) {
	// Usamos la conexión de lectura para obtener catálogos
	dbRead := s.TransactionManager.Connection().ConnectGormRead
	return s.catalogRepo.GetAll(ctx, dbRead)
}
