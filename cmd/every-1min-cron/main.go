package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// Handler de la función Lambda. Recibe un CloudWatchEvent.
func handleRequest(ctx context.Context, event events.CloudWatchEvent) error {
	// Imprime el evento que se recibió.
	// Esto es útil para depurar y ver el contenido del evento programado.
	fmt.Printf("Evento recibido de every 1min cron: %+v\n", event)

	// Puedes agregar tu lógica de negocio aquí.
	// Por ejemplo, procesar una cola SQS, llamar a otra API, etc.
	fmt.Println("¡Función ejecutada por el every 1min cron !")
	fmt.Println("¡Solo la función every 1min cron es actualizada !!!")
	fmt.Println("BUILD", time.Now().UTC().Format("2006-01-02T15:04:05"), "every 1min cron")

	// Si todo salió bien, devuelve nil.
	return nil
}

func main() {
	if os.Getenv("DEV_MODE") == "true" {
		log.Println("🏠 Modo desarrollo local - simulando evento every 1min cron")

		event := events.CloudWatchEvent{
			DetailType: "Scheduled Event",
			Source:     "aws.events",
		}

		err := handleRequest(context.Background(), event)
		if err != nil {
			log.Fatalf("Error en procesamiento local every 1min cron: %v", err)
		}
		log.Println("Procesamiento local every 1min cron terminado")
	} else {
		log.Println("🔥 Iniciando Lambda every 1min cron...")
		lambda.Start(handleRequest)
	}
}
