package utils

import (
	"encoding/json"
)

func ConvertJsonNumberToInt(n json.Number) int {
	v, err := n.Int64()
	if err != nil {
		return 0
	}
	return int(v)
}
