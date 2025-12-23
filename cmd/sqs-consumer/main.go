package main

import (
	"context"
	"encoding/json"
	"fmt"
	"go-fiber-core/internal/services/queue"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// LambdaResponse representa la respuesta del Lambda
type LambdaResponse struct {
	StatusCode int               `json:"statusCode"`
	Body       string            `json:"body"`
	Headers    map[string]string `json:"headers"`
	Records    []ProcessedRecord `json:"records,omitempty"`
}

// ProcessedRecord representa un registro procesado
type ProcessedRecord struct {
	MessageID    string `json:"messageId"`
	Status       string `json:"status"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

// SNSMessage representa un mensaje de SNS
type SNSMessage struct {
	Type             string `json:"Type"`
	MessageId        string `json:"MessageId"`
	Token            string `json:"Token,omitempty"`
	TopicArn         string `json:"TopicArn"`
	Subject          string `json:"Subject,omitempty"`
	Message          string `json:"Message"`
	SubscribeURL     string `json:"SubscribeURL,omitempty"`
	UnsubscribeURL   string `json:"UnsubscribeURL,omitempty"`
	Timestamp        string `json:"Timestamp"`
	SignatureVersion string `json:"SignatureVersion"`
	Signature        string `json:"Signature"`
	SigningCertURL   string `json:"SigningCertURL"`
}

var (
	sqsService *queue.SQSService
	snsClient  *sns.Client
)

func init() {
	log.Println("1 => Inicializando clientes de AWS...")

	// 2. Usar nuestro servicio de configuración único.
	ctx := context.Background()
	awsService, err := queue.NewAWSService(ctx)
	if err != nil {
		log.Fatalf("3 => ❌ Error al inicializar el servicio de AWS: %v", err)
	}

	// 4. Obtener la configuración de AWS
	awsConfig := awsService.GetConfig()

	// 5. CREAR EL CLIENTE SQS primero con la configuración centralizada.
	sqsClient := sqs.NewFromConfig(awsConfig)

	// 6. OBTENER LA URL de la cola desde el entorno.
	queueURL := os.Getenv("SQS_QUEUE_URL")
	if queueURL == "" {
		log.Fatalf("❌ La variable de entorno SQS_QUEUE_URL no está configurada")
	}

	// 7. Usar el nuevo constructor de tu servicio, pasándole el cliente y la URL.
	// Esta línea reemplaza la que te daba error.
	sqsService = queue.NewSQSService(sqsClient, queueURL)

	// 8. Usar la misma configuración para crear el cliente de SNS.
	snsClient = sns.NewFromConfig(awsConfig)
	log.Println("✅ Función DLQ-CONSUMER inicializados correctamente.")
}

// handleSQSEvent maneja los eventos de SQS
func handleSQSEvent(ctx context.Context, event events.SQSEvent) (LambdaResponse, error) {
	log.Printf("🔄 Processing SQS event with %d records", len(event.Records))

	var processedRecords []ProcessedRecord
	successCount := 0
	errorCount := 0

	for _, record := range event.Records {
		processedRecord := ProcessedRecord{
			MessageID: record.MessageId,
			Status:    "processed",
		}

		if isSNSMessage(record.Body) {
			log.Printf("📢 Detected SNS message in SQS record: %s", record.MessageId)
			if err := processSNSMessage(ctx, record.Body); err != nil {
				processedRecord.Status = "failed"
				processedRecord.ErrorMessage = err.Error()
				errorCount++
				log.Printf("❌ Error processing SNS message %s: %v", record.MessageId, err)
				// Retornamos el error para que SQS reintente el mensaje
				return LambdaResponse{}, err
			} else {
				successCount++
				log.Printf("✅ Successfully processed SNS message %s", record.MessageId)
			}
		} else {
			if err := processMessage(ctx, record); err != nil {
				processedRecord.Status = "failed"
				processedRecord.ErrorMessage = err.Error()
				errorCount++
				log.Printf("❌ Error processing message %s: %v", record.MessageId, err)
				// Retornamos el error para que SQS reintente el mensaje
				return LambdaResponse{}, err
			} else {
				successCount++
				log.Printf("✅ Successfully processed message %s", record.MessageId)
			}
		}

		processedRecords = append(processedRecords, processedRecord)
	}

	// Preparar respuesta
	response := LambdaResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Records: processedRecords,
	}

	responseBody := map[string]interface{}{
		"message": fmt.Sprintf("Processed %d records", len(event.Records)),
		"success": successCount,
		"errors":  errorCount,
		"records": processedRecords,
	}

	bodyBytes, _ := json.Marshal(responseBody)
	response.Body = string(bodyBytes)

	log.Printf("📊 Processing complete: %d success, %d errors", successCount, errorCount)
	return response, nil
}

// isSNSMessage verifica si el mensaje es de SNS
func isSNSMessage(body string) bool {
	var snsMessage SNSMessage
	if err := json.Unmarshal([]byte(body), &snsMessage); err != nil {
		return false
	}
	return snsMessage.Type != "" && (snsMessage.Type == "SubscriptionConfirmation" ||
		snsMessage.Type == "Notification" || snsMessage.Type == "UnsubscribeConfirmation")
}

// processSNSMessage procesa mensajes de SNS
func processSNSMessage(ctx context.Context, body string) error {
	var snsMessage SNSMessage
	if err := json.Unmarshal([]byte(body), &snsMessage); err != nil {
		return fmt.Errorf("error unmarshaling SNS message: %w", err)
	}

	log.Printf("📢 SNS Message Type: %s", snsMessage.Type)
	log.Printf("📢 SNS Topic ARN: %s", snsMessage.TopicArn)

	switch snsMessage.Type {
	case "SubscriptionConfirmation":
		return handleSubscriptionConfirmation(ctx, snsMessage)
	case "Notification":
		return handleSNSNotification(ctx, snsMessage)
	case "UnsubscribeConfirmation":
		log.Printf("🔄 Unsubscribe confirmation received for topic: %s", snsMessage.TopicArn)
		return nil
	default:
		return fmt.Errorf("unknown SNS message type: %s", snsMessage.Type)
	}
}

// handleSubscriptionConfirmation maneja la confirmación de suscripción SNS
func handleSubscriptionConfirmation(ctx context.Context, snsMessage SNSMessage) error {
	log.Printf("🔐 Processing subscription confirmation for topic: %s", snsMessage.TopicArn)

	if snsMessage.Token == "" {
		return fmt.Errorf("no token provided in subscription confirmation")
	}

	// El endpoint de LocalStack se maneja automáticamente por nuestro servicio de conexión,
	// pero la lógica de confirmación por HTTP todavía es necesaria.
	if isLocalStack() {
		return confirmSubscriptionLocalStack(ctx, snsMessage)
	}

	// Para AWS real, usar el cliente de SNS
	input := &sns.ConfirmSubscriptionInput{
		TopicArn: aws.String(snsMessage.TopicArn),
		Token:    aws.String(snsMessage.Token),
	}

	result, err := snsClient.ConfirmSubscription(ctx, input)
	if err != nil {
		return fmt.Errorf("error confirming subscription: %w", err)
	}

	log.Printf("✅ Subscription confirmed successfully. Subscription ARN: %s", *result.SubscriptionArn)
	return nil
}

// confirmSubscriptionLocalStack confirma la suscripción en LocalStack
func confirmSubscriptionLocalStack(ctx context.Context, snsMessage SNSMessage) error {
	if snsMessage.SubscribeURL == "" {
		return fmt.Errorf("no SubscribeURL provided for LocalStack confirmation")
	}

	log.Printf("🏠 Confirming subscription via LocalStack URL: %s", snsMessage.SubscribeURL)

	resp, err := http.Get(snsMessage.SubscribeURL)
	if err != nil {
		return fmt.Errorf("error making GET request to SubscribeURL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("subscription confirmation failed with status: %d", resp.StatusCode)
	}

	log.Printf("✅ LocalStack subscription confirmed successfully")
	return nil
}

// handleSNSNotification maneja las notificaciones SNS (mensajes de la DLQ)
func handleSNSNotification(ctx context.Context, snsMessage SNSMessage) error {
	log.Printf("🚨 Processing SNS notification from DLQ alarm")
	log.Printf("📋 Subject: %s", snsMessage.Subject)
	log.Printf("📋 Message: %s", snsMessage.Message)

	// Aquí puedes agregar lógica adicional para manejar las alarmas de DLQ

	var alarmData map[string]interface{}
	if err := json.Unmarshal([]byte(snsMessage.Message), &alarmData); err == nil {
		if alarmName, ok := alarmData["AlarmName"].(string); ok {
			log.Printf("🚨 Alarm triggered: %s", alarmName)
		}
		if newState, ok := alarmData["NewStateValue"].(string); ok {
			log.Printf("🚨 New state: %s", newState)
		}
	}

	return nil
}

// processMessage procesa un mensaje individual de SQS
func processMessage(ctx context.Context, record events.SQSMessage) error {
	log.Printf("🔄 Processing message: %s", record.MessageId)
	log.Printf("📋 Message body: %s", record.Body)

	// Parsear el mensaje
	var message queue.Message // Asumimos que existe este tipo de mensaje
	if err := json.Unmarshal([]byte(record.Body), &message); err != nil {
		return fmt.Errorf("error unmarshaling message: %w", err)
	}

	// Ejemplo de procesamiento
	log.Printf("📨 Processing message ID: %s", message.ID)
	log.Printf("🏷️  Message source: %s", message.Source)
	log.Printf("📝 Message content: %s", message.Body)

	if message.ID == "999" {
		log.Printf("❌ Error processing message ID: %s", message.ID)
		return fmt.Errorf("intentional error to trigger DLQ")
	}

	if message.Data != nil {
		log.Printf("📊 Message data: %+v", message.Data)
	}

	return nil
}

// isLocalStack verifica si estamos ejecutando en LocalStack
func isLocalStack() bool {
	endpoint := os.Getenv("AWS_ENDPOINT_URL")
	return endpoint != ""
}

// main función principal del Lambda
func main() {
	if os.Getenv("DEV_MODE") == "true" {
		log.Println("🏠 Modo desarrollo local - simulando evento SQS")

		// Simular evento SQS con 1 mensaje
		event := events.SQSEvent{
			Records: []events.SQSMessage{
				{
					MessageId: "local-1",
					Body:      `{"ID":"123", "Source":"localtest", "Body":"Mensaje de prueba", "Data":null}`,
				},
			},
		}

		resp, err := handleSQSEvent(context.Background(), event)
		if err != nil {
			log.Fatalf("Error en procesamiento local: %v", err)
		}

		log.Printf("Respuesta local: %+v", resp)
	} else {
		log.Println("🔥 Iniciando Lambda SQS...")
		lambda.Start(handleSQSEvent)
	}
}
