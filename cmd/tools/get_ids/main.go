package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
)

func main() {
	connStr := "postgres://gorm_writer:gorm_write_secret@localhost:5432/go_fiber_core?sslmode=disable"
	conn, err := pgx.Connect(context.Background(), connStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	var processTypeID int64
	err = conn.QueryRow(context.Background(), "SELECT id FROM process_types WHERE name = $1", "Test Auto Invoke Process").Scan(&processTypeID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "QueryRow failed: %v\n", err)
		os.Exit(1)
	}

	var versionID int64
	err = conn.QueryRow(context.Background(), "SELECT id FROM process_versions WHERE process_type_id = $1 AND version_number = 1", processTypeID).Scan(&versionID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "QueryRow failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("ProcessTypeID: %d\n", processTypeID)
	fmt.Printf("ProcessVersionID: %d\n", versionID)
}
