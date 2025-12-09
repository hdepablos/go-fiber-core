package dtos

import "time"

type BackofficeReversalExtra struct {
	StartProduct time.Time `json:"start_product"`
}

type Config struct {
	Config BackofficeReversal `json:"config"`
}

type BackofficeReversal struct {
	CustomerID       int                     `json:"customer_id"`
	InstallmentState []int                   `json:"installment_state"`
	Extra            BackofficeReversalExtra `json:"extra"`
	Imputation       bool                    `json:"imputation"`
}
