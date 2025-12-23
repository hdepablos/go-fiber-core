package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// Handler de la función Lambda. Recibe un CloudWatchEvent.
func handleRequest(ctx context.Context, event events.CloudWatchEvent) error {
	// Imprime el evento que se recibió.
	// Esto es útil para depurar y ver el contenido del evento programado.
	fmt.Printf("Evento de daily 24 cron recibido: %+v\n", event)

	// Puedes agregar tu lógica de negocio aquí.
	// Por ejemplo, procesar una cola SQS, llamar a otra API, etc.
	fmt.Println("¡Función ejecutada por el cron daily 24 cron !")

	// Si todo salió bien, devuelve nil.
	return nil
}

func main() {
	if os.Getenv("DEV_MODE") == "true" {
		log.Println("🏠 Modo desarrollo local - simulando evento cron daily 24 cron...")

		event := events.CloudWatchEvent{
			DetailType: "Scheduled Event",
			Source:     "aws.events",
		}

		err := handleRequest(context.Background(), event)
		if err != nil {
			log.Fatalf("Error en procesamiento local cron daily 24 cron: %v", err)
		}
		log.Println("Procesamiento local terminado daily 24 cron")
	} else {
		log.Println("🔥 Iniciando Lambda daily 24 cron...")
		lambda.Start(handleRequest)
	}
}
