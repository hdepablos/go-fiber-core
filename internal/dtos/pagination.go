package dtos

// type PaginationRequest struct {
// 	SortBy          []string `json:"sortBy"`
// 	SortDesc        []bool   `json:"sortDesc"`
// 	FilterBy        []string `json:"filterBy"`
// 	FilterValues    []any    `json:"filterValues"`
// 	RowsPerPage     int      `json:"rowsPerPage"`
// 	Page            int      `json:"page"`
// 	OptimizeWithKey string   `json:"optimize_with_key,omitempty"`
// }

type PaginationRequest struct {
	SortBy          []string `json:"sortBy" mapstructure:"sortBy"`
	SortDesc        []bool   `json:"sortDesc" mapstructure:"sortDesc"`
	FilterBy        []string `json:"filterBy" mapstructure:"filterBy"`
	FilterValues    []any    `json:"filterValues" mapstructure:"filterValues"`
	RowsPerPage     int      `json:"rowsPerPage" mapstructure:"rowsPerPage"`
	Page            int      `json:"page" mapstructure:"page"`
	OptimizeWithKey string   `json:"optimize_with_key,omitempty" mapstructure:"optimize_with_key,omitempty"`
}

type PaginationResponse[T any] struct {
	Data        []T            `json:"data" mapstructure:"data"`
	TotalRows   int64          `json:"totalRows" mapstructure:"totalRows"`
	TotalPages  int            `json:"totalPages" mapstructure:"totalPages"`
	Page        int            `json:"page" mapstructure:"page"`
	RowsPerPage int            `json:"rowsPerPage" mapstructure:"rowsPerPage"`
	Extras      map[string]any `json:"extras" mapstructure:"extras"`
}
