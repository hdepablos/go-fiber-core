package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	lumberjack "gopkg.in/natefinch/lumberjack.v2" // ✅ Esta línea soluciona el error
)

var (
	loggers sync.Map // key: use-case, value: *zap.Logger
)

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
	now := time.Now().Format("2006-01-02")
	logDir := "pkg/logs"
	if err := os.MkdirAll(logDir, os.ModePerm); err != nil {
		log.Fatalf("error creating log directory %s: %v", logDir, err)
	}

	logPath := filepath.Join(logDir, fmt.Sprintf("%s-%s.log", name, now))

	writer := zapcore.AddSync(&lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    50, // MB
		MaxBackups: 7,
		MaxAge:     30,   // días
		Compress:   true, // gzip
	})

	encoder := zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		TimeKey:      "timestamp",
		LevelKey:     "level",
		MessageKey:   "message",
		CallerKey:    "caller",
		EncodeTime:   customTimeEncoder,
		EncodeLevel:  zapcore.CapitalLevelEncoder,
		EncodeCaller: zapcore.ShortCallerEncoder,
	})

	core := zapcore.NewCore(encoder, writer, zapcore.DebugLevel)

	return zap.New(core, zap.AddCaller())
}

func customTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05"))
}
