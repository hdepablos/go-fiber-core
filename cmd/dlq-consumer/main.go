package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"go-fiber-core/cmd/api/di"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	_ "github.com/joho/godotenv/autoload"
)

var (
	appContainer *di.AppContainer
	appCleanup   func()
	configPath   string
)

// initializeApp se encarga de la Inyección de Dependencias (DI) usando Wire.
func initializeApp() {
	log.Println("****************************************************")
	log.Println("****************************************************")
	var BuildMarker = "lambda0dlq0consumer"
	_ = BuildMarker

	log.Println("🔥 Iniciando en modo AWS lambda0dlq0consumer")
	log.Println("🚀 Inicializando aplicación DLQ Consumer...")
	log.Println("****************************************************")
	log.Println("****************************************************")

	if appContainer != nil {
		return
	}

	res, cleanup, err := di.InitializeAppContainer(configPath)
	if err != nil {
		log.Fatalf("💀 Error fatal al inicializar dependencias (Wire): %v", err)
	}

	appContainer = res
	appCleanup = cleanup

	log.Println("🚀 DLQ Consumer: Dependencias e infraestructura cargadas correctamente.")
}

// Handler es el punto de entrada que AWS Lambda invoca ante un evento de SQS.
func Handler(ctx context.Context, sqsEvent events.SQSEvent) error {
	initializeApp()

	log.Printf("🌍 Evento recibido: %d mensajes a procesar.", len(sqsEvent.Records))

	for _, message := range sqsEvent.Records {
		log.Printf("📥 Procesando mensaje ID: %s", message.MessageId)

		if err := processMessage(ctx, message); err != nil {
			log.Printf("❌ Error al procesar mensaje %s: %v", message.MessageId, err)
			return err
		}
	}

	log.Println("✅ Batch procesado exitosamente.")
	return nil
}

func processMessage(ctx context.Context, message events.SQSMessage) error {
	// Ahora puedes usar appContainer.Connect.ConnectGormWrite para persistir datos
	fmt.Printf("--- Cuerpo del Mensaje ---\n%s\n------------------------\n", message.Body)

	if message.Body == "fail" {
		return fmt.Errorf("error de negocio simulado para mensaje: %s", message.MessageId)
	}

	log.Printf("✔️ Mensaje %s procesado con éxito. Este es del DLQ !!!", message.MessageId)
	return nil
}

func main() {
	// 1. Configuración de flags
	// Es importante que configPath se asigne antes de llamar a initializeApp()
	fPath := flag.String("config", "internal/appconfig/config.yml", "Ruta al archivo de configuración YAML")
	flag.Parse()
	configPath = *fPath

	// if os.Getenv("APP_ENV") == "lambda" {
	log.Println("🔥 Iniciando Lambda DLQ Consumer...")
	lambda.Start(Handler)
	// } else {
	// 	runLocal()
	// }
}

func runLocal() {
	log.Println("🏠 Modo desarrollo local activado.")

	event := events.SQSEvent{
		Records: []events.SQSMessage{
			{
				MessageId: "local-dev-001",
				Body:      "Mensaje de prueba local",
			},
		},
	}

	err := Handler(context.Background(), event)
	if err != nil {
		log.Printf("❌ Error en ejecución local: %v", err)
	}

	if appCleanup != nil {
		log.Println("♻️ Ejecutando limpieza de conexiones...")
		appCleanup()
	}
	log.Println("👋 Ejecución local terminada.")
}
