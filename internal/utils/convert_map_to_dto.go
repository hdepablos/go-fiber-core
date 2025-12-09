package utils

import (
	"encoding/json"
)

func ConvertMapToDTO(mapa map[string]any, result any) error {
	data, err := json.Marshal(mapa)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, result)
	if err != nil {
		return err
	}

	return nil
}
