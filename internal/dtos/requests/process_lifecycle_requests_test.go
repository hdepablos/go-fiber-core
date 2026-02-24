package requests

import (
	"testing"

	"github.com/go-playground/validator/v10"
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
