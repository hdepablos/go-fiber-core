package dtos

import (
	"go-fiber-core/internal/dtos/connect"

	"github.com/spf13/viper"
)

type DatabaseHandler struct {
	Config  *viper.Viper
	Connect connect.ConnectDTO
}
