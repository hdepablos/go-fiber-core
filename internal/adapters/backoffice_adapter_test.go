package adapters

import (
	"context"
	"encoding/json"
	"go-fiber-core/internal/dtos"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	resty "github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestServer crea un servidor de prueba y un adaptador que apunta a él.
// Esto nos permite simular respuestas de la API sin hacer llamadas reales.
func setupTestServer(handler http.HandlerFunc) (*httptest.Server, *BackofficeAdapter) {
	server := httptest.NewServer(handler)

	// Creamos un cliente Resty que apunta a la URL de nuestro servidor de prueba.
	httpClient := resty.New().SetBaseURL(server.URL)

	adapter := NewBackofficeAdapter(httpClient)

	return server, adapter
}

// TestPostReversal_Success prueba el "camino feliz": una petición exitosa (200 OK).
func TestPostReversal_Success(t *testing.T) {
	// 1. Arrange: Preparamos el entorno de la prueba.
	expectedResponse := map[string]string{"status": "ok", "message": "reversal processed"}
	jsonResponse, _ := json.Marshal(expectedResponse)

	// Creamos un manejador que simula una respuesta exitosa de la API.
	handler := func(w http.ResponseWriter, r *http.Request) {
		// Verificamos que el método sea POST.
		assert.Equal(t, http.MethodPost, r.Method, "Se esperaba un método POST")

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jsonResponse)
	}

	server, adapter := setupTestServer(handler)
	defer server.Close() // Nos aseguramos de cerrar el servidor al final del test.

	// 2. Act: Ejecutamos la función que queremos probar.
	requestBody := dtos.Config{ /* ... datos de prueba ... */ }
	resp, err := adapter.PostReversal(context.Background(), requestBody)

	// 3. Assert: Verificamos que los resultados sean los esperados.
	require.NoError(t, err, "No se esperaba un error en la llamada al adaptador")
	require.NotNil(t, resp, "La respuesta no debería ser nula")

	assert.Equal(t, http.StatusOK, resp.StatusCode(), "El código de estado debería ser 200 OK")
	assert.JSONEq(t, string(jsonResponse), string(resp.Body()), "El cuerpo de la respuesta no coincide con lo esperado")
}

// TestPostReversal_ServerError prueba cómo reacciona el adaptador a un error del servidor (500).
func TestPostReversal_ServerError(t *testing.T) {
	// 1. Arrange
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "internal server error"}`))
	}
	server, adapter := setupTestServer(handler)
	defer server.Close()

	// 2. Act
	resp, err := adapter.PostReversal(context.Background(), dtos.Config{})

	// 3. Assert
	require.NoError(t, err, "Un error 500 no debería producir un error en la conexión, solo un status code no exitoso")
	require.NotNil(t, resp, "La respuesta no debería ser nula")
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode(), "El código de estado debería ser 500")
	assert.Contains(t, string(resp.Body()), "internal server error", "El cuerpo debería contener el mensaje de error")
}

// TestPostReversal_ContextCancellation prueba que la petición se cancela si el contexto finaliza.
func TestPostReversal_ContextCancellation(t *testing.T) {
	// 1. Arrange
	handler := func(w http.ResponseWriter, r *http.Request) {
		// Simulamos una operación lenta que tarda más que el contexto.
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}
	server, adapter := setupTestServer(handler)
	defer server.Close()

	// Creamos un contexto que se cancela inmediatamente.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// 2. Act
	resp, err := adapter.PostReversal(ctx, dtos.Config{})

	// 3. Assert
	require.Error(t, err, "Se esperaba un error debido a la cancelación del contexto")
	assert.Nil(t, resp, "La respuesta debería ser nula si hay un error de contexto")
	// Verificamos que el error sea específicamente por el contexto cancelado.
	assert.ErrorIs(t, err, context.Canceled, "El error debería ser de tipo context.Canceled")
}
