package email

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- MOCK PARA LA INTERFAZ EmailSender ---

type MockEmailSender struct {
	mock.Mock
}

// Send es la implementación del mock que registra la llamada.
func (m *MockEmailSender) Send(ctx context.Context, to, subject, htmlContent string) error {
	args := m.Mock.Called(ctx, to, subject, htmlContent)
	return args.Error(0)
}

// --- TEST UNITARIO PARA TemplateSender ---

func TestTemplateSender_SendFromTemplate_Success(t *testing.T) {
	// --- Arrange (Preparación) ---

	// 1. Crear un directorio y una plantilla de email temporales para el test.
	// t.TempDir() crea una carpeta que se borra automáticamente al terminar el test.
	tempDir := t.TempDir()
	templateName := "welcome.md"
	templatePath := filepath.Join(tempDir, templateName)
	templateContent := "Hola, {{ .UserName }}!" // Un template simple

	err := os.WriteFile(templatePath, []byte(templateContent), 0644)
	require.NoError(t, err, "No se pudo crear el archivo de plantilla temporal")

	// 2. Crear las dependencias.
	mockSender := new(MockEmailSender)
	// Creamos el servicio que vamos a probar, apuntando a nuestro directorio temporal.
	templateSvc, err := NewTemplateSender(mockSender, tempDir)
	require.NoError(t, err, "NewTemplateSender no debería fallar")

	// 3. Definir los datos de entrada para el método.
	to := "test@example.com"
	subject := "Bienvenido"
	data := map[string]any{
		"UserName": "Alex",
	}

	// 4. Definir el resultado esperado. Blackfriday envuelve el texto en párrafos <p>.
	expectedHTML := "<p>Hola, Alex!</p>\n"

	// 5. Configurar la expectativa del Mock.
	// Esperamos que el método 'Send' del servicio base sea llamado con el HTML ya renderizado.
	mockSender.Mock.On("Send", mock.Anything, to, subject, expectedHTML).Return(nil)

	// --- Act (Ejecución) ---
	err = templateSvc.SendFromTemplate(context.Background(), to, subject, templateName, data)

	// --- Assert (Verificación) ---
	// 1. Verificamos que nuestro servicio no devolvió error.
	assert.NoError(t, err)

	// 2. Verificamos que el mock fue llamado exactamente como esperábamos.
	mockSender.Mock.AssertExpectations(t)
}
