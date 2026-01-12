package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"go-fiber-core/cmd/api/di"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

var (
	container  *di.AppContainer
	appCleanup func()
	configPath string
)

// initializeApp carga el grafo de dependencias completo (Servicios, Repos, Config)
func initializeApp() {
	if container != nil {
		return
	}

	// Inicializamos el Contenedor que definimos en wire.go
	res, cleanup, err := di.InitializeAppContainer(configPath)
	if err != nil {
		log.Fatalf("💀 Error crítico inicializando AppContainer: %v", err)
	}

	container = res
	appCleanup = cleanup
	log.Println("🚀 AppContainer: Infraestructura y servicios inyectados correctamente")

	log.Println("######")
	log.Println("mostrar variables config cron every 1min")
	log.Printf("%+v\n", res.Config)
	log.Println("######")

}

func handleRequest(ctx context.Context, event events.CloudWatchEvent) error {
	initializeApp()

	fmt.Printf("go-fiber-core ===> Cron Every 1min ejecutándose en entorno: %s\n", container.Config.App.AppEnv)

	// --- EJEMPLO DE USO DE SERVICIOS ---
	// Aquí puedes usar cualquier servicio inyectado en el contenedor
	// userService := container.UserReaderService
	// data, err := userService.FindAll(ctx)

	var BuildMarker = "lambda0every01min0cron"
	_ = BuildMarker

	log.Println("🔥 Iniciando en modo AWS lambda0every01min0cron")
	return nil
}

func main() {
	// Definir ruta de configuración
	flagPath := flag.String("config", "internal/appconfig/config.yml", "Ruta al archivo YAML")
	flag.Parse()
	configPath = *flagPath

	// Ejecución según el entorno
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" || os.Getenv("APP_ENV") == "lambda" {
		log.Println("🔥 Iniciando modo AWS Lambda new v3 ...")
		lambda.Start(handleRequest)
	} else {
		log.Println("🏠 Modo local detectado - Simulando evento...")

		err := handleRequest(context.Background(), events.CloudWatchEvent{
			DetailType: "Scheduled Event",
			Source:     "aws.events",
		})

		if err != nil {
			log.Fatalf("❌ Error en ejecución local: %v", err)
		}

		// En local cerramos las conexiones manualmente al terminar
		if appCleanup != nil {
			appCleanup()
		}
		log.Println("✅ Procesamiento local terminado.")
	}
}
