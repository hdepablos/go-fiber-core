package product

import (
	"context"
	"fmt"
	"go-fiber-core/internal/dtos/config"
)

//
// ────────────────────────────────────────────────
//  SERVICE #1 → Usa AppConfig completo (versión original)
// ────────────────────────────────────────────────
//

// ProductAPIService expone un método simple que accede a toda la AppConfig.
type ProductAPIService interface {
	PrintRedisConfig(ctx context.Context) error
}

// productAPIService implementa ProductAPIService.
type productAPIService struct {
	appConfig *config.AppConfig
}

// NewProductAPIService crea una nueva instancia inyectando AppConfig completo.
func NewProductAPIService(appConfig *config.AppConfig) ProductAPIService {
	return &productAPIService{
		appConfig: appConfig,
	}
}

// PrintRedisConfig imprime valores de configuración de Redis desde AppConfig.
func (s *productAPIService) PrintRedisConfig(ctx context.Context) error {
	fmt.Println("🚀 Comprobación de acceso a AppConfig desde ProductAPIService")
	fmt.Printf("Redis Host: %s\n", s.appConfig.Redis.RedisHost)
	fmt.Printf("Redis Port: %s\n", s.appConfig.Redis.RedisPort)
	fmt.Printf("Redis DB: %d\n", s.appConfig.Redis.RedisDatabase)
	return nil
}

//
// ────────────────────────────────────────────────
//  SERVICE #2 → Usa ServiceConfig (solo Redis + Backoffice)
// ────────────────────────────────────────────────
//

// ProductServiceConfigService maneja solo Redis y Backoffice.
type ProductServiceConfigService interface {
	PrintConfigs(ctx context.Context) error
}

// productServiceConfigService implementa ProductServiceConfigService.
type productServiceConfigService struct {
	serviceConfig *config.ServiceConfig
}

// NewProductServiceConfigService crea una nueva instancia inyectando solo Redis + Backoffice.
func NewProductServiceConfigService(serviceConfig *config.ServiceConfig) ProductServiceConfigService {
	return &productServiceConfigService{
		serviceConfig: serviceConfig,
	}
}

// PrintConfigs imprime Redis y Backoffice usando ServiceConfig.
func (s *productServiceConfigService) PrintConfigs(ctx context.Context) error {
	fmt.Println("🚀 Comprobación de acceso a ServiceConfig desde ProductServiceConfigService")

	fmt.Println("🔧 Redis Config:")
	fmt.Printf("Host: %s:%s\n", s.serviceConfig.Redis.RedisHost, s.serviceConfig.Redis.RedisPort)
	fmt.Printf("DB: %d\n", s.serviceConfig.Redis.RedisDatabase)
	fmt.Printf("Pool Size: %d\n", s.serviceConfig.Redis.RedisPoolSize)

	fmt.Println("🌐 Backoffice API Config:")
	fmt.Printf("URL: %s\n", s.serviceConfig.Backoffice.Url)
	fmt.Printf("Token: %s\n", s.serviceConfig.Backoffice.Token)

	return nil
}
