package config

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"go-fiber-core/internal/appconfig"

	"github.com/spf13/viper"
)

// --- 1. DEFINICIÓN DE STRUCTS ---

type AppConfig struct {
	App                 App                 `mapstructure:"app"`
	Server              Server              `mapstructure:"server"`
	JWTConfig           JWTConfig           `mapstructure:"jwt"`
	MultiDatabaseConfig MultiDatabaseConfig `mapstructure:"database"`
	Redis               Redis               `mapstructure:"redis"`
	S3                  S3                  `mapstructure:"s3"`
	EmailConfig         EmailConfig         `mapstructure:"email_config"`
	ApiBackoffice       ApiConfig           `mapstructure:"apis.backoffice"`
	ApiDiscord          ApiConfig           `mapstructure:"apis.discord"`
}

type MultiDatabaseConfig struct {
	Gorm GormConfig `mapstructure:"gorm"`
	Pgx  PgxConfig  `mapstructure:"pgx"`
}

type GormConfig struct {
	Write GormConnectionConfig `mapstructure:"write"`
	Read  GormConnectionConfig `mapstructure:"read"`
}

type PgxConfig struct {
	Write PgxConnectionConfig `mapstructure:"write"`
	Read  PgxConnectionConfig `mapstructure:"read"`
}

type GormConnectionConfig struct {
	Driver                   string `mapstructure:"driver"`
	Host                     string `mapstructure:"host"`
	Port                     int    `mapstructure:"port"`
	Username                 string `mapstructure:"username"`
	Password                 string `mapstructure:"password"`
	Database                 string `mapstructure:"database"`
	Schema                   string `mapstructure:"schema"`
	MaxOpenConns             int    `mapstructure:"max_open_conns"`
	MaxIdleConns             int    `mapstructure:"max_idle_conns"`
	MaxConnLifeTimeInSeconds int    `mapstructure:"max_conn_life_time_in_seconds"`
}

type PgxConnectionConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
	MaxConns int    `mapstructure:"max_conns"`
}

type App struct {
	AppName string `mapstructure:"app_name"`
	AppEnv  string `mapstructure:"app_env"`
}

type Server struct {
	ServerHeader string `mapstructure:"server_header"`
	ServerPort   string `mapstructure:"server_port"`
}

type JWTConfig struct {
	JwtAccessSecret     string        `mapstructure:"jwt_access_secret"`
	JwtRefreshSecret    string        `mapstructure:"jwt_refresh_secret"`
	JwtAccessTtlMinutes time.Duration `mapstructure:"jwt_access_ttl_minutes"`
	JwtRefreshTtlDays   time.Duration `mapstructure:"jwt_refresh_ttl_days"`
}

type Redis struct {
	RedisHost             string `mapstructure:"redis_host"`
	RedisPort             string `mapstructure:"redis_port"`
	RedisPassword         string `mapstructure:"redis_password"`
	RedisDatabase         int    `mapstructure:"redis_database"`
	RedisExpiresInSeconds int    `mapstructure:"redis_expires_in_seconds"`
	RedisPoolSize         int    `mapstructure:"redis_pool_size"`
}

type S3 struct {
	Bucket string `mapstructure:"bucket"`
}

type EmailConfig struct {
	SmtpHost     string `mapstructure:"smtp_host"`
	SmtpPort     int    `mapstructure:"smtp_port"`
	SmtpUsername string `mapstructure:"smtp_username"`
	SmtpPassword string `mapstructure:"smtp_password"`
	SmtpFrom     string `mapstructure:"smtp_from"`
}

type ApiConfig struct {
	Url   string `mapstructure:"url"`
	Token string `mapstructure:"token"`
}

// --- 2. LÓGICA DE CARGA DE CONFIGURACIÓN ---

// NewAppConfig carga y retorna la configuración de la aplicación.
// Soporta múltiples proveedores de secretos (Env, AWS) seleccionados dinámicamente.
func NewAppConfig(configPath string) (*AppConfig, error) {
	// 1. Determinar el proveedor de secretos
	var provider appconfig.SecretProvider
	ctx := context.Background()

	// Si SECRET_PROVIDER == "aws", usamos Secrets Manager.
	// De lo contrario, usamos variables de entorno por defecto.
	if os.Getenv("SECRET_PROVIDER") == "aws" {
		awsProvider, err := appconfig.NewAWSSecretProvider(ctx)
		if err != nil {
			return nil, fmt.Errorf("fallo al inicializar AWS Secret Provider: %w", err)
		}
		provider = awsProvider
	} else {
		provider = appconfig.NewEnvSecretProvider()
	}

	v := viper.New()

	// 🔴 ESTO TIENE QUE IR ANTES
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetConfigFile(configPath)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf(
			"error al cargar el archivo de configuración desde '%s': %w",
			configPath,
			err,
		)
	}

	// Expandir variables usando el Provider seleccionado
	// Viper soporta os.ExpandEnv nativamente, pero para soportar AWS necesitamos
	// iterar manualmente o usar una función personalizada de expansión.
	// Aquí usaremos una estrategia híbrida: Viper carga la estructura base,
	// y luego sobreescribimos valores sensibles si es necesario.
	//
	// Nota: Para simplificar esta implementación inicial, mantenemos la expansión de os.ExpandEnv
	// para el caso local, y en el futuro se puede extender para llamar a provider.GetSecret()
	// en claves específicas marcadas (ej: "aws:secret:prod/db/password").

	// Implementación simple de expansión de variables de entorno (compatible con EnvProvider)
	for _, key := range v.AllKeys() {
		value := v.GetString(key)
		if strings.Contains(value, "${") {
			// Expandir usando os.Expand, pero con nuestro provider
			expanded := os.Expand(value, func(envKey string) string {
				val, err := provider.GetSecret(ctx, envKey)
				if err != nil {
					// Loggear error si es crítico, o retornar vacío
					fmt.Printf("⚠️ Error obteniendo secreto %s: %v\n", envKey, err)
					return ""
				}
				return val
			})
			v.Set(key, expanded)
		}
	}

	cfg := &AppConfig{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf(
			"error al deserializar la configuración principal: %w",
			err,
		)
	}

	return cfg, nil
}
