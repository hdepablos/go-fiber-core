package lifecycle

import (
	"context"
	"encoding/json"

	"go-fiber-core/internal/models"
	"go-fiber-core/internal/services/exportmanager"

	"gorm.io/gorm"
)

type outputRegistrar struct {
	writeDB *gorm.DB
}

// NewOutputRegistrar crea la pieza que persiste el archivo final generado por el export.
func NewOutputRegistrar(writeDB *gorm.DB) exportmanager.OutputRegistrar {
	return &outputRegistrar{writeDB: writeDB}
}

// Register guarda la referencia final del archivo y su metadata operativa en la tabla destino.
func (r *outputRegistrar) Register(ctx context.Context, execCtx exportmanager.ExecutionContext, output exportmanager.OutputResult) error {
	metadata, err := json.Marshal(map[string]any{
		"bucket":        output.Bucket,
		"key":           output.Key,
		"content_type":  output.ContentType,
		"parts":         output.Parts,
		"total_records": execCtx.Summary.TotalRecords,
		"total_amount":  execCtx.Summary.TotalAmount,
		"redis_key":     execCtx.Input.RedisKey,
	})
	if err != nil {
		return err
	}

	fileSize := output.FileSize
	record := &models.BulkJobOutput{
		BulkJobID: execCtx.Input.ParentID,
		Type:      "csv",
		FilePath:  output.Path,
		FileSize:  &fileSize,
		Status:    models.BulkJobOutputStatusGenerated,
		Metadata:  metadata,
	}
	return r.writeDB.WithContext(ctx).Create(record).Error
}
