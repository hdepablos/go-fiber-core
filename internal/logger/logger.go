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

func GetLoggerToFile(name string, filePath string) *zap.Logger {
	key := fmt.Sprintf("%s|file:%s", name, filePath)
	if logger, ok := loggers.Load(key); ok {
		return logger.(*zap.Logger)
	}

	newLogger := createLoggerToFile(name, filePath)
	loggers.Store(key, newLogger)
	return newLogger
}

func ResolveProjectPath(relPath string) string {
	if filepath.IsAbs(relPath) {
		return relPath
	}

	cwd, err := os.Getwd()
	if err != nil {
		abs, err2 := filepath.Abs(relPath)
		if err2 != nil {
			return relPath
		}
		return abs
	}

	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.Join(dir, relPath)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	abs, err := filepath.Abs(relPath)
	if err != nil {
		return relPath
	}
	return abs
}

func createLogger(name string) *zap.Logger {
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "local" // Default to local for safety if not set, or change to production if preferred
	}

	var writers []zapcore.WriteSyncer

	// Configuración de Encoder (JSON Estructurado)
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder, // 2006-01-02T15:04:05.000Z
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	mode := strings.ToLower(os.Getenv("LOG_OUTPUT"))
	fileNeeded := mode == "file" || mode == "both" || (mode == "" && strings.ToLower(appEnv) == "local")
	stdoutNeeded := mode == "stdout" || mode == "both" || (mode == "" && strings.ToLower(appEnv) != "local")

	if fileNeeded {
		now := time.Now().Format("2006-01-02")
		logDir := resolveWritablePath("pkg/logs")
		logPath := filepath.Join(logDir, fmt.Sprintf("%s-%s.log", name, now))
		dir := filepath.Dir(logPath)
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating log directory %s: %v. Falling back to Stdout.\n", dir, err)
			writers = append(writers, zapcore.AddSync(os.Stdout))
		} else {
			writers = append(writers, zapcore.AddSync(&lumberjack.Logger{
				Filename:   logPath,
				MaxSize:    50,
				MaxBackups: 7,
				MaxAge:     30,
				Compress:   true,
			}))
		}
	}
	if stdoutNeeded {
		writers = append(writers, zapcore.Lock(os.Stdout))
	}
	if len(writers) == 0 {
		writers = append(writers, zapcore.Lock(os.Stdout))
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
			logLevel = zapcore.InfoLevel // Prod: Solo Info y superior (Ignora Debug)
		}
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.NewMultiWriteSyncer(writers...),
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

func createLoggerToFile(name string, filePath string) *zap.Logger {
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "local"
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	mode := strings.ToLower(os.Getenv("LOG_OUTPUT"))
	stdoutNeeded := mode == "stdout" || mode == "both" || (mode == "" && strings.ToLower(appEnv) != "local")

	filePath = resolveWritablePath(filePath)
	dir := filepath.Dir(filePath)
	var writers []zapcore.WriteSyncer
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating log directory %s: %v. Falling back to Stdout.\n", dir, err)
		writers = append(writers, zapcore.AddSync(os.Stdout))
	} else {
		writers = append(writers, zapcore.AddSync(&lumberjack.Logger{
			Filename:   filePath,
			MaxSize:    50,
			MaxBackups: 7,
			MaxAge:     30,
			Compress:   true,
		}))
	}
	if stdoutNeeded {
		writers = append(writers, zapcore.Lock(os.Stdout))
	}
	if len(writers) == 0 {
		writers = append(writers, zapcore.Lock(os.Stdout))
	}

	logLevelStr := os.Getenv("LOG_LEVEL")
	var logLevel zapcore.Level
	if logLevelStr != "" {
		if err := logLevel.UnmarshalText([]byte(logLevelStr)); err != nil {
			logLevel = zapcore.InfoLevel
		}
	} else {
		if strings.ToLower(appEnv) == "local" {
			logLevel = zapcore.DebugLevel
		} else {
			logLevel = zapcore.InfoLevel
		}
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.NewMultiWriteSyncer(writers...),
		logLevel,
	)

	initialFields := []zap.Field{
		zap.String("service", ServiceName),
		zap.String("version", Version),
	}

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

func resolveWritablePath(path string) string {
	if path == "" {
		return path
	}

	taskRoot := os.Getenv("LAMBDA_TASK_ROOT")
	if taskRoot == "" {
		taskRoot = "/var/task"
	}

	isLambda := os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" || os.Getenv("AWS_EXECUTION_ENV") != "" || os.Getenv("LAMBDA_TASK_ROOT") != ""
	if !isLambda {
		return path
	}

	if filepath.IsAbs(path) {
		if strings.HasPrefix(path, taskRoot+string(os.PathSeparator)) {
			rel := strings.TrimPrefix(path, taskRoot+string(os.PathSeparator))
			return filepath.Join(os.TempDir(), rel)
		}
		return path
	}

	return filepath.Join(os.TempDir(), path)
}
