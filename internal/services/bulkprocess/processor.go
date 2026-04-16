package bulkprocess

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"

	"go-fiber-core/internal/models"
	"go-fiber-core/internal/repositories/bulkjobitemmessage"
	"go-fiber-core/internal/services/batchflow"

	"gorm.io/gorm"
)

type bulkJobProcessor struct {
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

func NewBulkJobProcessor(writeDB *gorm.DB) *bulkJobProcessor {
	return &bulkJobProcessor{
		writeDB:       writeDB,
		messageWriter: bulkjobitemmessage.NewBulkJobItemMessageWriterRepo(),
	}
}

func (p *bulkJobProcessor) ProcessBatch(ctx context.Context, execCtx batchflow.ExecutionContext, batch batchflow.Batch) (batchflow.ProcessBatchResult, error) {
	items, previewItems, err := p.resolveItems(execCtx, batch)
	if err != nil {
		return batchflow.ProcessBatchResult{}, err
	}

	if err := p.writeDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i, item := range items {
			preview := previewItems[i]
			var lastDetail *string
			if preview.Message != "" {
				message := preview.Message
				lastDetail = &message
			}
			if err := tx.Model(&models.BulkJobItem{}).
				Where("id = ?", item.ID).
				Updates(map[string]any{
					"status_code":         preview.Status,
					"last_detail_message": lastDetail,
				}).Error; err != nil {
				return err
			}
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

func (p *bulkJobProcessor) PreviewBatch(ctx context.Context, execCtx batchflow.ExecutionContext, batch batchflow.Batch) (batchflow.PreviewBatchResult, error) {
	_, previewItems, err := p.resolveItems(execCtx, batch)
	if err != nil {
		return batchflow.PreviewBatchResult{}, err
	}
	return batchflow.PreviewBatchResult{
		Items:          previewItems,
		ProcessedCount: len(previewItems),
	}, nil
}

func (p *bulkJobProcessor) resolveItems(execCtx batchflow.ExecutionContext, batch batchflow.Batch) ([]batchItemPayload, []batchflow.PreviewItemResult, error) {
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
		"El proveedor externo rechazó el registro por datos inconsistentes",
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
