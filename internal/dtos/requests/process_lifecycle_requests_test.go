package requests

import (
	"testing"

	validator "github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

func TestResolveScenarioRequest_Validation(t *testing.T) {
	validate := validator.New()

	t.Run("SedeID=0 (Global) should be valid", func(t *testing.T) {
		roadmap := 0
		req := ResolveScenarioRequest{
			ProcessTypeID: 1,
			SedeID:        0,
			Roadmap:       &roadmap,
		}

		err := validate.Struct(req)
		assert.NoError(t, err)
	})

	t.Run("SedeID>0 (Specific) should be valid", func(t *testing.T) {
		roadmap := 0
		req := ResolveScenarioRequest{
			ProcessTypeID: 1,
			SedeID:        5,
			Roadmap:       &roadmap,
		}

		err := validate.Struct(req)
		assert.NoError(t, err)
	})

	t.Run("SedeID<0 should be invalid", func(t *testing.T) {
		roadmap := 0
		req := ResolveScenarioRequest{
			ProcessTypeID: 1,
			SedeID:        -1,
			Roadmap:       &roadmap,
		}

		err := validate.Struct(req)
		assert.Error(t, err)
	})
}

func TestRunBulkProcessRequest_ResolvedBulkJobID(t *testing.T) {
	t.Run("uses bulk_job_id when present", func(t *testing.T) {
		req := RunBulkProcessRequest{
			BulkJobID: 25,
			ID:        9,
		}

		assert.Equal(t, int64(25), req.ResolvedBulkJobID())
	})

	t.Run("falls back to id alias", func(t *testing.T) {
		req := RunBulkProcessRequest{
			ID: 9,
		}

		assert.Equal(t, int64(9), req.ResolvedBulkJobID())
	})
}

func TestCancelProcessRunRequest_ResolvedBulkJobID(t *testing.T) {
	t.Run("uses bulk_job_id when present", func(t *testing.T) {
		req := CancelProcessRunRequest{
			BulkJobID: 30,
			ID:        12,
		}

		assert.Equal(t, int64(30), req.ResolvedBulkJobID())
	})

	t.Run("falls back to id alias", func(t *testing.T) {
		req := CancelProcessRunRequest{
			ID: 12,
		}

		assert.Equal(t, int64(12), req.ResolvedBulkJobID())
	})
}
