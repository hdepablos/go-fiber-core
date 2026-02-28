package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/dtos"
	"go-fiber-core/internal/dtos/connect"
	"go-fiber-core/internal/dtos/requests"
	"go-fiber-core/internal/dtos/responses"
	"go-fiber-core/internal/models"
	refreshTokenRepo "go-fiber-core/internal/repositories/refreshtoken"
	sessionRepo "go-fiber-core/internal/repositories/session"
	userRepo "go-fiber-core/internal/repositories/user"
	"go-fiber-core/internal/services"
	"go-fiber-core/internal/services/cache"
	menuService "go-fiber-core/internal/services/menu"
)

// authService es la implementación de la interfaz AuthService.
type localAuthService struct {
	services.TransactionManager
	userReader       userRepo.UserReader
	refreshTokenRepo refreshTokenRepo.RefreshTokenRepository
	sessionRepo      sessionRepo.SessionRepository
	tokenService     TokenService
	menuReader       menuService.MenuReaderService
}

// NewAuthService crea una nueva instancia del servicio de autenticación.
// Decide qué implementación usar basándose en la configuración (Local vs Cognito).
func NewAuthService(
	userReader userRepo.UserReader,
	refreshTokenRepo refreshTokenRepo.RefreshTokenRepository,
	sessionRepo sessionRepo.SessionRepository,
	tokenService TokenService,
	menuReader menuService.MenuReaderService,
	connect *connect.ConnectDTO,
) AuthService {
	// Si AUTH_PROVIDER == "cognito", retornamos la implementación de Cognito.
	if os.Getenv("AUTH_PROVIDER") == "cognito" {
		return NewCognitoAuthService()
	}

	return &localAuthService{
		TransactionManager: services.NewTransactionManager(connect),
		userReader:         userReader,
		refreshTokenRepo:   refreshTokenRepo,
		sessionRepo:        sessionRepo,
		tokenService:       tokenService,
		menuReader:         menuReader,
	}
}

// ────────────────────────────────────────────────
// LOGIN
// ────────────────────────────────────────────────
func (s *localAuthService) Login(ctx context.Context, req requests.LoginRequest, userAgent, clientIP string) (*responses.LoginResponse, error) {
	dbRead := s.TransactionManager.Conn.ConnectGormRead

	// 1️⃣ Buscar usuario por email, incluyendo Roles
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

	// 2️⃣ Verificar contraseña
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, domain.ErrAuthentication
	}

	// 3️⃣ Generar Session ID
	sessionID := uuid.New()

	// 4️⃣ Generar tokens vinculados a la sesión
	userIDStr := strconv.FormatUint(user.ID, 10)
	accessToken, refreshToken, err := s.tokenService.GenerateTokens(userIDStr, sessionID.String())
	if err != nil {
		return nil, errors.New("error al generar tokens")
	}

	// 5️⃣ Guardar sesión y refresh token
	err = s.TransactionManager.ExecuteTx(ctx, func(tx *gorm.DB) error {
		// Guardar Sesión
		expiresAt := time.Now().Add(7 * 24 * time.Hour) // Misma duración que refresh token
		newSession := &models.Session{
			ID:        sessionID,
			UserID:    user.ID,
			UserAgent: userAgent,
			ClientIP:  clientIP,
			ExpiresAt: expiresAt,
			IsBlocked: false,
		}
		if err := s.sessionRepo.Create(ctx, tx, newSession); err != nil {
			return fmt.Errorf("error al crear sesión: %w", err)
		}

		newRefreshToken := &models.RefreshToken{
			UserID:    user.ID,
			Token:     refreshToken,
			ExpiresAt: expiresAt,
		}
		if err := s.refreshTokenRepo.Create(ctx, tx, newRefreshToken); err != nil {
			return errors.New("error al guardar refresh token")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// 6️⃣ Construir lista de roles
	var roleIDs []uint64
	var roleNames []string
	for _, r := range user.Roles {
		roleIDs = append(roleIDs, r.ID)
		roleNames = append(roleNames, r.Name)
	}

	// 7️⃣ Obtener menús
	menuItems, err := s.menuReader.GetMenuByUser(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("error al obtener el menú: %w", err)
	}

	// 8️⃣ Construir respuesta
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
func (s *localAuthService) Refresh(ctx context.Context, refreshTokenString string) (string, string, error) {
	// 1. Validar Token Criptográficamente
	token, err := s.tokenService.ValidateToken(refreshTokenString)
	if err != nil || !token.Valid {
		return "", "", domain.ErrAuthentication
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["typ"] != "refresh" {
		return "", "", domain.ErrAuthentication
	}

	// 2. Extraer Session ID de los claims (si existe)
	sessionIDStr, ok := claims["sid"].(string)
	if !ok {
		// Si es un token viejo sin SID, fallamos o permitimos renovación creando sesión nueva?
		// Por seguridad, mejor fallar o forzar re-login.
		// O intentar buscar si hay una sesión activa...
		// Asumiremos que si no hay SID, es inválido en este nuevo esquema estricto.
		return "", "", errors.New("token sin session ID (formato antiguo)")
	}
	sessionUUID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return "", "", errors.New("session ID inválido")
	}

	dbRead := s.TransactionManager.Conn.ConnectGormRead

	// 3. Verificar si la sesión es válida en DB
	session, err := s.sessionRepo.GetByID(ctx, dbRead, sessionUUID)
	if err != nil {
		return "", "", domain.ErrAuthentication // Sesión no encontrada
	}
	if session.IsBlocked {
		return "", "", domain.ErrAuthentication // Sesión revocada
	}
	if session.ExpiresAt.Before(time.Now()) {
		return "", "", domain.ErrAuthentication // Sesión expirada
	}

	// 4. Verificar existencia del refresh token en DB (rotación)
	storedToken, err := s.refreshTokenRepo.GetByToken(ctx, dbRead, refreshTokenString)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Reuse detection? Podríamos bloquear la sesión aquí si se detecta reuso.
			_ = s.sessionRepo.Revoke(ctx, s.TransactionManager.Conn.ConnectGormWrite, sessionUUID)
			return "", "", domain.ErrAuthentication
		}
		return "", "", fmt.Errorf("error al buscar refresh token: %w", err)
	}

	var newAccessToken, newRefreshToken string

	// 5. Rotar Tokens
	err = s.TransactionManager.ExecuteTx(ctx, func(tx *gorm.DB) error {
		// Borrar el refresh token ESPECÍFICO usado (no todos los del usuario)
		// Necesitamos un DeleteByID en refreshTokenRepo o DeleteByToken.
		// Como DeleteByUserID borra todos, NO LO USAMOS AQUÍ.
		// Asumiremos que refreshTokenRepo tiene Delete (por ID) o DeleteByToken.
		// Si no lo tiene, lo borramos con GORM raw o añadimos método.
		// Revisaré refreshTokenRepo después. Por ahora uso borrado manual si es necesario o asumo método.
		// El repositorio actual tenía DeleteByUserID.
		// Voy a borrar el token actual manualmente usando GORM dentro de la Tx si el repo no ayuda.
		if err := tx.Delete(&models.RefreshToken{}, storedToken.ID).Error; err != nil {
			return err
		}

		userIDStr := strconv.FormatUint(storedToken.UserID, 10)
		// Generar nuevos tokens MANTENIENDO el SessionID
		newAccessToken, newRefreshToken, err = s.tokenService.GenerateTokens(userIDStr, sessionIDStr)
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

		// Opcional: Actualizar ExpiresAt de la sesión para extenderla (sliding session)
		// s.sessionRepo.Extend(ctx, tx, sessionUUID, expiresAt)

		return nil
	})

	if err != nil {
		return "", "", err
	}

	return newAccessToken, newRefreshToken, nil
}

// ────────────────────────────────────────────────
// LOGOUT
// ────────────────────────────────────────────────
func (s *localAuthService) Logout(ctx context.Context, userID uint64) error {
	// 1. Revocar sesiones (DB + Redis Blacklist)
	if err := s.RevokeUserSessions(ctx, userID); err != nil {
		return err
	}

	// 2. Borrar refresh tokens (limpieza legacy)
	dbWrite := s.TransactionManager.Conn.ConnectGormWrite
	err := s.refreshTokenRepo.DeleteByUserID(ctx, dbWrite, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("error al borrar tokens: %w", err)
	}
	return nil
}

// ────────────────────────────────────────────────
// REVOKE SESSION (ADMIN / IMMEDIATE)
// ────────────────────────────────────────────────
func (s *localAuthService) RevokeSession(ctx context.Context, sessionIDStr string) error {
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return domain.ErrInvalidArgument
	}

	dbRead := s.TransactionManager.Conn.ConnectGormRead
	dbWrite := s.TransactionManager.Conn.ConnectGormWrite
	redisClient := s.TransactionManager.Conn.ConnectRedis

	// 1. Obtener sesión para calcular TTL de Redis
	session, err := s.sessionRepo.GetByID(ctx, dbRead, sessionID)
	if err != nil {
		return err
	}

	// 2. Revocar en DB
	if err := s.sessionRepo.Revoke(ctx, dbWrite, sessionID); err != nil {
		return err
	}

	// 3. Agregar a Blacklist en Redis (si aún es válida)
	if !session.IsBlocked && session.ExpiresAt.After(time.Now()) {
		ttl := time.Until(session.ExpiresAt)
		projectPrefix := os.Getenv("APP_NAME")
		if projectPrefix == "" {
			projectPrefix = "go-fiber-core"
		}
		key := fmt.Sprintf("%s:blacklist:session:%s", projectPrefix, sessionID.String())
		
		lockService := cache.NewRedisLockService(redisClient)
		if err := lockService.Set(ctx, key, "revoked", ttl); err != nil {
			fmt.Printf("Error setting redis blacklist: %v\n", err)
			// No fallamos la request si Redis falla, pero logueamos
		}
	}

	return nil
}

func (s *localAuthService) RevokeUserSessions(ctx context.Context, userID uint64) error {
	dbRead := s.TransactionManager.Conn.ConnectGormRead
	dbWrite := s.TransactionManager.Conn.ConnectGormWrite
	redisClient := s.TransactionManager.Conn.ConnectRedis

	// 1. Obtener sesiones activas para bloquearlas en Redis
	sessions, err := s.sessionRepo.GetActiveSessionsByUserID(ctx, dbRead, userID)
	if err != nil {
		return err
	}

	// 2. Blacklist en Redis
	lockService := cache.NewRedisLockService(redisClient)
	for _, sess := range sessions {
		ttl := time.Until(sess.ExpiresAt)
		if ttl > 0 {
			projectPrefix := os.Getenv("APP_NAME")
			if projectPrefix == "" {
				projectPrefix = "go-fiber-core"
			}
			key := fmt.Sprintf("%s:blacklist:session:%s", projectPrefix, sess.ID)
			// Pipeline podría ser mejor aquí, pero iterar es simple
			_ = lockService.Set(ctx, key, "revoked", ttl)
		}
	}

	// 3. Revocar en DB
	return s.sessionRepo.RevokeAllByUserID(ctx, dbWrite, userID)
}

func (s *localAuthService) RevokeAllSessions(ctx context.Context) error {
	dbRead := s.TransactionManager.Conn.ConnectGormRead
	dbWrite := s.TransactionManager.Conn.ConnectGormWrite
	redisClient := s.TransactionManager.Conn.ConnectRedis

	// 1️⃣ Obtener TODAS las sesiones activas
	var sessions []models.Session
	err := dbRead.WithContext(ctx).
		Where("is_blocked = ? AND expires_at > ?", false, time.Now()).
		Find(&sessions).Error
	if err != nil {
		return err
	}

	lockService := cache.NewRedisLockService(redisClient)

	projectPrefix := os.Getenv("APP_NAME")
	if projectPrefix == "" {
		projectPrefix = "go-fiber-core"
	}

	// 2️⃣ Blacklist por SID (igual que revoke-user)
	for _, sess := range sessions {
		ttl := time.Until(sess.ExpiresAt)
		if ttl > 0 {
			key := fmt.Sprintf("%s:blacklist:session:%s", projectPrefix, sess.ID)
			_ = lockService.Set(ctx, key, "revoked", ttl)
		}
	}

	// 3️⃣ Revocar TODO en DB
	return s.sessionRepo.RevokeAll(ctx, dbWrite)
}
// ────────────────────────────────────────────────
// GET ACTIVE SESSIONS PAGINATED
// ────────────────────────────────────────────────
func (s *localAuthService) GetActiveSessions(ctx context.Context, req dtos.PaginationRequest) (*dtos.PaginationResponse[models.Session], error) {
	dbRead := s.TransactionManager.Conn.ConnectGormRead
	return s.sessionRepo.GetActiveSessionsPaginated(ctx, dbRead, req)
}
