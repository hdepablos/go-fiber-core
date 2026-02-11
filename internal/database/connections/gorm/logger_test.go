package gorm

import (
	"bytes"
	"context"
	"log"
	"os"
	"strings"
	"testing"
	"time"
)

// TestLoggerBehavior verifica el comportamiento del logger según variables de entorno
func TestLoggerBehavior(t *testing.T) {
	// Guardar estado original
	origAppEnv := os.Getenv("APP_ENV")
	origDbLogLevel := os.Getenv("DB_LOG_LEVEL")
	defer func() {
		os.Setenv("APP_ENV", origAppEnv)
		os.Setenv("DB_LOG_LEVEL", origDbLogLevel)
	}()

	tests := []struct {
		name          string
		appEnv        string
		dbLogLevel    string
		expectContent string // Texto esperado en el log
		expectEmpty   bool   // Si se espera que NO haya log
	}{
		{
			name:          "Local default (APP_ENV=local, DB_LOG_LEVEL unset) -> Should log SQL",
			appEnv:        "local",
			dbLogLevel:    "",
			expectContent: "SELECT * FROM users",
			expectEmpty:   false,
		},
		{
			name:          "Production default (APP_ENV=production, DB_LOG_LEVEL unset) -> Should NOT log SQL",
			appEnv:        "production",
			dbLogLevel:    "",
			expectContent: "",
			expectEmpty:   true,
		},
		{
			name:          "Production with override (APP_ENV=production, DB_LOG_LEVEL=info) -> Should log SQL",
			appEnv:        "production",
			dbLogLevel:    "info",
			expectContent: "SELECT * FROM users",
			expectEmpty:   false,
		},
		{
			name:          "Local with silent (APP_ENV=local, DB_LOG_LEVEL=silent) -> Should NOT log SQL",
			appEnv:        "local",
			dbLogLevel:    "silent",
			expectContent: "",
			expectEmpty:   true,
		},
		{
			name:          "Local with error (APP_ENV=local, DB_LOG_LEVEL=error) -> Should NOT log SQL (only errors)",
			appEnv:        "local",
			dbLogLevel:    "error",
			expectContent: "", // Normal query should not log
			expectEmpty:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("APP_ENV", tt.appEnv)
			os.Setenv("DB_LOG_LEVEL", tt.dbLogLevel)

			// Buffer para capturar la salida
			var buf bytes.Buffer
			mockWriter := log.New(&buf, "", 0)

			// Obtener logger configurado
			gormLogger := getGormLogger(mockWriter)

			// Simular una query
			ctx := context.Background()
			begin := time.Now()
			sql := "SELECT * FROM users"
			rows := int64(1)
			
			// Llamar a Trace (que es lo que hace GORM después de una query)
			// Simulamos que tardó 10ms
			gormLogger.Trace(ctx, begin, func() (string, int64) {
				return sql, rows
			}, nil)

			output := buf.String()

			if tt.expectEmpty {
				if strings.TrimSpace(output) != "" {
					t.Errorf("Esperaba log vacío, pero obtuve: %s", output)
				}
			} else {
				if !strings.Contains(output, tt.expectContent) {
					t.Errorf("Esperaba que el log contuviera '%s', pero obtuve: %s", tt.expectContent, output)
				}
			}
		})
	}
}
