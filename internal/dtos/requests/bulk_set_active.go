package requests

type BulkSetActiveDTO struct {
	IDs    []uint64 `json:"ids"`
	Active bool     `json:"active"`
}
