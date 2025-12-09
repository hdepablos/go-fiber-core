package interfaces

import (
	"go-fiber-core/internal/dtos"

	"gorm.io/gorm"
)

type PaginationService interface {
	Paginate(db *gorm.DB, req dtos.PaginationRequest) (*gorm.DB, error)
	ApplyFilters(db *gorm.DB, req dtos.PaginationRequest) *gorm.DB
}
