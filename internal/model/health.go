// internal/model/health.go
package model

import (
	"database/sql"
	"time"
)

type HealthCheck struct {
	CheckID  int64     `json:"check_id"`
	DateTime time.Time `json:"datetime"`
}

func InsertHealthCheck(db *sql.DB) error {
	query := "INSERT INTO health_check (datetime) VALUES (UTC_TIMESTAMP())"
	_, err := db.Exec(query)
	return err
}
