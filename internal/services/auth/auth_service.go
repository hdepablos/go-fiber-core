package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
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
	"go-fiber-core/internal/logger"
	"go-fiber-core/internal/models"
	refreshTokenRepo "go-fiber-core/internal/repositories/refreshtoken"
	sessionRepo "go-fiber-core/internal/repositories/session"
	userRepo "go-fiber-core/internal/repositories/user"
	"go-fiber-core/internal/services"
	"go-fiber-core/internal/services/authlog"
	"go-fiber-core/internal/services/cache"
	menuService "go-fiber-core/internal/services/menu"

	redis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// authService es la implementación de la interfaz AuthService.
type localAuthService struct {
	services.TransactionManager
	userReader       userRepo.UserReader
	userWriter       userRepo.UserWriter
	refreshTokenRepo refreshTokenRepo.RefreshTokenRepository
	sessionRepo      sessionRepo.SessionRepository
	tokenService     TokenService
	menuReader       menuService.MenuReaderService
	authLog          authlog.AuthLogService
}

// NewAuthService crea una nueva instancia del servicio de autenticación.
// Decide qué implementación usar basándose en la configuración (Local vs Cognito).
func NewAuthService(
	userReader userRepo.UserReader,
	userWriter userRepo.UserWriter,
	refreshTokenRepo refreshTokenRepo.RefreshTokenRepository,
	sessionRepo sessionRepo.SessionRepository,
	tokenService TokenService,
	menuReader menuService.MenuReaderService,
	authLogService authlog.AuthLogService,
	connect *connect.ConnectDTO,
) AuthService {
	// Si AUTH_PROVIDER == "cognito", retornamos la implementación de Cognito.
	if os.Getenv("AUTH_PROVIDER") == "cognito" {
		return NewCognitoAuthService()
	}

	return &localAuthService{
		TransactionManager: services.NewTransactionManager(connect),
		userReader:         userReader,
		userWriter:         userWriter,
		refreshTokenRepo:   refreshTokenRepo,
		sessionRepo:        sessionRepo,
		tokenService:       tokenService,
		menuReader:         menuReader,
		authLog:            authLogService,
	}
}

func (s *localAuthService) createSessionAndTokens(ctx context.Context, userID uint64, userAgent, clientIP string) (accessToken string, refreshToken string, sessionID string, err error) {
	sessionUUID := uuid.New()

	accessToken, refreshToken, err = s.tokenService.GenerateTokens(strconv.FormatUint(userID, 10), sessionUUID.String())
	if err != nil {
		return "", "", "", errors.New("error al generar tokens")
	}

	err = s.TransactionManager.ExecuteTx(ctx, func(tx *gorm.DB) error {
		expiresAt := time.Now().Add(7 * 24 * time.Hour)
		newSession := &models.Session{
			ID:        sessionUUID,
			UserID:    userID,
			UserAgent: userAgent,
			ClientIP:  clientIP,
			ExpiresAt: expiresAt,
			IsBlocked: false,
		}
		if err := s.sessionRepo.Create(ctx, tx, newSession); err != nil {
			return fmt.Errorf("error al crear sesión: %w", err)
		}

		newRefreshToken := &models.RefreshToken{
			UserID:    userID,
			Token:     refreshToken,
			ExpiresAt: expiresAt,
		}
		if err := s.refreshTokenRepo.Create(ctx, tx, newRefreshToken); err != nil {
			return errors.New("error al guardar refresh token")
		}
		return nil
	})
	if err != nil {
		return "", "", "", err
	}

	return accessToken, refreshToken, sessionUUID.String(), nil
}

func (s *localAuthService) buildRolesAndMenu(ctx context.Context, user *models.User) (roleIDs []uint64, roleNames []string, menuItems []responses.MenuItemResponse, err error) {
	for _, r := range user.Roles {
		roleIDs = append(roleIDs, r.ID)
		roleNames = append(roleNames, r.Name)
	}

	menuItems, err = s.menuReader.GetMenuByUser(ctx, user.ID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error al obtener el menú: %w", err)
	}
	if len(menuItems) == 0 {
		return nil, nil, nil, domain.ErrNoMenuAccess
	}

	return roleIDs, roleNames, menuItems, nil
}

// ────────────────────────────────────────────────
// LOGIN
// ────────────────────────────────────────────────
func (s *localAuthService) Login(ctx context.Context, req requests.LoginRequest, userAgent, clientIP, origin, requestID string) (*responses.LoginResponse, error) {
	dbRead := s.TransactionManager.Conn.ConnectGormRead

	// 1️⃣ Buscar usuario por email, incluyendo Roles
	user, err := s.userReader.GetByEmailWithRoles(ctx, dbRead, req.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.tryLogLoginFailed(ctx, nil, req.Email, authlog.FailureUserNotFound, clientIP, userAgent, origin, requestID)
			return nil, domain.ErrAuthentication
		}
		s.tryLogLoginFailed(ctx, nil, req.Email, authlog.FailureInternalError, clientIP, userAgent, origin, requestID)
		return nil, fmt.Errorf("error al buscar usuario: %w", err)
	}

	if !user.IsActive {
		s.tryLogLoginFailed(ctx, &user.ID, user.Email, authlog.FailureUserInactive, clientIP, userAgent, origin, requestID)
		return nil, domain.ErrAuthentication
	}

	// 2️⃣ Verificar contraseña
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		s.tryLogLoginFailed(ctx, &user.ID, user.Email, authlog.FailureWrongPassword, clientIP, userAgent, origin, requestID)
		return nil, domain.ErrAuthentication
	}

	roleIDs, roleNames, menuItems, err := s.buildRolesAndMenu(ctx, user)
	if err != nil {
		if errors.Is(err, domain.ErrNoMenuAccess) {
			s.tryLogLoginFailed(ctx, &user.ID, user.Email, authlog.FailureNoMenuAccess, clientIP, userAgent, origin, requestID)
			return nil, err
		}
		s.tryLogLoginFailed(ctx, &user.ID, user.Email, authlog.FailureInternalError, clientIP, userAgent, origin, requestID)
		return nil, err
	}

	accessToken, refreshToken, _, err := s.createSessionAndTokens(ctx, user.ID, userAgent, clientIP)
	if err != nil {
		s.tryLogLoginFailed(ctx, &user.ID, user.Email, authlog.FailureInternalError, clientIP, userAgent, origin, requestID)
		return nil, err
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

	s.tryLogLoginSuccess(ctx, &user.ID, user.Email, clientIP, userAgent, origin, requestID)
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
func (s *localAuthService) Logout(ctx context.Context, userID uint64, userAgent, clientIP, origin, requestID string) error {
	// 1. Revocar sesiones (DB + Redis Blacklist)
	if err := s.RevokeUserSessions(ctx, userID); err != nil {
		return err
	}

	// 2. Borrar refresh tokens (limpieza legacy)
	dbWrite := s.TransactionManager.Conn.ConnectGormWrite
	err := s.refreshTokenRepo.DeleteByUserID(ctx, dbWrite, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.tryLogLogout(ctx, &userID, nil, clientIP, userAgent, origin, requestID)
			return nil
		}
		return fmt.Errorf("error al borrar tokens: %w", err)
	}
	s.tryLogLogout(ctx, &userID, nil, clientIP, userAgent, origin, requestID)
	return nil
}

func (s *localAuthService) tryLogLoginSuccess(ctx context.Context, userID *uint64, email string, clientIP, userAgent, origin, requestID string) {
	if s == nil || s.authLog == nil {
		return
	}
	log := logger.GetLoggerToFile("auth", logger.ResolveProjectPath("pkg/logs/auth.log")).With(zap.String("component", "auth_audit"))
	email = strings.TrimSpace(email)
	var emailSnapshot *string
	if email != "" {
		emailSnapshot = &email
	}
	var reqID *string
	if strings.TrimSpace(requestID) != "" {
		v := strings.TrimSpace(requestID)
		reqID = &v
	}
	var o *string
	if strings.TrimSpace(origin) != "" {
		v := strings.TrimSpace(origin)
		o = &v
	}
	if err := s.authLog.Log(ctx, authlog.Entry{
		UserID:        userID,
		EmailSnapshot: emailSnapshot,
		EventType:     authlog.EventLoginSuccess,
		IPAddress:     clientIP,
		UserAgent:     userAgent,
		RequestID:     reqID,
		Origin:        o,
	}); err != nil {
		log.Warn("authentication log insert failed", zap.String("event_type", string(authlog.EventLoginSuccess)), zap.Error(err))
	}
}

func (s *localAuthService) tryLogLoginFailed(ctx context.Context, userID *uint64, email string, reason authlog.FailureReason, clientIP, userAgent, origin, requestID string) {
	if s == nil || s.authLog == nil {
		return
	}
	log := logger.GetLoggerToFile("auth", logger.ResolveProjectPath("pkg/logs/auth.log")).With(zap.String("component", "auth_audit"))
	email = strings.TrimSpace(email)
	var emailSnapshot *string
	if email != "" {
		emailSnapshot = &email
	}
	r := reason
	var reqID *string
	if strings.TrimSpace(requestID) != "" {
		v := strings.TrimSpace(requestID)
		reqID = &v
	}
	var o *string
	if strings.TrimSpace(origin) != "" {
		v := strings.TrimSpace(origin)
		o = &v
	}
	if err := s.authLog.Log(ctx, authlog.Entry{
		UserID:        userID,
		EmailSnapshot: emailSnapshot,
		EventType:     authlog.EventLoginFailed,
		FailureReason: &r,
		IPAddress:     clientIP,
		UserAgent:     userAgent,
		RequestID:     reqID,
		Origin:        o,
	}); err != nil {
		log.Warn("authentication log insert failed", zap.String("event_type", string(authlog.EventLoginFailed)), zap.String("failure_reason", string(reason)), zap.Error(err))
	}
}

func (s *localAuthService) tryLogLogout(ctx context.Context, userID *uint64, emailSnapshot *string, clientIP, userAgent, origin, requestID string) {
	if s == nil || s.authLog == nil {
		return
	}
	log := logger.GetLoggerToFile("auth", logger.ResolveProjectPath("pkg/logs/auth.log")).With(zap.String("component", "auth_audit"))
	var reqID *string
	if strings.TrimSpace(requestID) != "" {
		v := strings.TrimSpace(requestID)
		reqID = &v
	}
	var o *string
	if strings.TrimSpace(origin) != "" {
		v := strings.TrimSpace(origin)
		o = &v
	}
	if err := s.authLog.Log(ctx, authlog.Entry{
		UserID:        userID,
		EmailSnapshot: emailSnapshot,
		EventType:     authlog.EventLogout,
		IPAddress:     clientIP,
		UserAgent:     userAgent,
		RequestID:     reqID,
		Origin:        o,
	}); err != nil {
		log.Warn("authentication log insert failed", zap.String("event_type", string(authlog.EventLogout)), zap.Error(err))
	}
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

// ────────────────────────────────────────────────
// GOOGLE OAUTH2
// ────────────────────────────────────────────────
func (s *localAuthService) GoogleAuthURL(state string) (string, error) {
	log := logger.GetLoggerToFile("auth", logger.ResolveProjectPath("pkg/logs/auth.log")).With(zap.String("component", "auth_google_service"))
	client, err := newGoogleOAuthClientFromEnv()
	if err != nil {
		log.Error("google oauth: configuración inválida", zap.Error(err))
		return "", err
	}
	return client.AuthCodeURL(state)
}

func (s *localAuthService) GoogleCallbackLogin(ctx context.Context, code, userAgent, clientIP, origin, requestID string) (*responses.GoogleOAuthLoginResponse, error) {
	log := logger.GetLoggerToFile("auth", logger.ResolveProjectPath("pkg/logs/auth.log")).With(zap.String("component", "auth_google_service"), zap.String("ip", clientIP))
	if code == "" {
		log.Warn("google callback login: missing code")
		s.tryLogLoginFailed(ctx, nil, "", authlog.FailureInternalError, clientIP, userAgent, origin, requestID)
		return nil, domain.ErrInvalidArgument
	}

	googleClient, err := newGoogleOAuthClientFromEnv()
	if err != nil {
		log.Error("google callback login: configuración inválida", zap.Error(err))
		s.tryLogLoginFailed(ctx, nil, "", authlog.FailureInternalError, clientIP, userAgent, origin, requestID)
		return nil, err
	}

	log.Info("google callback login: exchanging code", zap.String("ua", userAgent))
	token, err := googleClient.Exchange(ctx, code)
	if err != nil {
		log.Warn("google callback login: exchange failed", zap.Error(err))
		s.tryLogLoginFailed(ctx, nil, "", authlog.FailureOAuthExchangeFailed, clientIP, userAgent, origin, requestID)
		return nil, domain.ErrAuthentication
	}
	if token == nil || token.AccessToken == "" {
		log.Warn("google callback login: missing access token after exchange")
		s.tryLogLoginFailed(ctx, nil, "", authlog.FailureOAuthExchangeFailed, clientIP, userAgent, origin, requestID)
		return nil, domain.ErrAuthentication
	}

	log.Info("google callback login: fetching userinfo")
	info, err := googleClient.FetchUserInfo(ctx, token.AccessToken)
	if err != nil {
		log.Warn("google callback login: userinfo failed", zap.Error(err))
		s.tryLogLoginFailed(ctx, nil, "", authlog.FailureOAuthUserInfoFailed, clientIP, userAgent, origin, requestID)
		return nil, domain.ErrAuthentication
	}
	log.Info(
		"google callback login: userinfo ok",
		zap.String("email", info.Email),
		zap.Bool("verified_email", info.VerifiedEmail),
		zap.String("google_id", info.ID),
		zap.String("name", info.Name),
		zap.String("locale", info.Locale),
	)
	if !info.VerifiedEmail {
		log.Warn("google callback login: email not verified", zap.String("email", info.Email))
		s.tryLogLoginFailed(ctx, nil, info.Email, authlog.FailureEmailNotVerified, clientIP, userAgent, origin, requestID)
		return nil, domain.ErrAuthentication
	}

	allowedDomain := strings.TrimSpace(os.Getenv("GOOGLE_ALLOWED_DOMAIN"))
	if allowedDomain != "" {
		emailLower := strings.ToLower(strings.TrimSpace(info.Email))
		domainLower := strings.ToLower(allowedDomain)
		if !strings.HasSuffix(emailLower, "@"+domainLower) {
			log.Warn("google callback login: email domain not allowed", zap.String("allowed_domain", allowedDomain))
			s.tryLogLoginFailed(ctx, nil, info.Email, authlog.FailureDomainNotAllowed, clientIP, userAgent, origin, requestID)
			return nil, domain.ErrAuthentication
		}
	}

	log = log.With(zap.String("email", info.Email), zap.String("google_id", info.ID))
	dbRead := s.TransactionManager.Conn.ConnectGormRead

	user, err := s.userReader.GetByEmailWithRoles(ctx, dbRead, info.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("google callback login: user not found in database", zap.String("email", info.Email))
			s.tryLogLoginFailed(ctx, nil, info.Email, authlog.FailureUserNotFound, clientIP, userAgent, origin, requestID)
			return nil, domain.ErrNoMenuAccess
		} else {
			log.Error("google callback login: error loading user", zap.Error(err))
			s.tryLogLoginFailed(ctx, nil, info.Email, authlog.FailureInternalError, clientIP, userAgent, origin, requestID)
			return nil, errors.New("error obteniendo usuario")
		}
	}

	if user == nil || !user.IsActive {
		log.Warn("google callback login: user inactive or nil")
		var userID *uint64
		if user != nil {
			userID = &user.ID
		}
		s.tryLogLoginFailed(ctx, userID, info.Email, authlog.FailureUserInactive, clientIP, userAgent, origin, requestID)
		return nil, domain.ErrAuthentication
	}

	roleIDs, roleNames, menuItems, err := s.buildRolesAndMenu(ctx, user)
	if err != nil {
		log.Error("google callback login: error building roles/menu", zap.Error(err), zap.Uint64("user_id", user.ID))
		if errors.Is(err, domain.ErrNoMenuAccess) {
			s.tryLogLoginFailed(ctx, &user.ID, info.Email, authlog.FailureNoMenuAccess, clientIP, userAgent, origin, requestID)
		} else {
			s.tryLogLoginFailed(ctx, &user.ID, info.Email, authlog.FailureInternalError, clientIP, userAgent, origin, requestID)
		}
		return nil, err
	}

	log.Info("google callback login: creating session and jwt", zap.Uint64("user_id", user.ID))
	accessToken, refreshToken, _, err := s.createSessionAndTokens(ctx, user.ID, userAgent, clientIP)
	if err != nil {
		log.Error("google callback login: error creating session/tokens", zap.Error(err), zap.Uint64("user_id", user.ID))
		s.tryLogLoginFailed(ctx, &user.ID, info.Email, authlog.FailureInternalError, clientIP, userAgent, origin, requestID)
		return nil, err
	}

	log.Info("google callback login: success", zap.Uint64("user_id", user.ID), zap.Int("roles", len(roleNames)), zap.Int("menu_items", len(menuItems)))
	s.tryLogLoginSuccess(ctx, &user.ID, info.Email, clientIP, userAgent, origin, requestID)
	return &responses.GoogleOAuthLoginResponse{
		Provider:     "google",
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		UserID:       user.ID,
		UserName:     user.Name,
		RoleIDs:      roleIDs,
		Roles:        roleNames,
		Menu:         menuItems,
		User: responses.GoogleOAuthUser{
			GoogleID:      info.ID,
			Email:         info.Email,
			VerifiedEmail: info.VerifiedEmail,
			Name:          info.Name,
			GivenName:     info.GivenName,
			FamilyName:    info.FamilyName,
			Picture:       info.Picture,
			Locale:        info.Locale,
		},
	}, nil
}

func (s *localAuthService) googleOAuthStateKey(state string) string {
	projectPrefix := os.Getenv("APP_NAME")
	if projectPrefix == "" {
		projectPrefix = "go-fiber-core"
	}
	return fmt.Sprintf("%s:oauth:google:state:%s", projectPrefix, state)
}

func (s *localAuthService) SaveGoogleOAuthState(ctx context.Context, state string) error {
	if state == "" {
		return domain.ErrInvalidArgument
	}
	redisClient := s.TransactionManager.Conn.ConnectRedis
	if redisClient == nil {
		return errors.New("redis no configurado para oauth state")
	}

	key := s.googleOAuthStateKey(state)
	lockService := cache.NewRedisLockService(redisClient)
	return lockService.Set(ctx, key, "1", 10*time.Minute)
}

func (s *localAuthService) ConsumeGoogleOAuthState(ctx context.Context, state string) (bool, error) {
	if state == "" {
		return false, domain.ErrInvalidArgument
	}
	redisClient := s.TransactionManager.Conn.ConnectRedis
	if redisClient == nil {
		return false, errors.New("redis no configurado para oauth state")
	}

	key := s.googleOAuthStateKey(state)
	val, err := redisClient.GetDel(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, err
	}
	return val != "", nil
}

func (s *localAuthService) googleOAuthLoginKey(code string) string {
	projectPrefix := os.Getenv("APP_NAME")
	if projectPrefix == "" {
		projectPrefix = "go-fiber-core"
	}
	return fmt.Sprintf("%s:oauth:google:login:%s", projectPrefix, code)
}

func (s *localAuthService) SaveGoogleOAuthLoginResult(ctx context.Context, code string, result *responses.GoogleOAuthLoginResponse) error {
	if code == "" || result == nil {
		return domain.ErrInvalidArgument
	}
	redisClient := s.TransactionManager.Conn.ConnectRedis
	if redisClient == nil {
		return errors.New("redis no configurado para oauth exchange")
	}

	b, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("error serializando login result: %w", err)
	}

	return redisClient.Set(ctx, s.googleOAuthLoginKey(code), string(b), 2*time.Minute).Err()
}

func (s *localAuthService) ConsumeGoogleOAuthLoginResult(ctx context.Context, code string) (*responses.GoogleOAuthLoginResponse, error) {
	if code == "" {
		return nil, domain.ErrInvalidArgument
	}
	redisClient := s.TransactionManager.Conn.ConnectRedis
	if redisClient == nil {
		return nil, errors.New("redis no configurado para oauth exchange")
	}

	val, err := redisClient.GetDel(ctx, s.googleOAuthLoginKey(code)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, domain.ErrAuthentication
		}
		return nil, err
	}

	var out responses.GoogleOAuthLoginResponse
	if err := json.Unmarshal([]byte(val), &out); err != nil {
		return nil, domain.ErrAuthentication
	}

	return &out, nil
}
