package requests

// CreateImportRequest define el contrato esperado para la creación de un import.
// En HTTP se recibe como multipart/form-data junto al archivo `file`.
type CreateImportRequest struct {
	BranchID int64  `json:"branch_id" form:"branch_id"`
	RefCode  string `json:"ref_code" form:"ref_code"`
	Total    int    `json:"total" form:"total"`
	KeyCode  string `json:"key_code" form:"key_code"`
}
