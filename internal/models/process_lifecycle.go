package models

import "time"

type ProcessVersionRow struct {
	ID                   uint64
	ProcessTypeName      string
	ProcessTypeIsVisible bool
	VersionNumber        int
	SedeID               *uint64
	Status               string
	OperatorEmail        *string
	ValidFrom            *time.Time
	ValidTo              *time.Time
}

type ProcessVersionListItem struct {
	ID                   uint64     `json:"id"`
	ProcessTypeName      string     `json:"process_type_name"`
	ProcessTypeIsVisible bool       `json:"process_type_is_visible"`
	VersionNumber        int        `json:"version_number"`
	SedeID               *uint64    `json:"sede_id"`
	Status               string     `json:"status"`
	OperatorEmail        *string    `json:"operator_email"`
	ValidFrom            *time.Time `json:"valid_from"`
	ValidTo              *time.Time `json:"valid_to"`
}

