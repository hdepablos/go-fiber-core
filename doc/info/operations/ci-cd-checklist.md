# Checklist CI/CD para Equipo de Plataforma

## Objetivo
- Implementar un pipeline CI/CD robusto, seguro y reproducible para Go + Terraform + AWS (Lambda/APIs/consumers/crons) con entornos staging y producción.

## Identidad y Accesos AWS
- Configurar OIDC del proveedor CI (GitHub/GitLab) para AssumeRole sin llaves estáticas.
- Crear roles IAM por entorno (staging/prod) para CI:
  - Staging: permisos de Terraform Apply acotados a recursos del proyecto.
  - Producción: Terraform Plan por defecto; Apply controlado por gate manual.
- Crear roles de ejecución por servicio:
  - Lambda: adjuntar AWSLambdaBasicExecutionRole.
  - No-Lambda (ECS/containers/workers): logs:CreateLogStream, logs:PutLogEvents sobre log groups del proyecto.
- Segregar permisos por entorno/cuenta o por prefijos/ARNs.

## Terraform (State, Workspaces y Seguridad)
- Backend remoto por entorno:
  - S3 con versioning + cifrado SSE-KMS.
  - DynamoDB para locking de state.
- Workspaces: local, staging, prod.
- Validaciones en CI:
  - terraform fmt, validate, tflint, tfsec.
  - OPA/Conftest (retención mínima de logs, tags requeridas, cifrado).
- Flujo:
  - Plan como artefacto y comentario en PR.
  - Auto-apply en staging (tras aprobación PR).
  - Apply en producción con gate manual.

## Artefactos y Build
- Go:
  - go mod tidy + caché de módulos.
  - golangci-lint.
  - go test ./... con -race y cobertura.
- Lambdas:
  - GOOS=linux, GOARCH=arm64 cuando aplique.
  - ZIP determinístico por función, hash reproducible, artefacto del pipeline.
- Contenedores (si aplica):
  - Multi-arch (arm64/amd64).
  - Docker layer cache.
  - Escaneo de vulnerabilidades (Trivy/Grype).

## Despliegue
- Lambda:
  - Aliases por entorno (staging/prod).
  - Canary/Linear con CodeDeploy (p.ej. 10%/5m → 100%).
  - Variables por entorno; secretos en SSM/Secrets Manager.
- Terraform:
  - Importar recursos existentes críticos antes del primer apply (si corresponde).
  - Roles de entorno vía OIDC por job (staging/prod).

## Observabilidad y Logs
- CloudWatch Logs:
  - Lambda: /aws/lambda/<function_name>.
  - No-Lambda: /app/${project_name}/${service_name}.
  - Retención configurable (p.ej. 7–14 días).
- Aplicación (Zap):
  - LOG_OUTPUT=file|stdout|both (stdout en cloud para captura por CloudWatch).
  - DB_LOG_LEVEL=silent|error|warn|info y DB_SLOW_THRESHOLD_MS.
- Alarmas y rollback:
  - Métricas clave: errores 5xx, throttles, duración, DLQ.
  - Rollback automático con CodeDeploy ante alarmas.
- Trazas:
  - Evaluar X-Ray/OpenTelemetry si se requiere trazabilidad distribuida.

## Seguridad
- SAST (gosec), análisis de dependencias y contenedores.
- Secret scanning (gitleaks).
- SBOM (syft) publicado como artefacto.
- Firma/atestaciones (SLSA provenance) si el stack lo permite.
- Branch protection y revisiones obligatorias; permisos de apply prod restringidos.

## Red y Cifrado
- Buckets S3 (artefactos y state) cifrados (SSE-KMS) y con políticas de acceso mínimo.
- Logs cifrados (CloudWatch por defecto; KMS opcional).
- IAM con principio de menor privilegio y scoping por ARN.

## Variables y Secretos del CI
- Variables por entorno:
  - AWS_ACCOUNT_ID, AWS_REGION.
  - ROLE_TO_ASSUME_STAGING, ROLE_TO_ASSUME_PROD.
  - TF_BACKEND_BUCKET, TF_BACKEND_DDB_TABLE.
  - Flags de pipeline (RUN_INTEGRATION_TESTS, APPLY_STAGING_AUTOMATIC, REQUIRE_MANUAL_APPROVAL_PROD).
- Secretos gestionados en el proveedor CI (no en repos).

## Pruebas y Gates
- Pre-merge:
  - Lint, tests unitarios (-race), cobertura mínima.
  - Tests de integración acotados (LocalStack) con timeout.
  - Terraform plan + tflint + tfsec + OPA.
- Staging:
  - Build + Terraform apply.
  - Smoke tests post-deploy (salud de API, colas SQS).
- Producción:
  - Gate manual, plan visible.
  - Canary/linear deployment con alarmas vinculadas.

## Tagging y Trazabilidad
- Tags obligatorias: Project, Environment, ManagedBy, Owner.
- Conventional Commits, CHANGELOG automatizado y releases versionadas (SemVer).

## Costos y Gobierno
- Budgets por entorno con alertas.
- Cost Anomaly Detection (opcional).
- Retención de logs acorde a SLA/SLO.

## Entregables/Acciones para Plataforma
- Roles IAM por entorno (CI y runtime) con OIDC y privilegio mínimo.
- Backend de Terraform (S3 + DynamoDB) por entorno y workspaces definidos.
- Variables/secretos del CI configurados por entorno.
- CodeDeploy (aplicación y deployment groups) para Lambdas.
- Alarmas CloudWatch esenciales y vínculos a CodeDeploy para rollback.
- Documentación operativa:
  - Runbooks de promoción y rollback.
  - Procedimientos de importación a Terraform.
  - Flujo de versiones y políticas de aprobación.
