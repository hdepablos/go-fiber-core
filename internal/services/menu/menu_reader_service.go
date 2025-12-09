package menu

import (
	"context"
	"go-fiber-core/internal/dtos/responses"
	"go-fiber-core/internal/repositories/menu" // Importamos el Repositorio de Menú
)

// ────────────────────────────────────────────────
// INTERFAZ
// ────────────────────────────────────────────────
type MenuReaderService interface {
	GetMenuByUser(ctx context.Context, userID uint64) ([]responses.MenuItemResponse, error)
}

// ────────────────────────────────────────────────
// IMPLEMENTACIÓN
// ────────────────────────────────────────────────
type menuReaderService struct {
	menuReaderRepo menu.MenuReader // Inyectamos el Repositorio
}

// NewMenuReaderService crea una nueva instancia del servicio, inyectando el repositorio.
func NewMenuReaderService(menuReaderRepo menu.MenuReader) MenuReaderService {
	return &menuReaderService{menuReaderRepo: menuReaderRepo}
}

// ────────────────────────────────────────────────
// OBTENER MENÚ POR USUARIO
// ────────────────────────────────────────────────
// Obtiene la lista plana del repositorio y construye la jerarquía (árbol de menú).
func (s *menuReaderService) GetMenuByUser(ctx context.Context, userID uint64) ([]responses.MenuItemResponse, error) {
	// 1️⃣ OBTENER DATOS PLANOS del repositorio
	// Pasamos 'nil' o la conexión si el repositorio lo requiere, pero idealmente el repo
	// ya maneja su conexión inyectada.
	menus, err := s.menuReaderRepo.GetMenusByUserID(ctx, nil, userID)

	if err != nil {
		return nil, err
	}
	if len(menus) == 0 {
		return []responses.MenuItemResponse{}, nil
	}

	// 2️⃣ Mapear los menús por ID para construcción de jerarquía
	menuMap := make(map[uint]*responses.MenuItemResponse)
	for _, m := range menus {
		menuMap[m.ID] = &responses.MenuItemResponse{
			Type: m.ItemType,
			Text: m.ItemName,
			To:   m.ToPath,
			Icon: m.Icon,
		}
	}

	// 3️⃣ Construir jerarquía (dos pasadas para anidamiento correcto)
	var roots []responses.MenuItemResponse
	processed := make(map[uint]bool) // Mapa para marcar qué ítems son hijos anidados

	// PASADA 1: ANIDAR HIJOS
	for _, m := range menus {
		if m.ParentID != nil {
			currentMenuItem := menuMap[m.ID]
			parentID := *m.ParentID
			parent := menuMap[parentID]

			if parent != nil {
				// Anidamos al hijo al padre (copia de valor)
				parent.Children = append(parent.Children, *currentMenuItem)
				processed[m.ID] = true // Marcamos que este ID es un hijo anidado
			}
		}
	}

	// PASADA 2: RECOLECTAR SOLO RAÍCES (ítems que no fueron anidados)
	for _, m := range menus {
		if !processed[m.ID] {
			roots = append(roots, *menuMap[m.ID])
		}
	}

	return roots, nil
}
