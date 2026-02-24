package utils

import (
	"errors"
	"time"

	"go-fiber-core/internal/services/serviceconfig/contracts"
)

const (
	BuenosAiresLocationName = "America/Argentina/Buenos_Aires"
	DateTimeLayout          = "2006-01-02 15:04:05"
	DateLayout              = "2006-01-02"
	TimeLayout              = "15:04:05"
)

func GetIntInput(ctx *contracts.ServiceContext, key string) int {
	raw, _ := ctx.GetInputValue(key)
	switch v := raw.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

func GetIntConfig(cfg map[string]any, key string, defaultVal int) int {
	if cfg == nil {
		return defaultVal
	}
	v, ok := cfg[key]
	if !ok {
		return defaultVal
	}
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return defaultVal
	}
}

func getBuenosAiresLocation() *time.Location {
	loc, err := time.LoadLocation(BuenosAiresLocationName)
	if err != nil {
		return time.Local
	}
	return loc
}

func GetDateTimeInput(ctx *contracts.ServiceContext, key string) (time.Time, error) {
	raw, _ := ctx.GetInputValue(key)
	return parseToTime(raw, DateTimeLayout)
}

func GetDateInput(ctx *contracts.ServiceContext, key string) (time.Time, error) {
	raw, _ := ctx.GetInputValue(key)
	return parseToTime(raw, DateLayout)
}

func GetTimeInput(ctx *contracts.ServiceContext, key string) (time.Time, error) {
	raw, _ := ctx.GetInputValue(key)
	return parseToTime(raw, TimeLayout)
}

func GetDateTimeConfig(cfg map[string]any, key string) (time.Time, error) {
	if cfg == nil {
		return time.Time{}, ErrValueNotFound
	}
	raw, ok := cfg[key]
	if !ok {
		return time.Time{}, ErrValueNotFound
	}
	return parseToTime(raw, DateTimeLayout)
}

func GetDateConfig(cfg map[string]any, key string) (time.Time, error) {
	if cfg == nil {
		return time.Time{}, ErrValueNotFound
	}
	raw, ok := cfg[key]
	if !ok {
		return time.Time{}, ErrValueNotFound
	}
	return parseToTime(raw, DateLayout)
}

func GetTimeConfig(cfg map[string]any, key string) (time.Time, error) {
	if cfg == nil {
		return time.Time{}, ErrValueNotFound
	}
	raw, ok := cfg[key]
	if !ok {
		return time.Time{}, ErrValueNotFound
	}
	return parseToTime(raw, TimeLayout)
}

var ErrValueNotFound = errors.New("value not found")

func parseToTime(raw any, layout string) (time.Time, error) {
	switch v := raw.(type) {
	case time.Time:
		return v.In(getBuenosAiresLocation()), nil
	case string:
		if v == "" {
			return time.Time{}, ErrValueNotFound
		}
		loc := getBuenosAiresLocation()
		return time.ParseInLocation(layout, v, loc)
	default:
		return time.Time{}, ErrValueNotFound
	}
}
