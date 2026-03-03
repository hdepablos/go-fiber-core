package queue

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

// AWSService es una estructura simple para manejar la configuración de AWS.
type AWSService struct {
	cfg aws.Config
}

// NewAWSService crea y retorna una nueva instancia de AWSService.
// Se encarga de cargar la configuración de AWS.
func NewAWSService(ctx context.Context) (*AWSService, error) {
	// Carga la configuración por defecto de AWS.
	// Esto buscará las credenciales en el orden estándar:
	// - Variables de entorno (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY)
	// - Archivos de credenciales compartidos (~/.aws/credentials)
	// - Roles de IAM de ECS o EC2
	// - IAM roles for Service Accounts (IRSA) en EKS
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("fallo al cargar la configuración de AWS: %w", err)
	}

	// DEBUG: Imprimir configuración cargada para verificar entorno
	fmt.Printf("AWS Config Loaded - Region: %s, Endpoint: %s\n", cfg.Region, os.Getenv("AWS_ENDPOINT_URL"))
	for _, env := range os.Environ() {
		if len(env) > 4 && env[:4] == "AWS_" {
			fmt.Println("ENV:", env)
		}
	}

	// Si hay una URL de endpoint personalizada en el entorno (ej: LocalStack), asegurar que se use.
	// Aunque el SDK v2 moderno soporta AWS_ENDPOINT_URL, a veces es necesario ser explícito con el BaseEndpoint
	// si la versión del SDK es antigua o hay comportamientos específicos.
	if endpoint := os.Getenv("AWS_ENDPOINT_URL"); endpoint != "" {
		fmt.Printf("⚠️ Usando Custom Endpoint: %s\n", endpoint)
		// Forzamos el BaseEndpoint en la configuración si no se cargó automáticamente
		// Nota: En versiones muy recientes de AWS SDK Go V2, esto es automático, pero esto asegura compatibilidad.
		cfg.BaseEndpoint = aws.String(endpoint)
	}

	return &AWSService{
		cfg: cfg,
	}, nil
}

// GetConfig retorna la configuración de AWS cargada.
// Es el método clave para la reutilización, ya que cualquier otro servicio
// puede llamarlo para obtener la configuración y crear clientes de servicios específicos.
func (s *AWSService) GetConfig() aws.Config {
	return s.cfg
}

// Ejemplo de uso:
// Funciona como un conector para otros servicios de AWS.
//
// Importa la configuración en tu `dlq-consumer`:
//
// import (
// 	"mi_proyecto/internal/config"
// )
//
// En tu función principal o `main`:
//
// awsService, err := config.NewAWSService(context.Background())
// if err != nil {
// 	log.Fatalf("No se pudo inicializar el servicio de AWS: %v", err)
// }
//
// awsConfig := awsService.GetConfig()
//
// Y luego, usa `awsConfig` para crear un cliente de SQS, por ejemplo:
//
// sqsClient := sqs.NewFromConfig(awsConfig)
