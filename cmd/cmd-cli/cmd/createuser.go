package cmd

import (
	"fmt"
	"go-fiber-core/cmd/api/di"
	"go-fiber-core/internal/models"

	"github.com/spf13/cobra"
)

var createUserCmd = &cobra.Command{
	Use:   "createUser",
	Short: "Crea un usuario de prueba en la base de datos.",
	Long: `Inicializa la configuración y dependencias del proyecto mediante Wire
			para crear un usuario por defecto con los siguientes datos:
			- Nombre:   test
			- Email:    test@test.com
			- Password: 123456`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("▶️ Iniciando creación de usuario de prueba...")

		// --- 1. Inicializar dependencias con Wire ---
		server, cleanup, err := di.InitializeServer("internal/appconfig/config.yml")
		if err != nil {
			return fmt.Errorf("❌ Error inicializando dependencias: %w", err)
		}
		defer cleanup()
		fmt.Println("⚙️ Dependencias inicializadas correctamente.")

		// --- 2. Acceder al servicio de usuarios ---
		userService := server.UserWriterService
		if userService == nil {
			return fmt.Errorf("❌ No se pudo obtener UserWriterService desde el servidor")
		}
		fmt.Println("🔧 Servicio de usuario obtenido desde el contenedor DI.")

		// --- 3. Crear el usuario ---
		user := &models.User{
			Name:     "test",
			Email:    "test@test.com",
			Password: "123456",
		}

		fmt.Printf("👤 Creando usuario con email: %s...\n", user.Email)
		if err := userService.Create(cmd.Context(), user); err != nil {
			return fmt.Errorf("❌ No se pudo crear el usuario: %w", err)
		}

		fmt.Println("🎉 ¡Usuario 'test' creado exitosamente!")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(createUserCmd)
}
