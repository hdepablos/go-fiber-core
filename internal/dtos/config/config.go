package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// --- 1. DEFINICIÓN DE STRUCTS ---

type AppConfig struct {
	App                 App                 `mapstructure:"app"`
	Server              Server              `mapstructure:"server"`
	JWTConfig           JWTConfig           `mapstructure:"jwt"`
	MultiDatabaseConfig MultiDatabaseConfig `mapstructure:"database"`
	Redis               Redis               `mapstructure:"redis"`
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
func NewAppConfig(configPath string) (*AppConfig, error) {
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

	// Expandir variables de entorno (${VAR})
	for _, key := range v.AllKeys() {
		value := v.GetString(key)
		fmt.Println("esto es el valor")
		fmt.Println(value)
		if strings.Contains(value, "${") {

			fmt.Println("Entro xxxxx")
			fmt.Println("Entro")
			fmt.Println(value)

			v.Set(key, os.ExpandEnv(value))
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
