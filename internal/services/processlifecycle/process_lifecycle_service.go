package processlifecycle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"go-fiber-core/internal/contextkeys"
	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/dtos"
	"go-fiber-core/internal/dtos/connect"
	"go-fiber-core/internal/dtos/requests"
	"go-fiber-core/internal/logger"
	"go-fiber-core/internal/models"
	"go-fiber-core/internal/services/cache"
	"go-fiber-core/internal/services/processlifecyclemanager"
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"
)

type Service interface {
	ReplicateProcessVersion(ctx context.Context, processVersionID int64, operatorID int64) (int64, error)
	PromoteProcessVersion(ctx context.Context, processVersionID int64, operatorID int64, comment string) error
	ResolveProcessVersion(ctx context.Context, processTypeID int64, sedeID int64, overrideProcessVersionID *int64, roadmap int, useCache bool) (int64, []Step, error)
	MoveProcessVersionToTest(ctx context.Context, processVersionID int64) error
	ListProcessVersions(ctx context.Context, req dtos.PaginationRequest) (*dtos.PaginationResponse[models.ProcessVersionListItem], error)
	GetProcessVersionByID(ctx context.Context, processVersionID int64) (*models.ProcessVersionListItem, error)
	GetProcessStepsByVersionID(ctx context.Context, processVersionID int64) ([]Step, error)
	Run(ctx context.Context, req requests.RunProcessRequest) (int64, *contracts.ServiceContext, error)
}

type Step struct {
	Name         string          `json:"name"`
	ExecutionKey string          `json:"execution_key"`
	Config       json.RawMessage `json:"config"`
	StepOrder    int32           `json:"step_order"`
}

type service struct {
	conn          *connect.ConnectDTO
	replicateSvc  processlifecyclemanager.ReplicateService
	promoteSvc    processlifecyclemanager.PromoteService
	resolveSvc    processlifecyclemanager.ResolveService
	moveToTestSvc processlifecyclemanager.MoveToTestService
}

func NewService(conn *connect.ConnectDTO) Service {
	resolveSvc := processlifecyclemanager.NewResolveService(conn)
	return &service{
		conn:          conn,
		replicateSvc:  processlifecyclemanager.NewReplicateService(conn),
		promoteSvc:    processlifecyclemanager.NewPromoteService(conn),
		resolveSvc:    resolveSvc,
		moveToTestSvc: processlifecyclemanager.NewMoveToTestService(conn),
	}
}

func (s *service) ReplicateProcessVersion(ctx context.Context, processVersionID int64, operatorID int64) (int64, error) {
	return s.replicateSvc.ReplicateProcessVersion(ctx, processVersionID, operatorID)
}

// PromoteProcessVersion promueve una versión de proceso específica a PRODUCCIÓN.
// Este método orquesta la validación, bloqueo distribuido (Redis) y actualización atómica en base de datos.
func (s *service) PromoteProcessVersion(ctx context.Context, processVersionID int64, operatorID int64, comment string) error {
	// Inicializar logger específico para este servicio
	log := logger.GetLogger("process_lifecycle_service")

	// 1. Log de entrada: Registrar intento de promoción con parámetros clave
	log.Info("Starting PromoteProcessVersion execution",
		zap.Int64("process_version_id", processVersionID),
		zap.Int64("operator_id", operatorID),
		zap.String("comment", comment),
	)

	// Validación básica de argumentos
	if processVersionID <= 0 || operatorID <= 0 {
		log.Warn("Invalid arguments provided",
			zap.Int64("process_version_id", processVersionID),
			zap.Int64("operator_id", operatorID),
		)
		return domain.ErrInvalidArgument
	}

	// 2. Obtener conexión a base de datos (Escritura)
	dbRead := s.conn.ConnectGormRead
	if dbRead == nil {
		err := fmt.Errorf("gorm read connection is not initialized")
		log.Error("Database connection failed", zap.Error(err))
		return err
	}

	// 3. Configurar cliente Redis para bloqueo distribuido
	redisClient := s.conn.ConnectRedis
	projectPrefix := os.Getenv("APP_NAME")
	if projectPrefix == "" {
		projectPrefix = "go-fiber-core"
	}

	// 4. Consultar información del proceso para construir la clave de bloqueo
	// Necesitamos process_type_id para bloquear la concurrencia global de ese tipo de proceso
	var processTypeID int64
	log.Debug("Fetching process info for locking strategy", zap.Int64("process_version_id", processVersionID))

	var pv processlifecyclemanager.ProcessVersion
	err := dbRead.WithContext(ctx).
		Select("process_type_id").
		Where("id = ? AND archived_at IS NULL", processVersionID).
		First(&pv).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrNotFound
		}
		log.Error("Failed to fetch process info from DB",
			zap.Int64("process_version_id", processVersionID),
			zap.Error(err),
		)
		return err
	}
	processTypeID = pv.ProcessTypeID

	// Construcción de la clave de bloqueo "Blocker/Blacklist": app:lifecycle:block:{processTypeID}
	// Esta clave indicará a ResolveProcessVersion que DEBE ir a base de datos
	blockerKey := fmt.Sprintf("%s:lifecycle:block:%d", projectPrefix, processTypeID)

	// Construcción del patrón de claves a invalidar (para referencia o limpieza futura)
	// app:lifecycle:resolution:{processTypeID}:*
	// cachePattern := fmt.Sprintf("%s:lifecycle:resolution:%d:*", projectPrefix, processTypeID)

	lockService := cache.NewRedisLockService(redisClient)

	// 5. Establecer "Blocker" en Redis
	// Esto fuerza a todas las lecturas concurrentes a ir a BD mientras se procesa la promoción
	// y por un breve periodo después para asegurar consistencia.
	if redisClient != nil {
		log.Info("Setting Redis blocker key", zap.String("blocker_key", blockerKey))
		// TTL de 30 segundos debería ser suficiente para cubrir la transacción y propagación
		if err := lockService.Set(ctx, blockerKey, "1", 30*time.Second); err != nil {
			log.Warn("Failed to set Redis blocker", zap.String("key", blockerKey), zap.Error(err))
		} else {
			log.Debug("Redis blocker set successfully", zap.String("key", blockerKey))
		}
	} else {
		log.Warn("Redis client not available - skipping blocker setup")
	}

	log.Info("Executing PromoteProcessVersion (GORM transaction)",
		zap.Int64("process_version_id", processVersionID),
	)

	err = s.promoteSvc.PromoteProcessVersion(ctx, processVersionID, operatorID, comment)
	if err != nil {
		log.Error("PromoteProcessVersion failed",
			zap.Int64("process_version_id", processVersionID),
			zap.Error(err),
		)
		return err
	}

	// 7. Limpiar Caché (Invalidación por patrón)
	// Nota: El Blocker sigue activo por su TTL (30s) para asegurar que
	// cualquier nodo vea la promoción antes de volver a cachear.
	if redisClient != nil {
		// Aquí idealmente borraríamos todas las llaves que empiecen con "app:lifecycle:resolution:{processTypeID}:"
		// Por simplicidad, el blocker ya cumple la función de invalidación lógica.
		log.Debug("Redis promotion cleanup finished (blocker still active via TTL)")
	}

	log.Info("PromoteProcessVersion completed successfully",
		zap.Int64("process_version_id", processVersionID),
	)

	return nil
}

func (s *service) ResolveProcessVersion(ctx context.Context, processTypeID int64, sedeID int64, overrideProcessVersionID *int64, roadmap int, useCache bool) (int64, []Step, error) {
	if processTypeID <= 0 {
		return 0, nil, domain.ErrInvalidArgument
	}

	// Normalize overrideProcessVersionID: treat 0 as nil (no override)
	if overrideProcessVersionID != nil && *overrideProcessVersionID == 0 {
		overrideProcessVersionID = nil
	}

	redisClient := s.conn.ConnectRedis
	projectPrefix := os.Getenv("APP_NAME")
	if projectPrefix == "" {
		projectPrefix = "go-fiber-core"
	}

	var resolvedID int64
	var steps []Step

	// Estrategia de Caching:
	// 1. Si hay override (> 0), SIEMPRE vamos a BD (sin cache).
	// 2. Si es override=0 (PROD):
	//    a. Verificamos si existe "blocker" (app:lifecycle:block:{typeID}). Si existe -> BD.
	//    b. Si no hay blocker -> Intentamos Cache (app:lifecycle:resolution:{type}:{sede}:{roadmap}).
	//    c. Si no hay cache -> BD -> Guardar en Cache.

	shouldUseCache := useCache && overrideProcessVersionID == nil && redisClient != nil
	cacheKey := fmt.Sprintf("%s:lifecycle:resolution:%d:%d:%d", projectPrefix, processTypeID, sedeID, roadmap)
	blockerKey := fmt.Sprintf("%s:lifecycle:block:%d", projectPrefix, processTypeID)

	if shouldUseCache {
		lockService := cache.NewRedisLockService(redisClient)
		redisCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
		defer cancel()

		// 1. Verificar Blocker
		// Si existe el blocker, significa que hubo una promoción reciente y debemos ir a BD
		isBlocked, _ := lockService.Get(redisCtx, blockerKey)

		if len(isBlocked) > 0 {
			// Blocker activo -> forzar lectura de BD
			// No deshabilitamos shouldUseCache para permitir escritura posterior si quisiéramos,
			// pero según la regla "si está bloqueado no cachear", mejor lo deshabilitamos para esta request.
			shouldUseCache = false
		} else {
			// 2. Intentar leer del Cache Específico
			cached, err := lockService.Get(redisCtx, cacheKey)
			if err == nil && len(cached) > 0 {
				var payload struct {
					ProcessVersionID int64  `json:"process_version_id"`
					Steps            []Step `json:"steps"`
				}
				if err := json.Unmarshal([]byte(cached), &payload); err == nil {
					return payload.ProcessVersionID, payload.Steps, nil
				}
			}
		}
	}

	managerResolvedID, managerSteps, err := s.resolveSvc.ResolveProcessVersion(ctx, processTypeID, sedeID, overrideProcessVersionID, roadmap)
	if err != nil {
		return 0, nil, err
	}
	resolvedID = managerResolvedID
	steps = make([]Step, 0, len(managerSteps))
	for _, st := range managerSteps {
		steps = append(steps, Step{
			Name:         st.Name,
			ExecutionKey: st.ExecutionKey,
			Config:       st.Config,
			StepOrder:    st.StepOrder,
		})
	}

	if shouldUseCache {
		lockService := cache.NewRedisLockService(redisClient)

		payload := struct {
			ProcessVersionID int64  `json:"process_version_id"`
			Steps            []Step `json:"steps"`
		}{
			ProcessVersionID: resolvedID,
			Steps:            steps,
		}

		encoded, err := json.Marshal(payload)
		if err == nil {
			redisCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
			// Guardar en cache con expiración (ej. 1 hora o lo que prefieras)
			_ = lockService.Set(redisCtx, cacheKey, encoded, 1*time.Hour)
			cancel()
		}
	}

	return resolvedID, steps, nil
}

func (s *service) MoveProcessVersionToTest(ctx context.Context, processVersionID int64) error {
	return s.moveToTestSvc.MoveProcessVersionToTest(ctx, processVersionID)
}

func (s *service) ListProcessVersions(ctx context.Context, req dtos.PaginationRequest) (*dtos.PaginationResponse[models.ProcessVersionListItem], error) {
	db := s.conn.ConnectGormRead
	if db == nil {
		return nil, fmt.Errorf("gorm read connection is not initialized")
	}

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.RowsPerPage <= 0 {
		req.RowsPerPage = 15
	}

	base := db.WithContext(ctx).
		Table("process_versions pv").
		Joins("JOIN process_types pt ON pt.id = pv.process_type_id").
		Joins("LEFT JOIN users u ON u.id = pv.operator_id").
		Joins(`LEFT JOIN (
			SELECT
				h.process_version_id,
				h.promoted_at AS valid_from,
				LEAD(h.promoted_at) OVER (
					PARTITION BY v.process_type_id, v.sede_id
					ORDER BY h.promoted_at
				) AS valid_to
			FROM process_version_history h
			JOIN process_versions v ON v.id = h.process_version_id
		) hv ON hv.process_version_id = pv.id`).
		Where("pt.is_visible = ? AND pt.archived_at IS NULL AND pv.archived_at IS NULL", true)

	for i, f := range req.FilterBy {
		if i >= len(req.FilterValues) {
			continue
		}

		val := req.FilterValues[i]

		switch f {
		case "status":
			base = base.Where("pv.status = ?", val)
		case "sede_id":
			base = base.Where("pv.sede_id = ?", val)
		case "process_type_id":
			base = base.Where("pv.process_type_id = ?", val)
		case "name", "process_type_name":
			base = base.Where("pt.name ILIKE ?", fmt.Sprintf("%%%v%%", val))
		case "operator_email":
			base = base.Where("u.email ILIKE ?", fmt.Sprintf("%%%v%%", val))
		}
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, err
	}

	if total == 0 {
		return &dtos.PaginationResponse[models.ProcessVersionListItem]{
			Data:        []models.ProcessVersionListItem{},
			TotalRows:   0,
			TotalPages:  0,
			Page:        req.Page,
			RowsPerPage: req.RowsPerPage,
			Extras:      map[string]any{},
		}, nil
	}

	if len(req.SortBy) > 0 {
		for i, field := range req.SortBy {
			desc := false
			if i < len(req.SortDesc) {
				desc = req.SortDesc[i]
			}

			column := ""
			switch field {
			case "id":
				column = "pv.id"
			case "version_number":
				column = "pv.version_number"
			case "status":
				column = "pv.status"
			case "sede_id":
				column = "pv.sede_id"
			case "name", "process_type_name":
				column = "pt.name"
			case "operator_email":
				column = "u.email"
			default:
				continue
			}

			if desc {
				base = base.Order(column + " DESC")
			} else {
				base = base.Order(column + " ASC")
			}
		}
	} else {
		base = base.Order("pv.id DESC")
	}

	offset := (req.Page - 1) * req.RowsPerPage

	var items []models.ProcessVersionListItem
	if err := base.
		Select(`
			pv.id AS id,
			pt.name AS process_type_name,
			pt.is_visible AS process_type_is_visible,
			pv.version_number,
			pv.sede_id,
			pv.status,
			u.email AS operator_email,
			hv.valid_from,
			hv.valid_to`,
		).
		Limit(req.RowsPerPage).
		Offset(offset).
		Scan(&items).Error; err != nil {
		return nil, err
	}

	totalPages := 0
	if req.RowsPerPage > 0 {
		totalPages = int((total + int64(req.RowsPerPage) - 1) / int64(req.RowsPerPage))
	}

	return &dtos.PaginationResponse[models.ProcessVersionListItem]{
		Data:        items,
		TotalRows:   total,
		TotalPages:  totalPages,
		Page:        req.Page,
		RowsPerPage: req.RowsPerPage,
		Extras:      map[string]any{},
	}, nil
}

func (s *service) GetProcessVersionByID(ctx context.Context, processVersionID int64) (*models.ProcessVersionListItem, error) {
	if processVersionID <= 0 {
		return nil, domain.ErrInvalidArgument
	}

	db := s.conn.ConnectGormRead
	if db == nil {
		return nil, fmt.Errorf("gorm read connection is not initialized")
	}

	var row models.ProcessVersionRow

	err := db.WithContext(ctx).
		Table("process_versions pv").
		Select(`
			pv.id,
			pt.name AS process_type_name,
			pt.is_visible AS process_type_is_visible,
			pv.version_number,
			pv.sede_id,
			pv.status,
			u.email AS operator_email,
			hv.valid_from,
			hv.valid_to
		`).
		Joins("JOIN process_types pt ON pt.id = pv.process_type_id").
		Joins("LEFT JOIN users u ON u.id = pv.operator_id").
		Joins(`LEFT JOIN (
			SELECT
				h.process_version_id,
				h.promoted_at AS valid_from,
				LEAD(h.promoted_at) OVER (
					PARTITION BY v.process_type_id, v.sede_id
					ORDER BY h.promoted_at
				) AS valid_to
			FROM process_version_history h
			JOIN process_versions v ON v.id = h.process_version_id
		) hv ON hv.process_version_id = pv.id`).
		Where("pv.id = ? AND pv.archived_at IS NULL", processVersionID).
		Scan(&row).Error
	if err != nil {
		return nil, err
	}

	item := &models.ProcessVersionListItem{
		ID:                   row.ID,
		ProcessTypeName:      row.ProcessTypeName,
		ProcessTypeIsVisible: row.ProcessTypeIsVisible,
		VersionNumber:        row.VersionNumber,
		SedeID:               row.SedeID,
		Status:               row.Status,
		OperatorEmail:        row.OperatorEmail,
		ValidFrom:            row.ValidFrom,
		ValidTo:              row.ValidTo,
	}
	return item, nil
}

func (s *service) GetProcessStepsByVersionID(ctx context.Context, processVersionID int64) ([]Step, error) {
	if processVersionID <= 0 {
		return nil, domain.ErrInvalidArgument
	}

	db := s.conn.ConnectGormRead
	if db == nil {
		return nil, fmt.Errorf("gorm read connection is not initialized")
	}

	var steps []Step
	err := db.WithContext(ctx).
		Table("process_steps").
		Select(`
			name,
			execution_key,
			config,
			step_order
		`).
		Where("process_version_id = ?", processVersionID).
		Order("step_order ASC").
		Scan(&steps).Error
	if err != nil {
		return nil, err
	}

	if steps == nil {
		return []Step{}, nil
	}

	return steps, nil
}

func BuildServiceRegistryFromSteps(steps []Step) ([]serviceconfig.ServiceRegistryRow, error) {
	rows := make([]serviceconfig.ServiceRegistryRow, 0, len(steps))

	for _, step := range steps {
		errorTolerance := "inherit"
		var requiredKeys []string
		var timeout time.Duration

		if len(step.Config) > 0 {
			var cfg struct {
				ErrorTolerance string   `json:"error_tolerance"`
				RequiredKeys   []string `json:"required_keys"`
				TimeoutMs      int64    `json:"timeout_ms"`
			}
			if err := json.Unmarshal(step.Config, &cfg); err != nil {
				return nil, domain.ErrInternal
			}

			if cfg.ErrorTolerance != "" {
				e := strings.ToLower(cfg.ErrorTolerance)
				switch e {
				case "critical", "tolerable", "inherit":
					errorTolerance = e
				default:
					errorTolerance = "inherit"
				}
			}

			if len(cfg.RequiredKeys) > 0 {
				requiredKeys = cfg.RequiredKeys
			}

			if cfg.TimeoutMs > 0 {
				timeout = time.Duration(cfg.TimeoutMs) * time.Millisecond
			}
		}

		row := serviceconfig.ServiceRegistryRow{
			Path:           step.ExecutionKey,
			Order:          int(step.StepOrder),
			ErrorTolerance: errorTolerance,
			Config:         step.Config,
			RequiredKeys:   requiredKeys,
			Timeout:        timeout,
		}
		rows = append(rows, row)
	}

	return rows, nil
}

func (s *service) Run(ctx context.Context, req requests.RunProcessRequest) (int64, *contracts.ServiceContext, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// Validate required fields (Strict Validation)
	if req.SedeID == nil {
		return 0, nil, domain.ErrInvalidArgument
	}
	if req.OverrideProcessVersionID == nil {
		return 0, nil, domain.ErrInvalidArgument
	}
	if req.Roadmap == nil {
		return 0, nil, domain.ErrInvalidArgument
	}
	if req.Input == nil {
		return 0, nil, domain.ErrInvalidArgument
	}

	// Inject context variables from request
	req.Input["sede_id"] = *req.SedeID
	req.Input["roadmap"] = *req.Roadmap
	if req.OperatorID > 0 {
		req.Input["operator_id"] = req.OperatorID
	}

	// UseCache = true:
	// - If override_process_version_id is set, ResolveProcessVersion logic IGNORES cache and goes to DB.
	// - If override_process_version_id is nil (Production), it uses Redis for performance.
	processVersionID, steps, err := s.ResolveProcessVersion(ctx, req.ProcessTypeID, *req.SedeID, req.OverrideProcessVersionID, *req.Roadmap, true)
	if err != nil {
		return 0, nil, err
	}

	registryRows, err := BuildServiceRegistryFromSteps(steps)
	if err != nil {
		return 0, nil, err
	}

	serviceCtx := contracts.NewServiceContextFromInput(ctx, req.Input)

	// Inicializar métricas solo si es modo Test (override > 0)
	isTestMode := req.OverrideProcessVersionID != nil && *req.OverrideProcessVersionID > 0
	if isTestMode {
		serviceCtx.Metrics = &contracts.ExecutionMetrics{
			ExecutionID:     uuid.New().String(), // Requiere "github.com/google/uuid"
			GoroutinesCount: runtime.NumGoroutine(),
		}
		// Iniciar medición de tiempo
		start := time.Now()
		defer func() {
			serviceCtx.Metrics.TotalDurationMs = time.Since(start).Milliseconds()
			serviceCtx.Metrics.GoroutinesCount = runtime.NumGoroutine() // Actualizar al final

			// Medir memoria
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			serviceCtx.Metrics.MemoryUsedMB = float64(m.Alloc) / 1024 / 1024
		}()

		// Inyectar el ServiceContext en el contexto de Go para que GORM lo vea
		ctx = context.WithValue(ctx, contextkeys.DBMetricsCollectorKey, serviceCtx)
		serviceCtx.Ctx = ctx // Actualizar también dentro del struct
	}

	if err := serviceconfig.ExecuteServicesInOrder(ctx, registryRows, serviceCtx); err != nil {
		return processVersionID, serviceCtx, err
	}

	return processVersionID, serviceCtx, nil
}

 
