package auth

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/dtos/connect"
	"go-fiber-core/internal/dtos/requests"
	"go-fiber-core/internal/dtos/responses"
	"go-fiber-core/internal/models"
	refreshTokenRepo "go-fiber-core/internal/repositories/refreshtoken"
	userRepo "go-fiber-core/internal/repositories/user"
	"go-fiber-core/internal/services"
	menuService "go-fiber-core/internal/services/menu"
)

// authService es la implementación de la interfaz AuthService.
type authService struct {
	services.TransactionManager
	userReader       userRepo.UserReader
	refreshTokenRepo refreshTokenRepo.RefreshTokenRepository
	tokenService     TokenService
	menuReader       menuService.MenuReaderService
}

// NewAuthService crea una nueva instancia del servicio de autenticación.
func NewAuthService(
	userReader userRepo.UserReader,
	refreshTokenRepo refreshTokenRepo.RefreshTokenRepository,
	tokenService TokenService,
	menuReader menuService.MenuReaderService,
	connect *connect.ConnectDTO,
) AuthService {
	return &authService{
		TransactionManager: services.NewTransactionManager(connect),
		userReader:         userReader,
		refreshTokenRepo:   refreshTokenRepo,
		tokenService:       tokenService,
		menuReader:         menuReader,
	}
}

// ────────────────────────────────────────────────
// LOGIN
// ────────────────────────────────────────────────
func (s *authService) Login(ctx context.Context, req requests.LoginRequest) (*responses.LoginResponse, error) {
	dbRead := s.TransactionManager.Conn.ConnectGormRead

	// 1️⃣ Buscar usuario por email, incluyendo Roles (asumimos GetByEmailWithRoles existe)
	user, err := s.userReader.GetByEmailWithRoles(ctx, dbRead, req.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrAuthentication
		}
		return nil, fmt.Errorf("error al buscar usuario: %w", err)
	}

	if !user.IsActive {
		return nil, domain.ErrAuthentication
	}

	// 2️⃣ Verificar contraseña usando bcrypt
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, domain.ErrAuthentication
	}

	// 3️⃣ Generar tokens
	userIDStr := strconv.FormatUint(user.ID, 10)
	accessToken, refreshToken, err := s.tokenService.GenerateTokens(userIDStr)
	if err != nil {
		return nil, errors.New("error al generar tokens")
	}

	// 4️⃣ Guardar nuevo refresh token y limpiar el anterior dentro de una transacción
	err = s.TransactionManager.ExecuteTx(ctx, func(tx *gorm.DB) error {
		if err := s.refreshTokenRepo.DeleteByUserID(ctx, tx, user.ID); err != nil {
			log.Printf("ADVERTENCIA: no se pudo eliminar el refresh token anterior para el usuario %d: %v", user.ID, err)
		}
		expiresAt := time.Now().Add(7 * 24 * time.Hour)
		newRefreshToken := &models.RefreshToken{
			UserID:    user.ID,
			Token:     refreshToken,
			ExpiresAt: expiresAt,
		}
		if err := s.refreshTokenRepo.Create(ctx, tx, newRefreshToken); err != nil {
			return errors.New("error al guardar la sesión")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// 5️⃣ Construir lista de roles
	var roleIDs []uint64
	var roleNames []string
	for _, r := range user.Roles {
		roleIDs = append(roleIDs, r.ID)
		roleNames = append(roleNames, r.Name)
	}

	// 6️⃣ Obtener menús del usuario: Llama al servicio de menú que construye la jerarquía
	menuItems, err := s.menuReader.GetMenuByUser(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("error al obtener el menú: %w", err)
	}

	// 7️⃣ Construir respuesta
	resp := &responses.LoginResponse{
		Token:        accessToken,
		RefreshToken: refreshToken,
		UserName:     user.Name,
		RoleIDs:      roleIDs,
		Roles:        roleNames,
		Menu:         menuItems,
	}

	return resp, nil
}

// ────────────────────────────────────────────────
// REFRESH TOKEN
// ────────────────────────────────────────────────
func (s *authService) Refresh(ctx context.Context, refreshTokenString string) (string, string, error) {
	token, err := s.tokenService.ValidateToken(refreshTokenString)
	if err != nil || !token.Valid {
		return "", "", domain.ErrAuthentication
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["typ"] != "refresh" {
		return "", "", domain.ErrAuthentication
	}

	var newAccessToken, newRefreshToken string

	dbRead := s.TransactionManager.Conn.ConnectGormRead
	storedToken, err := s.refreshTokenRepo.GetByToken(ctx, dbRead, refreshTokenString)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", domain.ErrAuthentication
		}
		return "", "", fmt.Errorf("error al buscar refresh token: %w", err)
	}

	err = s.TransactionManager.ExecuteTx(ctx, func(tx *gorm.DB) error {
		if err := s.refreshTokenRepo.DeleteByUserID(ctx, tx, storedToken.UserID); err != nil {
			return fmt.Errorf("error al eliminar refresh token anterior por UserID: %w", err)
		}
		userIDStr := strconv.FormatUint(storedToken.UserID, 10)
		newAccessToken, newRefreshToken, err = s.tokenService.GenerateTokens(userIDStr)
		if err != nil {
			return errors.New("error al generar nuevos tokens")
		}
		expiresAt := time.Now().Add(7 * 24 * time.Hour)
		newRefreshTokenModel := &models.RefreshToken{
			UserID:    storedToken.UserID,
			Token:     newRefreshToken,
			ExpiresAt: expiresAt,
		}
		if err := s.refreshTokenRepo.Create(ctx, tx, newRefreshTokenModel); err != nil {
			return errors.New("error al guardar la nueva sesión")
		}
		return nil
	})

	if err != nil {
		if errors.Is(err, domain.ErrAuthentication) {
			return "", "", err
		}
		log.Printf("ERROR en Refresh transaction: %v", err)
		return "", "", domain.ErrAuthentication
	}

	return newAccessToken, newRefreshToken, nil
}

// ────────────────────────────────────────────────
// LOGOUT
// ────────────────────────────────────────────────
func (s *authService) Logout(ctx context.Context, userID uint64) error {
	dbWrite := s.TransactionManager.Conn.ConnectGormWrite
	err := s.refreshTokenRepo.DeleteByUserID(ctx, dbWrite, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("error al cerrar sesión para el usuario %d: %w", userID, err)
	}
	return nil
}
