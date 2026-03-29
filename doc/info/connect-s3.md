# Configuración y Conexión con AWS S3 (`connect-s3`)

El proyecto utiliza el SDK oficial de AWS (`aws-sdk-go-v2`) para interactuar con S3, centralizando la configuración a través de la abstracción `queue.NewAWSService(ctx)`. Esto permite que los servicios que requieren S3 sean agnósticos al entorno (LocalStack, AWS, etc.).

## 1. Variables de Entorno

El comportamiento de S3 está determinado íntegramente por las siguientes variables de entorno.

### Obligatorias (Según el Entorno)

| Variable | Descripción | Valor Local (LocalStack) | Valor Prod (AWS) |
| :--- | :--- | :--- | :--- |
| `S3_BUCKET` | Nombre del bucket donde se guardarán/leerán los archivos. | `shared-local-dev` | `gofibercore-app-data-prod` |
| `AWS_ENDPOINT_URL` | URL personalizada para interceptar tráfico de AWS. **Solo usar en local**. | `http://localstack:4566` (o `http://127.0.0.1:4566` fuera de docker) | *(Vacio)* El SDK usará el endpoint real de AWS. |
| `AWS_REGION` | Región de AWS donde está el bucket. | `us-east-1` | `us-east-1` |

### Autenticación (Credenciales)

La autenticación se maneja automáticamente según el orden de prelación del SDK de AWS:

**Para LocalStack (Local):**
Debes forzar credenciales falsas en el `.env` para que el SDK no intente buscar perfiles de tu máquina.
```env
AWS_ACCESS_KEY_ID=test
AWS_SECRET_ACCESS_KEY=test
```

**Para AWS (Producción/Staging):**
- **EKS / Kubernetes:** **No definir** `AWS_ACCESS_KEY_ID` ni `AWS_SECRET_ACCESS_KEY`. El SDK usará los roles IRSA (IAM Roles for Service Accounts) inyectados en el Pod.
- **Lambda:** El SDK usará automáticamente el Rol de Ejecución asociado a la función Lambda.

---

## 2. Abstracción de Conexión en Código

Para conectarte a S3 desde cualquier servicio en Go, no necesitas instanciar credenciales manualmente. Simplemente usa el `AWSService` existente:

```go
import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go-fiber-core/internal/services/queue"
)

func MiServicio(ctx context.Context) error {
	// 1. Cargar configuración base (Aplica variables de entorno automáticamente)
	awsSvc, err := queue.NewAWSService(ctx)
	if err != nil {
		return err
	}

	// 2. Crear el cliente de S3
	s3Client := s3.NewFromConfig(awsSvc.GetConfig())

	// 3. Usar el cliente
	_, err = s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String("mi-bucket"),
	})
	
	return err
}
```

---

## 3. Creación Automática de Buckets (Solo Local)

En el entorno de desarrollo, LocalStack puede perder su estado al reiniciarse. Para evitar el error `NoSuchBucket`, los servicios que suben archivos (como `process_batch.go` o `merge.go`) incluyen una lógica de seguridad para **crear el bucket automáticamente si no existe, pero SOLO en entorno local**.

```go
// Ensure bucket exists in local/test environment
if os.Getenv("APP_ENV") == "local" || os.Getenv("APP_ENV") == "" {
	_, err := s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String("shared-local-dev"),
	})
	// Se ignora el error porque si ya existe, no importa.
}
```

> **⚠️ Importante:** Si `APP_ENV=production`, el servicio asume que el bucket fue creado previamente por Terraform/Infraestructura y fallará si no existe, lo cual es el comportamiento correcto de seguridad.

---

## 4. Comandos de Utilidad en el Makefile

Para interactuar con el bucket local desde la terminal sin necesidad de código, se han agregado comandos en el `Makefile`:

- **Ver contenido del bucket:**
  ```bash
  make s3-ls
  # Output: Lista recursiva de todos los archivos en el S3_BUCKET
  ```
- **Descargar un archivo (Se guarda en `tmp/s3_downloads/`):**
  ```bash
  make s3-download key=exports/multi_queue_batch_one_table/csv/final.csv
  ```
- **Eliminar un archivo específico (pide confirmación):**
  ```bash
  make s3-rm key=ruta/del/archivo.csv
  ```
- **Eliminar un directorio entero (pide confirmación):**
  ```bash
  make s3-rm-dir prefix=exports/multi_queue_batch_one_table/csv/run-uuid/
  ```