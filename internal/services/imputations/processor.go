package imputations

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

type batchItemPayload struct {
	ID                int64                `json:"id"`
	BulkJobID         int64                `json:"bulk_job_id"`
	RowNumber         int                  `json:"row_number"`
	ReferenceKey      string               `json:"reference_key"`
	StatusCode        models.BulkJobStatus `json:"status_code"`
	LastDetailMessage *string              `json:"last_detail_message"`
	Data              json.RawMessage      `json:"data"`
}

func NewProcessor(writeDB *gorm.DB) *processor {
	return &processor{
		writeDB:       writeDB,
		messageWriter: bulkjobitemmessage.NewBulkJobItemMessageWriterRepo(),
	}
}

func (p *processor) ProcessBatch(ctx context.Context, execCtx batchflow.ExecutionContext, batch batchflow.Batch) (batchflow.ProcessBatchResult, error) {
	items, previewItems, err := p.resolveItems(execCtx, batch)
	if err != nil {
		return batchflow.ProcessBatchResult{}, err
	}

	if err := p.writeDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := updateBatchItemStatuses(ctx, tx, items, previewItems); err != nil {
			return err
		}
		for i, item := range items {
			preview := previewItems[i]
			for _, msg := range preview.Messages {
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

func (p *processor) PreviewBatch(ctx context.Context, execCtx batchflow.ExecutionContext, batch batchflow.Batch) (batchflow.PreviewBatchResult, error) {
	_, previewItems, err := p.resolveItems(execCtx, batch)
	if err != nil {
		return batchflow.PreviewBatchResult{}, err
	}
	return batchflow.PreviewBatchResult{
		Items:          previewItems,
		ProcessedCount: len(previewItems),
	}, nil
}

func (p *processor) resolveItems(execCtx batchflow.ExecutionContext, batch batchflow.Batch) ([]batchItemPayload, []batchflow.PreviewItemResult, error) {
	items := make([]batchItemPayload, 0, len(batch.Items))
	results := make([]batchflow.PreviewItemResult, 0, len(batch.Items))
	for _, raw := range batch.Items {
		var item batchItemPayload
		if err := json.Unmarshal(raw, &item); err != nil {
			return nil, nil, fmt.Errorf("unmarshal batch item: %w", err)
		}
		items = append(items, item)
		results = append(results, buildPreviewResult(execCtx.Input.RedisKey, item))
	}
	return items, results, nil
}

func updateBatchItemStatuses(ctx context.Context, tx *gorm.DB, items []batchItemPayload, previewItems []batchflow.PreviewItemResult) error {
	if len(items) == 0 {
		return nil
	}

	var statusCase strings.Builder
	var detailCase strings.Builder
	args := make([]any, 0, (len(items)*4)+1)
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
		args = append(args, item.ID, preview.Status)

		detailCase.WriteString(" WHEN ? THEN ?")
		args = append(args, item.ID, lastDetail)

		ids = append(ids, item.ID)
	}
	statusCase.WriteString(" ELSE status_code END")
	detailCase.WriteString(" ELSE last_detail_message END")
	args = append(args, ids)

	query := fmt.Sprintf(`
		UPDATE bulk_job_items
		SET status_code = %s,
		    last_detail_message = %s,
		    updated_at = NOW()
		WHERE id IN ?
	`, statusCase.String(), detailCase.String())

	return tx.WithContext(ctx).Exec(query, args...).Error
}

// Reemplaza esta logica por la decision real del proceso.
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

func hashBucket(redisKey string, itemID int64, rowNumber int) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(fmt.Sprintf("%s:%d:%d", redisKey, itemID, rowNumber)))
	return h.Sum32() % 100
}

func errorMessage(item batchItemPayload, bucket uint32) string {
	options := []string{
		"Error validando el registro contra la politica del proveedor",
		"El proveedor externo rechazo el registro por datos inconsistentes",
		"No fue posible procesar el registro por una regla de negocio",
	}
	return fmt.Sprintf("%s (item_id=%d, row=%d, bucket=%d)", options[int(bucket)%len(options)], item.ID, item.RowNumber, bucket)
}

func detailMessage(item batchItemPayload, bucket uint32) string {
	options := []string{
		"Registro procesado con observaciones",
		"Registro procesado con ajuste informado por el proveedor",
		"Registro procesado con detalle operativo",
	}
	return fmt.Sprintf("%s (item_id=%d, row=%d, bucket=%d)", options[int(bucket)%len(options)], item.ID, item.RowNumber, bucket)
}
