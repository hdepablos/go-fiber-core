package imports

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/dtos/connect"
	"go-fiber-core/internal/models"
	bulkJobRepo "go-fiber-core/internal/repositories/bulkjob"
	bulkJobConfigRepo "go-fiber-core/internal/repositories/bulkjobconfig"
	bulkJobItemRepo "go-fiber-core/internal/repositories/bulkjobitem"

	"gorm.io/gorm"
)

type Service interface {
	Process(ctx context.Context, operatorID uint64, branchID int64, refCode string, total int, keyCode string, fileName string, file io.Reader) error
}

type service struct {
	conn       *connect.ConnectDTO
	jobReader  bulkJobRepo.BulkJobReader
	jobWriter  bulkJobRepo.BulkJobWriter
	itemReader bulkJobItemRepo.BulkJobItemReader
	itemWriter bulkJobItemRepo.BulkJobItemWriter
	cfgReader  bulkJobConfigRepo.BulkJobConfigReader
}

//	Config pgx | gorm
const bulkJobItemsInsertDriver = "pgx"

func NewService(
	conn *connect.ConnectDTO,
	jobReader bulkJobRepo.BulkJobReader,
	jobWriter bulkJobRepo.BulkJobWriter,
	itemReader bulkJobItemRepo.BulkJobItemReader,
	itemWriter bulkJobItemRepo.BulkJobItemWriter,
	cfgReader bulkJobConfigRepo.BulkJobConfigReader,
) Service {
	return &service{
		conn:       conn,
		jobReader:  jobReader,
		jobWriter:  jobWriter,
		itemReader: itemReader,
		itemWriter: itemWriter,
		cfgReader:  cfgReader,
	}
}

func (s *service) Process(ctx context.Context, operatorID uint64, branchID int64, refCode string, total int, keyCode string, fileName string, file io.Reader) error {
	refCode = strings.TrimSpace(refCode)
	keyCode = strings.TrimSpace(keyCode)
	if refCode == "" || keyCode == "" || total < 0 {
		return domain.ErrInvalidArgument
	}

	adjustedTotal, delimiter, referenceField, footerLines, err := s.resolveConfig(ctx, operatorID, refCode, total)
	if err != nil {
		return err
	}

	job, err := s.getOrCreateJob(ctx, operatorID, branchID, refCode, keyCode, adjustedTotal, fileName)
	if err != nil {
		return err
	}

	maxRow, err := s.itemReader.GetMaxRowNumber(ctx, s.conn.ConnectGormRead, job.ID)
	if err != nil {
		return err
	}

	br := bufio.NewReader(file)
	headerLineBytes, err := br.ReadBytes('\n')
	if err != nil {
		if errors.Is(err, io.EOF) && len(headerLineBytes) > 0 {
			// seguimos, header sin newline final
		} else {
			return domain.ErrInvalidArgument
		}
	}

	headerLine := strings.TrimRight(string(headerLineBytes), "\r\n")
	if delimiter == ',' {
		if detected, ok := detectDelimiter(headerLine); ok {
			delimiter = detected
		}
	}

	reader := csv.NewReader(io.MultiReader(strings.NewReader(headerLine+"\n"), br))
	reader.Comma = delimiter
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	header, err := reader.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return domain.ErrInvalidArgument
		}
		return fmt.Errorf("error leyendo header csv: %w", err)
	}
	for i := range header {
		header[i] = strings.TrimSpace(header[i])
	}

	var pendingFooter [][]string
	if footerLines > 0 {
		pendingFooter = make([][]string, 0, footerLines+1)
	}

	rowNumber := maxRow
	if rowNumber == 0 {
		rowNumber = 1
	}
	flushEvery := 5000
	batch := make([]*models.BulkJobItem, 0, flushEvery)

	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		if bulkJobItemsInsertDriver == "pgx" && s.conn.ConnectPgxWrite != nil {
			if err := s.itemWriter.BulkCreatePGX(ctx, s.conn.ConnectPgxWrite, batch); err != nil {
				return err
			}
		} else {
			if err := s.itemWriter.BulkCreate(ctx, s.conn.ConnectGormWrite, batch); err != nil {
				return err
			}
		}
		batch = batch[:0]
		return nil
	}

	buildItem := func(record []string) (*models.BulkJobItem, error) {
		rowNumber++
		row := make(map[string]any, len(header))
		for i, key := range header {
			if key == "" {
				key = fmt.Sprintf("col_%d", i+1)
			}
			if i < len(record) {
				row[key] = strings.TrimSpace(record[i])
			} else {
				row[key] = ""
			}
		}

		refKey := pickReferenceKey(row, header, referenceField)
		if refKey == "" {
			refKey = fmt.Sprintf("row_%d", rowNumber)
		}

		raw, err := json.Marshal(row)
		if err != nil {
			return nil, err
		}
		return &models.BulkJobItem{
			BulkJobID:    job.ID,
			RowNumber:    rowNumber,
			ReferenceKey: refKey,
			Data:         raw,
			StatusCode:   models.BulkJobStatusImported,
		}, nil
	}

	for {
		record, err := reader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("error leyendo csv: %w", err)
		}

		if footerLines > 0 {
			pendingFooter = append(pendingFooter, record)
			if len(pendingFooter) <= footerLines {
				continue
			}
			record = pendingFooter[0]
			pendingFooter = pendingFooter[1:]
		}

		item, err := buildItem(record)
		if err != nil {
			return err
		}
		batch = append(batch, item)
		if len(batch) >= flushEvery {
			if err := flush(); err != nil {
				return err
			}
		}
	}

	if err := flush(); err != nil {
		return err
	}

	if err := s.jobWriter.UpdateStatus(ctx, s.conn.ConnectGormWrite, job.ID, models.BulkJobStatusImported); err != nil {
		return err
	}

	return nil
}

func (s *service) getOrCreateJob(ctx context.Context, operatorID uint64, branchID int64, refCode string, keyCode string, total int, fileName string) (*models.BulkJob, error) {
	dbRead := s.conn.ConnectGormRead
	existing, err := s.jobReader.GetByKeyCode(ctx, dbRead, keyCode)
	if err == nil && existing != nil {
		if strings.TrimSpace(existing.RefCode) != refCode {
			return nil, domain.ErrInvalidArgument
		}

		dbWrite := s.conn.ConnectGormWrite
		updates := map[string]any{}
		if existing.BranchID == 0 && branchID != 0 {
			updates["branch_id"] = branchID
			existing.BranchID = branchID
		}
		if existing.TotalDetailItems == 0 && total > 0 {
			updates["total_detail_items"] = total
			existing.TotalDetailItems = total
		}
		if existing.FileName == nil && strings.TrimSpace(fileName) != "" {
			v := filepath.Base(strings.TrimSpace(fileName))
			updates["file_name"] = v
			existing.FileName = &v
		}

		if len(updates) > 0 {
			if err := dbWrite.WithContext(ctx).Model(&models.BulkJob{}).Where("id = ?", existing.ID).Updates(updates).Error; err != nil {
				return nil, err
			}
		}
		return existing, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	var f *string
	if strings.TrimSpace(fileName) != "" {
		v := filepath.Base(strings.TrimSpace(fileName))
		f = &v
	}

	key := keyCode
	job := &models.BulkJob{
		OperatorID:          operatorID,
		BranchID:            branchID,
		KeyCode:             &key,
		RefCode:             refCode,
		StatusCode:          models.BulkJobStatusImporting,
		TotalDetailItems:    total,
		TotalProcessedItems: 0,
		FileName:            f,
	}
	if err := s.jobWriter.Create(ctx, s.conn.ConnectGormWrite, job); err != nil {
		return nil, err
	}
	return job, nil
}

func (s *service) resolveConfig(ctx context.Context, operatorID uint64, refCode string, total int) (int, rune, string, int, error) {
	delimiter := rune(',')
	referenceField := ""
	footerLines := 0

	dbRead := s.conn.ConnectGormRead
	cfg, err := s.cfgReader.GetActiveByRefCode(ctx, dbRead, operatorID, refCode)
	if err != nil {
		return total, delimiter, referenceField, footerLines, nil
	}

	var raw map[string]any
	if err := json.Unmarshal(cfg.Config, &raw); err != nil {
		return total, delimiter, referenceField, footerLines, nil
	}

	if v, ok := raw["number_lines_header"]; ok {
		if n, ok := asInt(v); ok && n > 0 {
			total -= n
		}
	}
	if v, ok := raw["number_lines_footer"]; ok {
		if n, ok := asInt(v); ok && n > 0 {
			total -= n
			footerLines = n
		}
	}
	if v, ok := raw["number_lines"]; ok {
		if n, ok := asInt(v); ok && n > 0 {
			total = int(float64(total) / float64(n))
		}
	}
	if v, ok := raw["delimiter"]; ok {
		if s, ok := v.(string); ok && s != "" {
			delimiter = []rune(s)[0]
		}
	}
	if v, ok := raw["reference_field"]; ok {
		if s, ok := v.(string); ok {
			referenceField = strings.TrimSpace(s)
		}
	}

	if total < 0 {
		total = 0
	}
	return total, delimiter, referenceField, footerLines, nil
}

func asInt(v any) (int, bool) {
	switch t := v.(type) {
	case int:
		return t, true
	case int64:
		return int(t), true
	case float64:
		return int(t), true
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(t))
		return i, err == nil
	default:
		return 0, false
	}
}

func pickReferenceKey(row map[string]any, header []string, referenceField string) string {
	if referenceField != "" {
		if v, ok := row[referenceField]; ok {
			if s, ok := v.(string); ok {
				return strings.TrimSpace(s)
			}
		}
	}

	if v, ok := row["reference_key"]; ok {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}

	candidates := []string{"reference_key", "reference", "cbu", "customer_id", "dni", "cuit"}
	for _, c := range candidates {
		if v, ok := row[c]; ok {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}

	for _, key := range header {
		if v, ok := row[key]; ok {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}

	return ""
}

func detectDelimiter(headerLine string) (rune, bool) {
	count := func(sep string) int { return strings.Count(headerLine, sep) }
	semi := count(";")
	comma := count(",")
	tab := count("\t")

	if semi == 0 && comma == 0 && tab == 0 {
		return ',', false
	}
	if semi >= comma && semi >= tab {
		return ';', true
	}
	if tab >= comma {
		return '\t', true
	}
	return ',', true
}
