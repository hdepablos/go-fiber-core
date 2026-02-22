package processlifecycle

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/dtos/connect"
)

type Service interface {
	ReplicateProcessVersion(ctx context.Context, processVersionID int64) (int64, error)
	PromoteProcessVersion(ctx context.Context, processVersionID int64, operatorID int64, comment string) error
	ResolveProcessVersion(ctx context.Context, processTypeID int64, sedeID int64, overrideProcessVersionID *int64) (int64, []Step, error)
	MoveProcessVersionToTest(ctx context.Context, processVersionID int64) error
}

type Step struct {
	Name         string          `json:"name"`
	ExecutionKey string          `json:"execution_key"`
	Config       json.RawMessage `json:"config"`
	StepOrder    int32           `json:"step_order"`
}

type service struct {
	conn *connect.ConnectDTO
}

func NewService(conn *connect.ConnectDTO) Service {
	return &service{
		conn: conn,
	}
}

func (s *service) ReplicateProcessVersion(ctx context.Context, processVersionID int64) (int64, error) {
	if processVersionID <= 0 {
		return 0, domain.ErrInvalidArgument
	}

	db := s.conn.ConnectPgxWrite
	if db == nil {
		return 0, fmt.Errorf("pgx write connection is not initialized")
	}

	var newVersionID int64
	err := db.
		QueryRow(ctx, `SELECT replicate_process_version($1)`, processVersionID).
		Scan(&newVersionID)
	if err != nil {
		return 0, mapPgxError(err)
	}

	return newVersionID, nil
}

func (s *service) PromoteProcessVersion(ctx context.Context, processVersionID int64, operatorID int64, comment string) error {
	if processVersionID <= 0 || operatorID <= 0 {
		return domain.ErrInvalidArgument
	}

	db := s.conn.ConnectPgxWrite
	if db == nil {
		return fmt.Errorf("pgx write connection is not initialized")
	}

	_, err := db.Exec(ctx, `SELECT promote_process_version($1, $2, $3)`, processVersionID, operatorID, comment)
	if err != nil {
		return mapPgxError(err)
	}
	return nil
}

func (s *service) ResolveProcessVersion(ctx context.Context, processTypeID int64, sedeID int64, overrideProcessVersionID *int64) (int64, []Step, error) {
	if processTypeID <= 0 {
		return 0, nil, domain.ErrInvalidArgument
	}

	db := s.conn.ConnectPgxWrite
	if db == nil {
		return 0, nil, fmt.Errorf("pgx write connection is not initialized")
	}

	var resolvedID int64
	var stepsJSON []byte

	if overrideProcessVersionID != nil {
		err := db.
			QueryRow(ctx, `SELECT process_version_id, process_steps FROM resolve_process_version($1, $2, $3)`, processTypeID, sedeID, *overrideProcessVersionID).
			Scan(&resolvedID, &stepsJSON)
		if err != nil {
			return 0, nil, mapPgxError(err)
		}
	} else {
		err := db.
			QueryRow(ctx, `SELECT process_version_id, process_steps FROM resolve_process_version($1, $2, NULL)`, processTypeID, sedeID).
			Scan(&resolvedID, &stepsJSON)
		if err != nil {
			return 0, nil, mapPgxError(err)
		}
	}

	var steps []Step
	if len(stepsJSON) == 0 {
		steps = []Step{}
	} else {
		if err := json.Unmarshal(stepsJSON, &steps); err != nil {
			return 0, nil, domain.ErrInternal
		}
	}

	return resolvedID, steps, nil
}

func (s *service) MoveProcessVersionToTest(ctx context.Context, processVersionID int64) error {
	if processVersionID <= 0 {
		return domain.ErrInvalidArgument
	}

	db := s.conn.ConnectPgxWrite
	if db == nil {
		return fmt.Errorf("pgx write connection is not initialized")
	}

	_, err := db.Exec(ctx, `SELECT move_process_version_to_test($1)`, processVersionID)
	if err != nil {
		return mapPgxError(err)
	}

	return nil
}

func mapPgxError(err error) error {
	if err == nil {
		return nil
	}

	msg := err.Error()

	switch {
	case strings.Contains(msg, "Process version not found or archived"):
		return domain.ErrNotFound
	case strings.Contains(msg, "Process type does not exist or is archived"):
		return domain.ErrNotFound
	case strings.Contains(msg, "No active version found"):
		return domain.ErrNotFound
	case strings.Contains(msg, "Override version invalid"):
		return domain.ErrInvalidArgument
	case strings.Contains(msg, "Only DRAFT versions can be moved to TEST"):
		return domain.ErrInvalidArgument
	case strings.Contains(msg, "Cannot promote version without steps"),
		strings.Contains(msg, "Promotion comment exceeds 300 characters"):
		return domain.ErrInvalidArgument
	default:
		return domain.ErrInternal
	}
}
