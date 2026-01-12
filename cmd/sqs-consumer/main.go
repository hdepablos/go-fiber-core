package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"

	"go-fiber-core/cmd/api/di"
	"go-fiber-core/internal/services/queue"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	_ "github.com/joho/godotenv/autoload"
)

var (
	appContainer *di.AppContainer
	appCleanup   func()
	configPath   string
)

// SNSMessage representa la estructura de notificaciones de AWS
type SNSMessage struct {
	Type         string `json:"Type"`
	MessageId    string `json:"MessageId"`
	TopicArn     string `json:"TopicArn"`
	Subject      string `json:"Subject"`
	Message      string `json:"Message"`
	SubscribeURL string `json:"SubscribeURL"`
	Token        string `json:"Token,omitempty"`
}

func initializeApp() {
	if appContainer != nil {
		return
	}
	res, cleanup, err := di.InitializeAppContainer(configPath)
	if err != nil {
		log.Fatalf("💀 Error en DI (sqs-consumer): %v", err)
	}
	appContainer = res
	appCleanup = cleanup
	log.Println("🚀 SQS Consumer: Infraestructura inyectada correctamente")
}

// Handler procesa el evento de SQS
func Handler(ctx context.Context, event events.SQSEvent) (interface{}, error) {
	initializeApp()

	log.Printf("🔄 Recibidos %d registros para procesar", len(event.Records))

	for _, record := range event.Records {
		// Determinar si es un mensaje directo de SQS o viene envuelto en SNS
		if isSNSMessage(record.Body) {
			if err := handleSNS(ctx, record.Body); err != nil {
				return nil, err // Provoca reintento en SQS
			}
		} else {
			if err := handleStandardSQS(ctx, record); err != nil {
				return nil, err // Provoca reintento en SQS -> Eventualmente DLQ
			}
		}
	}

	return map[string]string{"status": "ok"}, nil
}

func isSNSMessage(body string) bool {
	var m struct{ Type string }
	_ = json.Unmarshal([]byte(body), &m)
	return m.Type == "Notification" || m.Type == "SubscriptionConfirmation"
}

func handleSNS(ctx context.Context, body string) error {
	var snsMsg SNSMessage
	if err := json.Unmarshal([]byte(body), &snsMsg); err != nil {
		return err
	}

	if snsMsg.Type == "SubscriptionConfirmation" && snsMsg.SubscribeURL != "" {
		log.Printf("🔐 Confirmando suscripción SNS: %s", snsMsg.TopicArn)
		resp, err := http.Get(snsMsg.SubscribeURL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		return nil
	}

	log.Printf("📢 SNS Notification recibida: %s", snsMsg.Subject)
	return nil
}

func handleStandardSQS(ctx context.Context, record events.SQSMessage) error {
	log.Printf("📦 Procesando mensaje: %s", record.MessageId)

	// --- LÓGICA DE NEGOCIO Y PRUEBA DE DLQ ---
	var msg queue.Message
	if err := json.Unmarshal([]byte(record.Body), &msg); err != nil {
		log.Printf("⚠️ Error unmarshaling: %v", err)
		return nil // No reintentamos si el formato es inválido (venenoso)
	}

	// Lógica de fallo solicitada: Si el ID es 999, lanzamos error para que SQS reintente
	// y tras N intentos (Redrive Policy) pase a la DLQ.
	if msg.ID == "999" {
		log.Printf("❌ ERROR SIMULADO: ID 999 detectado. Forzando reintento para activar DLQ.")
		return fmt.Errorf("fallo intencional: mensaje con ID 999 enviado a reintento")
	}

	// Aquí usarías tus servicios inyectados
	// appContainer.Connect.ConnectGormWrite.Create(&SomeModel{...})

	var BuildMarker = "lambda0sqs0consumer"
	_ = BuildMarker

	log.Println("🔥 Iniciando en modo AWS lambda0sqs0consumer")
	log.Printf("✅ Mensaje %s procesado exitosamente", msg.ID)

	return nil
}

func main() {
	fPath := flag.String("config", "internal/appconfig/config.yml", "Ruta al config")
	flag.Parse()
	configPath = *fPath

	// if os.Getenv("APP_ENV") == "lambda" {
	lambda.Start(Handler)
	// } else {
	// 	runLocal()
	// }
}

func runLocal() {
	log.Println("🏠 Ejecución Local (Simulación)")

	// Caso que debe FALLAR para ir a DLQ
	mockEvent := events.SQSEvent{
		Records: []events.SQSMessage{
			{
				MessageId: "sqs-test-id-999",
				Body:      `{"ID":"999", "Source":"CLI", "Body":"Este mensaje debe ir a la DLQ"}`,
			},
		},
	}

	_, err := Handler(context.Background(), mockEvent)
	if err != nil {
		log.Printf("🔴 Resultado esperado del test local: %v", err)
	}

	if appCleanup != nil {
		appCleanup()
	}
}
