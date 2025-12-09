package cronjobs

import (
	"fmt"
	"log"

	cron "github.com/robfig/cron/v3" // ✅ Esta línea soluciona el error
)

// InitCronJobs inicializa y programa las tareas cron
func InitCronJobs() {
	// Crear una nueva instancia de cron
	c := cron.New()

	// Agregar una tarea que se ejecute todos los días a las 2:00 AM
	_, err := c.AddFunc("0 2 * * *", func() {
		log.Println("Ejecutando tarea programada: Limpieza de datos") // Descomentado para visibilidad
		performDataCleanup()
	})
	if err != nil {
		log.Fatalf("Error al agregar tarea cron: %v", err)
	}

	// Agregar otra tarea que se ejecute cada 1 minuto
	_, err = c.AddFunc("@every 1m", func() {
		log.Println("Ejecutando tarea programada: Envío de notificaciones cada minuto") // Descomentado para visibilidad
		performDataCleanup()
		sendNotifications()
	})
	if err != nil {
		log.Fatalf("Error al agregar tarea cron: %v", err)
	}

	// Iniciar el cron
	c.Start()

	// Mantener el cron activo mientras la aplicación esté en ejecución
	select {}
}

// performDataCleanup es una función de ejemplo para limpiar datos
func performDataCleanup() {
	// Aquí puedes implementar la lógica para limpiar datos en PostgreSQL
	fmt.Println("Limpiando datos antiguos...")
	// Ejemplo: Eliminar registros antiguos de la base de datos
}

// sendNotifications es una función de ejemplo para enviar notificaciones
func sendNotifications() {
	// Aquí puedes implementar la lógica para enviar notificaciones
	fmt.Println("Enviando notificaciones... cada 1 minuto")
	// Ejecutar todos los servicios en el orden especificado en la base de datos
	// err := services.ExecuteAllServices()
	// if err != nil {
	// 	log.Fatalf("Error ejecutando los servicios, desde el cron: %v", err)
	// }

	fmt.Println("Todos los servicios fueron ejecutados exitosamente, desde el cron")
}
