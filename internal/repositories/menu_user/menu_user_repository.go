package menu_user

import (
	"context"
	"go-fiber-core/internal/dtos"

	"go-fiber-core/internal/models"
	"go-fiber-core/internal/services/pagination"
	"fmt"

	"gorm.io/gorm"
)

type MenuUserPaginationRepository struct {
	ps *pagination.PaginationService[models.MenuUser]
}

func NewMenuUserPaginationRepository(ps *pagination.PaginationService[models.MenuUser]) MenuUserPagination {
	return &MenuUserPaginationRepository{ps: ps}
}

func (r *MenuUserPaginationRepository) GetAllPaginated(
	ctx context.Context,
	db *gorm.DB,
	req dtos.PaginationRequest,
) (*dtos.PaginationResponse[models.MenuUser], error) {

	return r.ps.Execute(
		db.WithContext(ctx).
			Select("menu_user.id, menu_user.menu_id, menu_user.user_id, menu_user.operator_id, menu_user.is_active, menu_user.created_at, menu_user.updated_at, menu_user.deleted_at").
			Joins("LEFT JOIN menus ON menus.id = menu_user.menu_id").
			Joins("LEFT JOIN users ON users.id = menu_user.user_id").       // User normal
			Joins("LEFT JOIN users AS operators ON operators.id = menu_user.operator_id"), // Operador
		req,
		func(q *gorm.DB) *gorm.DB {
			return q.
				Preload("Menu").
				Preload("User").
				Preload("Operator", func(db *gorm.DB) *gorm.DB {
					return db.Unscoped()
				})
		},
		nil,
	)
}
// Menus asociados al usuario (Sin duplicados y compatible con Postgres)
func (r *MenuUserPaginationRepository) GetMenusByUser(
	ctx context.Context,
	db *gorm.DB,
	userID uint,
	req dtos.PaginationRequest,
) (*dtos.PaginationResponse[models.MenuUserResponse], error) {

	var raw []models.MenuUserRow
	var result []models.MenuUserResponse
	var total int64

	base := db.WithContext(ctx).
		Table("menu_user mu").
		Joins("LEFT JOIN menus m ON m.id = mu.menu_id").
		Joins("LEFT JOIN users u ON u.id = mu.user_id").
		Joins("LEFT JOIN users o ON o.id = mu.operator_id").
		Where("mu.user_id = ? AND m.item_type = ?", userID, "link")

	/* =================================================
	   🔥 FILTROS DINÁMICOS (desde tu JSON)
	================================================= */
	for i, f := range req.FilterBy {

		if i >= len(req.FilterValues) {
			continue
		}

		val := req.FilterValues[i]

		switch f {

			case "operator_id":
				base = base.Where("mu.operator_id = ?", val)
			
			case "user_id":
				base = base.Where("mu.user_id = ?", val)
			
			case "menu_id":
				base = base.Where("mu.menu_id = ?", val)
			
			case "item_name":
				base = base.Where("m.item_name ILIKE ?", "%"+fmt.Sprint(val)+"%")
			
			/* 🔥 NUEVO */
			case "is_active": // estado del vínculo menu_user
				base = base.Where("mu.is_active = ?", val)
			
			case "operator.is_active": // estado del operador (users)
				base = base.Where("o.is_active = ?", val)
			
			case "users.is_active": // estado del usuario (users)
				base = base.Where("u.is_active = ?", val)

			case "menus.is_active": // estado del menú (menus)
				base = base.Where("m.is_active = ?", val)

				// ===== FUZZY (ILIKE) =====
			case "email", "user.email":
				base = base.Where("u.email ILIKE ?", "%"+fmt.Sprint(val)+"%")
			
			case "operator_name", "operator.name":
				base = base.Where("o.name ILIKE ?", "%"+fmt.Sprint(val)+"%")
			
			case "menu.name":
				base = base.Where("m.item_name ILIKE ?", "%"+fmt.Sprint(val)+"%")
		}

	}

	/* =================================================
	   🔥 SORT DINÁMICO
	================================================= */
	if len(req.SortBy) > 0 {

		for i, col := range req.SortBy {

			desc := false
			if i < len(req.SortDesc) {
				desc = req.SortDesc[i]
			}

			dir := "ASC"
			if desc {
				dir = "DESC"
			}

			switch col {

			case "item_name":
				base = base.Order("m.item_name " + dir)

			case "created_at":
				base = base.Order("mu.created_at " + dir)

			case "operator_id":
				base = base.Order("mu.operator_id " + dir)

			default:
				base = base.Order("mu.id DESC")
			}
		}
	} else {
		base = base.Order("mu.created_at DESC")
	}

	/* =================================================
	   COUNT
	================================================= */
	if err := base.Count(&total).Error; err != nil {
		return nil, err
	}

	/* =================================================
	   DATA (SCAN plano)
	================================================= */
	if err := base.
		Select(`
			mu.id, mu.menu_id, mu.user_id, mu.operator_id,
			mu.is_active, mu.created_at, mu.updated_at,

			m.id   AS menu_id_ref,
			m.item_type AS menu_type,
			m.item_name AS menu_name,
			m.icon AS menu_icon,

			u.id   AS user_id_ref,
			u.name AS user_name,
			u.email AS user_email,

			o.id   AS operator_id_ref,
			o.name AS operator_name
		`).
		Limit(req.RowsPerPage).
		Offset((req.Page - 1) * req.RowsPerPage).
		Scan(&raw).Error; err != nil {
		return nil, err
	}

	/* =================================================
	   MAPEO → STRUCT FINAL
	================================================= */
	for _, r := range raw {

		item := models.MenuUserResponse{
			ID:         r.ID,
			MenuID:     r.MenuID,
			UserID:     r.UserID,
			OperatorID: r.OperatorID,
			IsActive:   r.IsActive,
			CreatedAt:  r.CreatedAt,
			UpdatedAt:  r.UpdatedAt,
			Menu: models.MenuResponse{
				ID:   r.MenuIDRef,
				Type: r.MenuType,
				Name: r.MenuName,
				Icon: r.MenuIcon,
			},
			User: models.UserResponse{
				ID:    r.UserIDRef,
				Name:  r.UserName,
				Email: r.UserEmail,
			},
		}

		if r.OperatorIDRef != nil && r.OperatorName != nil {
			item.Operator = &models.UserResponse{
				ID:   *r.OperatorIDRef,
				Name: *r.OperatorName,
			}
		}

		result = append(result, item)
	}

	totalPages := 0
	if req.RowsPerPage > 0 {
		totalPages = int((total + int64(req.RowsPerPage) - 1) / int64(req.RowsPerPage))
	}

	return &dtos.PaginationResponse[models.MenuUserResponse]{
		Data:        result,
		TotalRows:   total,
		TotalPages:  totalPages,
		Page:        req.Page,
		RowsPerPage: req.RowsPerPage,
		Extras:      map[string]any{},
	}, nil
}


// Menus NO asociados al usuario (SIN duplicados, 1 fila = 1 menú)
func (r *MenuUserPaginationRepository) GetMenusNotByUser(
	ctx context.Context,
	db *gorm.DB,
	userID uint,
	req dtos.PaginationRequest,
) (*dtos.PaginationResponse[models.MenuUser], error) {

	return r.ps.Execute(
		db.WithContext(ctx).
			Model(&models.MenuUser{}).
			Joins("JOIN menus ON menus.id = menu_user.menu_id").
			Where("menus.item_type = ?", "link").
			Where(`
				menus.id NOT IN (
					SELECT menu_id 
					FROM menu_user 
					WHERE user_id = ?
				)
			`, userID),
		req,
		func(q *gorm.DB) *gorm.DB {
			return q.Preload("Menu")
		},
		nil,
	)
}
