package appconfig

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// AWSSecretProvider implementa SecretProvider obteniendo secretos desde AWS Secrets Manager.
// Ideal para entornos de producción seguros donde no se quieren exponer secretos como variables de entorno.
type AWSSecretProvider struct {
	client *secretsmanager.Client
}

// NewAWSSecretProvider inicializa el cliente de Secrets Manager.
// Carga la configuración de AWS del entorno (Roles IAM, Perfiles, etc).
func NewAWSSecretProvider(ctx context.Context) (*AWSSecretProvider, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("error cargando configuración de AWS: %w", err)
	}

	client := secretsmanager.NewFromConfig(cfg)
	return &AWSSecretProvider{
		client: client,
	}, nil
}

func (p *AWSSecretProvider) GetSecret(ctx context.Context, key string) (string, error) {
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(key),
	}

	result, err := p.client.GetSecretValue(ctx, input)
	if err != nil {
		return "", fmt.Errorf("error obteniendo secreto '%s' de AWS: %w", key, err)
	}

	if result.SecretString != nil {
		return *result.SecretString, nil
	}

	// En caso de secretos binarios (menos común para config de texto)
	return "", fmt.Errorf("el secreto '%s' no tiene contenido de texto (SecretString es nil)", key)
}
