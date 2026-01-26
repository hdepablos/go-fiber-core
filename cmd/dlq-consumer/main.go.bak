package main

import (
	"context"
	"fmt"
	"go-fiber-core/internal/services/queue"

	"log"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	// "github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

// clientSQS es una variable global para el cliente de SQS.
// Se inicializa una sola vez para ser reutilizado en múltiples invocaciones de la Lambda.
var clientSQS *sqs.Client

// init se ejecuta una vez, cuando la Lambda se inicia por primera vez.
func init() {
	// 1. Inicializar el servicio de conexión a AWS
	ctx := context.Background()
	awsService, err := queue.NewAWSService(ctx)
	if err != nil {
		// En un entorno de Lambda, si init falla, la función no se desplegará o no se ejecutará correctamente.
		log.Fatalf("Error fatal al inicializar el servicio de AWS: %v", err)
	}

	// 2. Crear un cliente de SQS usando la configuración del servicio AWS.
	clientSQS = sqs.NewFromConfig(awsService.GetConfig())
	log.Println("🚀 Cliente de DLQ inicializado exitosamente.")
}

// Handler es la función principal que AWS Lambda invocará.
// Recibe un evento de SQS con uno o más mensajes.
func Handler(ctx context.Context, sqsEvent events.SQSEvent) error {
	log.Printf("🌍 Recibidos %d mensajes del evento DLQ", len(sqsEvent.Records))

	// URL de la cola, extraída del primer mensaje.
	// Asumimos que todos los mensajes vienen de la misma cola.
	if len(sqsEvent.Records) == 0 {
		return nil
	}
	queueURL := os.Getenv("SQS_DLQ_URL")
	if queueURL == "" {
		log.Println("Variable de entorno SQS_DLQ_URL no está configurada. Usando el origen del mensaje.")
		queueURL = getQueueURLFromARN(sqsEvent.Records[0].EventSourceARN)
	}

	// Iterar sobre cada mensaje recibido en el evento
	for _, message := range sqsEvent.Records {
		// TODO: Lógica de negocio para reprocesar el mensaje
		log.Printf("Procesando mensaje con ID: %s", message.MessageId)
		log.Printf("Cuerpo del mensaje en el DLQ: %s", message.Body)

		// Simulación del procesamiento. Si algo falla aquí, la función `processMessage` debería retornar un error.
		// En este caso, el mensaje no se eliminaría y Lambda lo reintentaría.
		err := processMessage(ctx, message)
		if err != nil {
			log.Printf("Error al procesar mensaje %s: %v. No se eliminará de la cola.", message.MessageId, err)
			// Retornar el error para que AWS Lambda sepa que el procesamiento falló.
			// Esto hará que el mensaje vuelva a estar visible en la cola según su visibilidad timeout.
			return fmt.Errorf("error al procesar mensaje %s: %w", message.MessageId, err)
		}
	}

	// Si llegamos a este punto, todos los mensajes fueron procesados exitosamente.
	// La función Lambda no necesita eliminar los mensajes. SQS, al ver que la invocación
	// no retornó un error, elimina los mensajes del batch automáticamente.
	log.Println("Todos los mensajes procesados exitosamente.")
	log.Println("===============================================")
	return nil
}

// processMessage contiene la lógica para manejar un solo mensaje.
func processMessage(ctx context.Context, message events.SQSMessage) error {
	// Lógica de negocio aquí. Por ejemplo, reenviar el mensaje a la cola original,
	// guardarlo en una base de datos de errores, etc.

	fmt.Println("######")
	fmt.Printf("%+v\n", message)
	fmt.Println("######")

	// Ejemplo: Si el cuerpo del mensaje contiene "fail", simulamos un error.
	if message.Body == "fail" {
		return fmt.Errorf("error de negocio simulado para mensaje %s", message.MessageId)
	}

	// Si el procesamiento es exitoso, retornamos nil.
	log.Printf("Mensaje %s procesado con éxito.", message.MessageId)
	return nil
}

// getQueueURLFromARN extrae la URL de la cola de su ARN.
func getQueueURLFromARN(arn string) string {
	// Implementación simple para extraer el nombre de la cola del ARN.
	// Ejemplo de ARN: arn:aws:sqs:us-east-1:123456789012:nombre-de-la-cola
	parts := strings.Split(arn, ":")
	queueName := parts[len(parts)-1]
	region := parts[3]
	accountID := parts[4]

	// Formato de la URL de SQS
	return fmt.Sprintf("https://sqs.%s.amazonaws.com/%s/%s", region, accountID, queueName)
}

func main() {
	if os.Getenv("DEV_MODE") == "true" {
		log.Println("🏠 Modo desarrollo local - simulando evento DLQ")

		event := events.SQSEvent{
			Records: []events.SQSMessage{
				{
					MessageId:      "dlq-local-1",
					Body:           "Mensaje de error simulado",
					EventSourceARN: "arn:aws:sqs:us-east-1:000000000000:dlq-local",
				},
			},
		}

		err := Handler(context.Background(), event)
		if err != nil {
			log.Fatalf("Error en procesamiento local DLQ: %v", err)
		}

		log.Println("Procesamiento local DLQ terminado v3")
	} else {
		log.Println("🔥 Iniciando Lambda DLQ...")
		lambda.Start(Handler)
	}
}
