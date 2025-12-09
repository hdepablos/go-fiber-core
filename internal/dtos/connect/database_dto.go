package connect

import "encoding/json"

type DataBaseDTO struct {
	DbHost                     string      `json:"db_host"`
	DbPort                     string      `json:"db_port"`
	DbUsername                 string      `json:"db_username"`
	DbPassword                 string      `json:"db_password"`
	DbDatabase                 string      `json:"db_database"`
	DbSchema                   string      `json:"db_schema"`
	DbMaxOpenConns             json.Number `json:"db_max_open_conns"`
	DbMaxIdleConns             json.Number `json:"db_max_idle_conns"`
	DbMaxConnLifeTimeInSeconds json.Number `json:"db_max_conn_life_time_in_seconds"`
}
