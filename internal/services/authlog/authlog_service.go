package authlog

import (
	"context"
	"fmt"
	"strings"

	"go-fiber-core/internal/dtos/connect"
	"go-fiber-core/internal/models"
	authLogRepo "go-fiber-core/internal/repositories/authenticationlog"
)

type EventType string

const (
	EventLoginSuccess EventType = "login_success"
	EventLoginFailed  EventType = "login_failed"
	EventLogout       EventType = "logout"
)

type FailureReason string

const (
	FailureWrongPassword FailureReason = "wrong_password"
	FailureUserNotFound  FailureReason = "user_not_found"
	FailureUserInactive  FailureReason = "user_inactive"
	FailureNoMenuAccess  FailureReason = "no_menu_access"
	FailureEmailNotVerified FailureReason = "email_not_verified"
	FailureDomainNotAllowed  FailureReason = "domain_not_allowed"
	FailureOAuthExchangeFailed FailureReason = "oauth_exchange_failed"
	FailureOAuthUserInfoFailed FailureReason = "oauth_userinfo_failed"
	FailureInternalError FailureReason = "internal_error"
)

type DeviceType string

const (
	DeviceDesktop DeviceType = "desktop"
	DeviceMobile  DeviceType = "mobile"
	DeviceTablet  DeviceType = "tablet"
	DeviceUnknown DeviceType = "unknown"
)

type AuthLogService interface {
	Log(ctx context.Context, entry Entry) error
}

type Entry struct {
	UserID        *uint64
	EmailSnapshot *string
	EventType     EventType
	FailureReason *FailureReason
	IPAddress     string
	UserAgent     string
	Country       *string
	City          *string
	RequestID     *string
	Origin        *string
}

type authLogService struct {
	conn   *connect.ConnectDTO
	writer authLogRepo.AuthenticationLogWriter
}

func NewAuthLogService(conn *connect.ConnectDTO, writer authLogRepo.AuthenticationLogWriter) AuthLogService {
	return &authLogService{
		conn:   conn,
		writer: writer,
	}
}

func (s *authLogService) Log(ctx context.Context, entry Entry) error {
	if s == nil || s.conn == nil || s.conn.ConnectGormWrite == nil {
		return fmt.Errorf("gorm write connection is not initialized")
	}
	if s.writer == nil {
		return fmt.Errorf("authentication log writer is not initialized")
	}

	ua := strings.TrimSpace(entry.UserAgent)
	ip := strings.TrimSpace(entry.IPAddress)
	if ua == "" {
		ua = "unknown"
	}
	if ip == "" {
		ip = "unknown"
	}

	browser := detectBrowser(ua)
	osName := detectOS(ua)
	deviceType := detectDeviceType(ua)

	var failureReason *string
	if entry.FailureReason != nil {
		v := string(*entry.FailureReason)
		failureReason = &v
	}

	log := &models.AuthenticationLog{
		UserID:          entry.UserID,
		EmailSnapshot:   entry.EmailSnapshot,
		EventType:       string(entry.EventType),
		FailureReason:   failureReason,
		IPAddress:       ip,
		UserAgent:       ua,
		Browser:         browser,
		OperatingSystem: osName,
		DeviceType:      string(deviceType),
		Country:         entry.Country,
		City:            entry.City,
		RequestID:       entry.RequestID,
		Origin:          entry.Origin,
	}

	return s.writer.Create(ctx, s.conn.ConnectGormWrite, log)
}

func detectBrowser(ua string) string {
	u := strings.ToLower(ua)
	switch {
	case strings.Contains(u, "edg/") || strings.Contains(u, "edge/"):
		return "Edge"
	case strings.Contains(u, "opr/") || strings.Contains(u, "opera"):
		return "Opera"
	case strings.Contains(u, "chrome/") && !strings.Contains(u, "edg/") && !strings.Contains(u, "opr/"):
		return "Chrome"
	case strings.Contains(u, "safari/") && !strings.Contains(u, "chrome/"):
		return "Safari"
	case strings.Contains(u, "firefox/"):
		return "Firefox"
	default:
		return "Unknown"
	}
}

func detectOS(ua string) string {
	u := strings.ToLower(ua)
	switch {
	case strings.Contains(u, "windows nt"):
		return "Windows"
	case strings.Contains(u, "mac os x") && !strings.Contains(u, "iphone") && !strings.Contains(u, "ipad"):
		return "MacOS"
	case strings.Contains(u, "android"):
		return "Android"
	case strings.Contains(u, "iphone") || strings.Contains(u, "ipad") || strings.Contains(u, "ios"):
		return "iOS"
	case strings.Contains(u, "linux"):
		return "Linux"
	default:
		return "Unknown"
	}
}

func detectDeviceType(ua string) DeviceType {
	u := strings.ToLower(ua)
	switch {
	case strings.Contains(u, "ipad") || strings.Contains(u, "tablet"):
		return DeviceTablet
	case strings.Contains(u, "mobile") || strings.Contains(u, "iphone") || strings.Contains(u, "android"):
		return DeviceMobile
	case u == "" || u == "unknown":
		return DeviceUnknown
	default:
		return DeviceDesktop
	}
}
