package queue

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// AWSConfigProvider define el contrato mínimo para reutilizar configuración AWS.
type AWSConfigProvider interface {
	GetConfig() aws.Config
	NewS3Client() *s3.Client
}

// awsService es una estructura simple para manejar la configuración de AWS.
type awsService struct {
	cfg aws.Config
}

// NewAWSService crea y retorna una nueva instancia de AWSService.
// Se encarga de cargar la configuración de AWS.
func NewAWSService(ctx context.Context) (AWSConfigProvider, error) {
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
	// Preferimos AWS_ENDPOINT_URL, pero si no existe usamos LOCALSTACK_ENDPOINT_BASE para mantener
	// consistencia con los comandos del Makefile y el resto del entorno local.
	if endpoint := resolveAWSEndpoint(); endpoint != "" {
		fmt.Printf("⚠️ Usando Custom Endpoint: %s\n", endpoint)
		cfg.BaseEndpoint = aws.String(endpoint)
	}

	return &awsService{
		cfg: cfg,
	}, nil
}

// GetConfig retorna la configuración de AWS cargada.
// Es el método clave para la reutilización, ya que cualquier otro servicio
// puede llamarlo para obtener la configuración y crear clientes de servicios específicos.
func (s *awsService) GetConfig() aws.Config {
	return s.cfg
}

func (s *awsService) NewS3Client() *s3.Client {
	endpoint := resolveAWSEndpoint()
	return s3.NewFromConfig(s.cfg, func(o *s3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true
		}
	})
}

func resolveAWSEndpoint() string {
	if endpoint := os.Getenv("AWS_ENDPOINT_URL"); endpoint != "" {
		return endpoint
	}
	if endpoint := os.Getenv("LOCALSTACK_ENDPOINT_BASE"); endpoint != "" {
		return endpoint
	}
	return ""
}

func IsLocalAWSEndpoint() bool {
	endpoint := strings.ToLower(resolveAWSEndpoint())
	return strings.Contains(endpoint, "localhost") ||
		strings.Contains(endpoint, "127.0.0.1") ||
		strings.Contains(endpoint, "localstack")
}
