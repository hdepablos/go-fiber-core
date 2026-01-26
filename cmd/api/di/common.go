package di

import (
	"go-fiber-core/internal/dtos/config"
)

func initializeCommon(configPath string) (*config.AppConfig, func(), error) {
	cfg, err := config.NewAppConfig(configPath)
	if err != nil {
		return nil, nil, err
	}

	cleanup := func() {
		// cerrar db, redis, etc
	}

	return cfg, cleanup, nil
}
