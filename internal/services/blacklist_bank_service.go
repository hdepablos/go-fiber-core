package services

// Para este ejemplo, la lista negra está definida en el código.
// En una aplicación real, esto podría venir de la base de datos o de un archivo de configuración.
var blacklistedBankCodes = map[string]bool{
	"0001": true,
	"0003": true,
	"0005": true,
}

// IBlacklistBankService define la interfaz para nuestro servicio.
type IBlacklistBankService interface {
	IsEntityCodeBlacklisted(entityCode string) bool
}

// NewBlacklistBankService crea una nueva instancia del servicio.
func NewBlacklistBankService() IBlacklistBankService {
	return &blacklistBankService{}
}

// blacklistBankService es la implementación concreta de la interfaz.
type blacklistBankService struct{}

// IsEntityCodeBlacklisted verifica si un código de entidad está en la lista negra.
func (s *blacklistBankService) IsEntityCodeBlacklisted(entityCode string) bool {
	// Usar un mapa para la búsqueda es más eficiente que recorrer un slice.
	_, exists := blacklistedBankCodes[entityCode]
	return exists
}
