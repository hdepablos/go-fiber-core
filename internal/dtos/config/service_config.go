package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// ServiceConfig representa solo las partes de configuración necesarias
// para servicios o comandos específicos (por ejemplo, Redis y Backoffice).
type ServiceConfig struct {
	Redis      Redis     // Configuración del cliente Redis
	Backoffice ApiConfig // Configuración de la API Backoffice
}

// NewServiceConfig carga solo las secciones Redis y Backoffice del archivo de configuración.
func NewServiceConfig(configPath string) (*ServiceConfig, error) {
	v := viper.New()
	v.SetConfigFile(configPath)
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error al leer archivo de configuración '%s': %w", configPath, err)
	}

	// Expande variables de entorno dentro del YAML
	for _, key := range v.AllKeys() {
		value := v.GetString(key)
		if strings.Contains(value, "${") {
			v.Set(key, os.ExpandEnv(value))
		}
	}

	// Creamos un objeto temporal para leer solo las secciones necesarias
	cfg := &ServiceConfig{}

	// Secciones específicas
	if err := v.UnmarshalKey("redis", &cfg.Redis); err != nil {
		return nil, fmt.Errorf("error al leer sección redis: %w", err)
	}

	if err := v.UnmarshalKey("apis.backoffice", &cfg.Backoffice); err != nil {
		return nil, fmt.Errorf("error al leer sección apis.backoffice: %w", err)
	}

	return cfg, nil
}
