// cmd/sendemail.go (o el nombre que tenga el archivo)
package cmd

import (
	"fmt"
	"go-fiber-core/internal/dtos/config"
	"go-fiber-core/internal/services/email"
	"go-fiber-core/internal/utils"
	"log"

	"github.com/spf13/cobra"
)

// sendemailCmd representa el comando para enviar un email de prueba.
var sendemailCmd = &cobra.Command{
	Use:   "sendemail",
	Short: "Envía un email de prueba usando el servicio de plantillas.",
	Long: `Este comando inicializa la configuración y los servicios de la aplicación
para enviar un email de prueba basado en la plantilla 'contact.md'.

Es útil para verificar que la configuración de SMTP y las plantillas de correo
están funcionando correctamente.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("▶️  Iniciando envío de email de prueba...")

		// --- 1. Cargar Configuración ---
		configPath := "internal/appconfig/config.yml"

		// CAMBIO: Se capturan los dos valores (config y error) de NewAppConfig.
		appConfig, err := config.NewAppConfig(configPath)
		if err != nil {
			return fmt.Errorf("❌ error cargando la configuración: %w", err)
		}
		fmt.Println("⚙️  Configuración cargada.")

		// --- 2. Inyección de Dependencias (DI) ---
		var emailSvc email.EmailSender
		if utils.IsProduction(*appConfig) {
			emailSvc = email.NewGomailService(appConfig.EmailConfig)
		} else {
			emailSvc = email.NewLogSender(appConfig.EmailConfig)
		}

		templateSvc, err := email.NewTemplateSender(emailSvc, "internal/templates")
		if err != nil {
			return fmt.Errorf("❌ no se pudieron cargar las plantillas de email: %w", err)
		}
		fmt.Println("🔧 Servicios de email inicializados.")

		// --- 3. Preparar y Enviar el Email ---
		data := map[string]any{
			"Name": "Héctor Depablos T.",
			"Edad": 45,
		}

		to := "destinatario@ejemplo.com"
		subject := "Correo de prueba desde Comando"
		templateName := "contact.md"

		fmt.Printf("📬 Enviando email a '%s' usando la plantilla '%s'...\n", to, templateName)

		err = templateSvc.SendFromTemplate(cmd.Context(), to, subject, templateName, data)
		if err != nil {
			return fmt.Errorf("❌ no se pudo enviar el email: %w", err)
		}

		log.Println("✅ ¡Email de prueba enviado/registrado exitosamente!")
		fmt.Println("Fin del comando.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(sendemailCmd)
}
