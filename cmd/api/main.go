// Package main implements the entry point for the API server.
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	// CAMBIO: Se importa el paquete 'di' que contiene el inyector de Wire.
	"go-fiber-core/cmd/api/di"
	"go-fiber-core/internal/server"

	_ "github.com/joho/godotenv/autoload"
)

// CAMBIO: La función gracefulShutdown ahora es mucho más simple.
// Su única responsabilidad es apagar el servidor HTTP.
// La función `cleanup` de Wire se encarga de cerrar las demás conexiones.
func gracefulShutdown(fiberServer *server.FiberServer, done chan bool) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()

	log.Println("⬇️ Shutting down gracefully, press Ctrl+C again to force")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// CAMBIO: Se llama al método a través del campo .App
	if err := fiberServer.App.ShutdownWithContext(shutdownCtx); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			log.Printf("❌ Server forced to shutdown with error: %v", err)
		}
	}
	log.Println("✅ HTTP server stopped.")

	done <- true
}

func main() {
	// Carga de configuración desde el flag
	configPath := flag.String("config", "internal/appconfig/config.yml", "Ruta al archivo de configuración YAML")
	flag.Parse()

	// --- INICIALIZACIÓN CON WIRE ---
	// ¡Toda la creación de dependencias se reduce a esta línea!
	server, cleanup, err := di.InitializeServer(*configPath)
	if err != nil {
		log.Fatalf("💀 Failed to initialize server: %v", err)
	}
	log.Println("🚀 Dependencies initialized successfully!")

	// La función 'cleanup' que retorna Wire se usará para el cierre ordenado.
	// Se ejecutará cuando la función main termine.
	defer cleanup()

	

	// --- ARRANQUE Y CIERRE ORDENADO ---
	done := make(chan bool, 1)

	go func() {
		log.Printf("🚀 Starting server on port :9009")
		if err := server.App.Listen(":9009"); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("❌ HTTP server error: %s", err)
		}
	}()
	// CAMBIO: La llamada a gracefulShutdown ahora es más simple.
	go gracefulShutdown(server, done)

	<-done
	log.Println("👋 Graceful shutdown complete. Exiting.")
}
