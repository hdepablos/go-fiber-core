package requests

type BulkAssignMenuUsersRequest struct {
	MenuIDs []uint64 `json:"menu_ids"`
	UserIDs []uint64 `json:"user_ids"`
}
