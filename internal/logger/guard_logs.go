package logger

import "go.uber.org/zap"

const (
	LogTypeRedisGuard     = "redis_guard"
	LogTypeRateLimitGuard = "rate_limit_guard"
	LogTypeExecutionGuard = "execution_guard"
)

func RedisGuardLogger() *zap.Logger {
	return GetLogger("redis-guard").With(zap.String("log_type", LogTypeRedisGuard))
}

func RateLimitGuardLogger() *zap.Logger {
	return GetLogger("rate-limit-guard").With(zap.String("log_type", LogTypeRateLimitGuard))
}

func ExecutionGuardLogger() *zap.Logger {
	return GetLogger("execution-guard").With(zap.String("log_type", LogTypeExecutionGuard))
}

func LogRedisError(operation string, err error, fields ...zap.Field) {
	if err == nil {
		return
	}
	baseFields := []zap.Field{
		zap.String("event_type", "redis_operation_error"),
		zap.String("operation", operation),
		zap.Error(err),
	}
	RedisGuardLogger().Error("redis operation failed", append(baseFields, fields...)...)
}

func LogInternalRateLimit(eventType string, err error, fields ...zap.Field) {
	if err == nil {
		return
	}
	baseFields := []zap.Field{
		zap.String("event_type", eventType),
		zap.String("scope", "internal"),
		zap.Error(err),
	}
	RateLimitGuardLogger().Warn("internal rate limit triggered", append(baseFields, fields...)...)
}

func LogExternalHTTPRateLimit(source string, statusCode int, fields ...zap.Field) {
	baseFields := []zap.Field{
		zap.String("event_type", "external_http_429"),
		zap.String("scope", "external"),
		zap.String("source", source),
		zap.Int("status_code", statusCode),
	}
	RateLimitGuardLogger().Warn("external dependency returned 429", append(baseFields, fields...)...)
}

func LogExternalDependencyError(source string, err error, fields ...zap.Field) {
	if err == nil {
		return
	}
	baseFields := []zap.Field{
		zap.String("event_type", "external_dependency_error"),
		zap.String("scope", "external"),
		zap.String("source", source),
		zap.Error(err),
	}
	RateLimitGuardLogger().Error("external dependency request failed", append(baseFields, fields...)...)
}

func LogExternalDependencyTimeout(source string, err error, fields ...zap.Field) {
	if err == nil {
		return
	}
	baseFields := []zap.Field{
		zap.String("event_type", "external_dependency_timeout"),
		zap.String("scope", "external"),
		zap.String("source", source),
		zap.Error(err),
	}
	RateLimitGuardLogger().Warn("external dependency request timed out", append(baseFields, fields...)...)
}

func LogExecutionGuard(eventType string, fields ...zap.Field) {
	baseFields := []zap.Field{
		zap.String("event_type", eventType),
	}
	ExecutionGuardLogger().Warn("execution guard triggered", append(baseFields, fields...)...)
}
