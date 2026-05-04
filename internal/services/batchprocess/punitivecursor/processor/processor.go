package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strings"

	"go-fiber-core/internal/models"
	"go-fiber-core/internal/repositories/bulkjobitemmessage"
	"go-fiber-core/internal/services/batchflow"

	"gorm.io/gorm"
)

type processor struct {
	writeDB       *gorm.DB
	messageWriter bulkjobitemmessage.BulkJobItemMessageWriter
}

// batchItemPayload representa el registro del detalle tal como viaja dentro de cada batch.
type batchItemPayload struct {
	ID                int64                `json:"id"`
	BulkJobID         int64                `json:"bulk_job_id"`
	RowNumber         int                  `json:"row_number"`
	ReferenceKey      string               `json:"reference_key"`
	StatusCode        models.BulkJobStatus `json:"status_code"`
	LastDetailMessage *string              `json:"last_detail_message"`
	Data              json.RawMessage      `json:"data"`
}

// NewProcessor crea la pieza que delega la logica de negocio lote por lote y luego persiste en bloque.
func NewProcessor(writeDB *gorm.DB) *processor {
	return &processor{
		writeDB:       writeDB,
		messageWriter: bulkjobitemmessage.NewBulkJobItemMessageWriterRepo(),
	}
}

// ProcessBatch recibe el lote completo, ejecuta processBatchOriented y luego persiste status/mensajes en bloque.
func (p *processor) ProcessBatch(ctx context.Context, execCtx batchflow.ExecutionContext, batch batchflow.Batch) (batchflow.ProcessBatchResult, error) {
	items, err := resolveItems(batch)
	if err != nil {
		return batchflow.ProcessBatchResult{}, err
	}
	previewItems, err := processBatchOriented(ctx, execCtx, items)
	if err != nil {
		return batchflow.ProcessBatchResult{}, err
	}

	if err := p.writeDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Primero se actualiza el detalle y luego se guardan mensajes operativos complementarios.
		if err := updateBatchItemStatuses(ctx, tx, items, previewItems); err != nil {
			return err
		}
		for i, item := range items {
			preview := previewItems[i]
			for _, msg := range preview.Messages {
				// Cada mensaje detallado queda persistido para auditoria y soporte operativo.
				record := &models.BulkJobItemMessage{
					BulkJobItemID: item.ID,
					Severity:      msg.Severity,
					DetailMessage: msg.DetailMessage,
				}
				if msg.Code != "" {
					code := msg.Code
					record.Code = &code
				}
				if len(msg.Meta) > 0 {
					metaBytes, err := json.Marshal(msg.Meta)
					if err != nil {
						return err
					}
					record.Meta = &metaBytes
				}
				if err := p.messageWriter.Create(ctx, tx, record); err != nil {
					return err
				}
			}
		}
		return nil
	}); err != nil {
		return batchflow.ProcessBatchResult{}, err
	}

	return batchflow.ProcessBatchResult{
		ProcessedCount: len(items),
	}, nil
}

// PreviewBatch reutiliza la misma estrategia por lote sin escribir en base de datos.
func (p *processor) PreviewBatch(ctx context.Context, execCtx batchflow.ExecutionContext, batch batchflow.Batch) (batchflow.PreviewBatchResult, error) {
	items, err := resolveItems(batch)
	if err != nil {
		return batchflow.PreviewBatchResult{}, err
	}
	previewItems, err := processBatchOriented(ctx, execCtx, items)
	if err != nil {
		return batchflow.PreviewBatchResult{}, err
	}
	return batchflow.PreviewBatchResult{
		Items:          previewItems,
		ProcessedCount: len(previewItems),
	}, nil
}

// resolveItems deserializa el lote completo para que el servicio batch-oriented trabaje en bloque.
func resolveItems(batch batchflow.Batch) ([]batchItemPayload, error) {
	items := make([]batchItemPayload, 0, len(batch.Items))
	for _, raw := range batch.Items {
		var item batchItemPayload
		if err := json.Unmarshal(raw, &item); err != nil {
			return nil, fmt.Errorf("unmarshal batch item: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

// processBatchOriented es el punto de extension del developer para la logica por lote.
// La version scaffold deja una implementacion deterministica para que compile desde el minuto cero.
func processBatchOriented(ctx context.Context, execCtx batchflow.ExecutionContext, items []batchItemPayload) ([]batchflow.PreviewItemResult, error) {
	_ = ctx
	results := make([]batchflow.PreviewItemResult, 0, len(items))
	for _, item := range items {
		results = append(results, buildPreviewResult(execCtx.Input.RedisKey, item))
	}
	return results, nil
}

// updateBatchItemStatuses es donde se persiste el status del detalle y el ultimo mensaje por item.
func updateBatchItemStatuses(ctx context.Context, tx *gorm.DB, items []batchItemPayload, previewItems []batchflow.PreviewItemResult) error {
	if len(items) == 0 {
		return nil
	}

	var statusCase strings.Builder
	var detailCase strings.Builder
	statusArgs := make([]any, 0, len(items)*2)
	detailArgs := make([]any, 0, len(items)*2)
	ids := make([]int64, 0, len(items))

	statusCase.WriteString("CASE id")
	detailCase.WriteString("CASE id")
	for i, item := range items {
		preview := previewItems[i]
		var lastDetail any
		if preview.Message != "" {
			lastDetail = preview.Message
		}

		statusCase.WriteString(" WHEN ? THEN ?")
		statusArgs = append(statusArgs, item.ID, preview.Status)

		detailCase.WriteString(" WHEN ? THEN ?")
		detailArgs = append(detailArgs, item.ID, lastDetail)

		ids = append(ids, item.ID)
	}
	statusCase.WriteString(" ELSE status_code END")
	detailCase.WriteString(" ELSE last_detail_message END")
	args := make([]any, 0, len(statusArgs)+len(detailArgs)+1)
	args = append(args, statusArgs...)
	args = append(args, detailArgs...)
	args = append(args, ids)

	query := fmt.Sprintf(
		"UPDATE bulk_job_items "+
			"SET status_code = %s, "+
			"last_detail_message = %s, "+
			"updated_at = NOW() "+
			"WHERE id IN ?",
		statusCase.String(),
		detailCase.String(),
	)

	return tx.WithContext(ctx).Exec(query, args...).Error
}

// Reemplaza esta logica por la decision real del proceso.
// Aqui el developer recibe el lote completo y puede agrupar ids para resolver updates masivos.
// Se deja una version deterministica para que el scaffold sea ejecutable desde el minuto cero.
func buildPreviewResult(redisKey string, item batchItemPayload) batchflow.PreviewItemResult {
	bucket := hashBucket(redisKey, item.ID, item.RowNumber)
	status := models.BulkJobStatusProcessed
	message := ""
	messages := []batchflow.PreviewMessage{}

	switch {
	case bucket < 3:
		status = models.BulkJobStatusErrorProcess
		message = errorMessage(item, bucket)
		messages = append(messages, batchflow.PreviewMessage{
			Severity:      "ERROR",
			Code:          "ERROR_PROCESS",
			DetailMessage: message,
			Meta:          map[string]any{"bucket": bucket},
		})
	case bucket < 30:
		status = models.BulkJobStatusProcessedWithDetails
		message = detailMessage(item, bucket)
		messages = append(messages, batchflow.PreviewMessage{
			Severity:      "WARNING",
			Code:          "DETAIL_PROCESS",
			DetailMessage: message,
			Meta:          map[string]any{"bucket": bucket},
		})
	}

	return batchflow.PreviewItemResult{
		ItemID:       item.ID,
		RowNumber:    item.RowNumber,
		ReferenceKey: item.ReferenceKey,
		Status:       string(status),
		Message:      message,
		Messages:     messages,
		Metadata: map[string]any{
			"bulk_job_id": item.BulkJobID,
		},
	}
}

// hashBucket deja una salida deterministica para el scaffold sin acoplarlo a una regla real de negocio.
func hashBucket(redisKey string, itemID int64, rowNumber int) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(fmt.Sprintf("%s:%d:%d", redisKey, itemID, rowNumber)))
	return h.Sum32() % 100
}

// errorMessage fabrica un mensaje de error de ejemplo cuando el item cae en un bucket de error.
func errorMessage(item batchItemPayload, bucket uint32) string {
	options := []string{
		"Error validando el registro contra la politica del proveedor",
		"El proveedor externo rechazo el registro por datos inconsistentes",
		"No fue posible procesar el registro por una regla de negocio",
	}
	return fmt.Sprintf("%s (item_id=%d, row=%d, bucket=%d)", options[int(bucket)%len(options)], item.ID, item.RowNumber, bucket)
}

// detailMessage fabrica un mensaje de detalle de ejemplo cuando el item requiere observaciones.
func detailMessage(item batchItemPayload, bucket uint32) string {
	options := []string{
		"Registro procesado con observaciones",
		"Registro procesado con ajuste informado por el proveedor",
		"Registro procesado con detalle operativo",
	}
	return fmt.Sprintf("%s (item_id=%d, row=%d, bucket=%d)", options[int(bucket)%len(options)], item.ID, item.RowNumber, bucket)
}
