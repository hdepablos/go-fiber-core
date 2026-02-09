package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

var (
	loggers sync.Map // key: use-case, value: *zap.Logger
)

// Global constants/vars for service identity
var (
	ServiceName = os.Getenv("SERVICE_NAME")
	Version     = os.Getenv("VERSION")
	Developer   = os.Getenv("DEVELOPER")
)

func init() {
	if ServiceName == "" {
		ServiceName = "go-fiber-core" // Default fallback
	}
	if Version == "" {
		Version = "1.0.0" // Default fallback
	}
}

func GetLogger(name string) *zap.Logger {
	// Chequear si ya existe
	if logger, ok := loggers.Load(name); ok {
		return logger.(*zap.Logger)
	}

	// Si no existe, lo crea y lo guarda
	newLogger := createLogger(name)
	loggers.Store(name, newLogger)
	return newLogger
}

func createLogger(name string) *zap.Logger {
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "local" // Default to local for safety if not set, or change to production if preferred
	}

	var writer zapcore.WriteSyncer

	// Configuración de Encoder (JSON Estructurado)
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:      "timestamp",
		LevelKey:     "level",
		NameKey:      "logger",
		CallerKey:    "caller",
		MessageKey:   "message",
		StacktraceKey: "stacktrace",
		LineEnding:   zapcore.DefaultLineEnding,
		EncodeLevel:  zapcore.CapitalLevelEncoder,
		EncodeTime:   zapcore.ISO8601TimeEncoder, // 2006-01-02T15:04:05.000Z
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller: zapcore.ShortCallerEncoder,
	}

	// Lógica de Salida según Entorno
	if strings.ToLower(appEnv) == "local" {
		// --- MODO LOCAL: Archivos (Lumberjack) ---
		now := time.Now().Format("2006-01-02")
		logDir := "pkg/logs"
		
		// Asegurar que el directorio padre existe
		// Si 'name' contiene slashes (ej: "auth/login"), filepath.Join lo manejará, 
		// pero debemos asegurar que el directorio exista.
		logPath := filepath.Join(logDir, fmt.Sprintf("%s-%s.log", name, now))
		dir := filepath.Dir(logPath)
		
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			// Fallback a Stdout si falla el FS
			fmt.Fprintf(os.Stderr, "Error creating log directory %s: %v. Falling back to Stdout.\n", dir, err)
			writer = zapcore.AddSync(os.Stdout)
		} else {
			writer = zapcore.AddSync(&lumberjack.Logger{
				Filename:   logPath,
				MaxSize:    50, // MB
				MaxBackups: 7,
				MaxAge:     30,   // días
				Compress:   true, // gzip
			})
		}
	} else {
		// --- MODO NUBE/PROD: Stdout (CloudWatch) ---
		// En Lambda/Docker, escribir a stdout es la mejor práctica.
		writer = zapcore.Lock(os.Stdout)
	}

	// Determinar nivel de log
	logLevelStr := os.Getenv("LOG_LEVEL")
	var logLevel zapcore.Level
	
	if logLevelStr != "" {
		// Intentar parsear el nivel configurado explícitamente
		if err := logLevel.UnmarshalText([]byte(logLevelStr)); err != nil {
			logLevel = zapcore.InfoLevel // Fallback seguro
		}
	} else {
		// Default inteligente según entorno
		if strings.ToLower(appEnv) == "local" {
			logLevel = zapcore.DebugLevel // Local: Ver todo (Debug)
		} else {
			logLevel = zapcore.InfoLevel  // Prod: Solo Info y superior (Ignora Debug)
		}
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		writer,
		logLevel,
	)

	// Campos base que siempre queremos (Service, Version)
	// 'trace_id' se debe inyectar dinámicamente en cada log
	initialFields := []zap.Field{
		zap.String("service", ServiceName),
		zap.String("version", Version),
	}

	// Solo agregar el campo developer si estamos en entorno LOCAL.
	// Esto protege contra configuraciones accidentales en Producción.
	// Nota: appEnv ya fue obtenida al inicio de createLogger, pero aquí la volvemos a obtener por claridad
	// o usamos la variable local si estuviera disponible en este scope (pero initialFields está al final).
	// Simplemente re-leemos o usamos la variable appEnv definida arriba si estuviera en scope.
	// Como appEnv se definió al principio de la función, podemos reusarla si no hay shadowing,
	// pero para evitar líos de scope en este bloque, simplemente verificamos os.Getenv directo o reutilizamos sin :=
	
	// La variable appEnv ya existe en la función createLogger (línea 48).
	if Developer != "" && (strings.ToLower(os.Getenv("APP_ENV")) == "local" || os.Getenv("APP_ENV") == "") {
		initialFields = append(initialFields, zap.String("developer", Developer))
	}

	return zap.New(core, zap.AddCaller()).With(initialFields...)
}

// Helpers para estructura

// ContextFields genera el campo "context" estructurado
func ContextFields(reqID, batchID, handler string) zap.Field {
	ctxMap := make(map[string]interface{})
	if reqID != "" {
		ctxMap["request_id"] = reqID
	}
	if batchID != "" {
		ctxMap["batch_id"] = batchID
	}
	if handler != "" {
		ctxMap["handler"] = handler
	}
	return zap.Any("context", ctxMap)
}
