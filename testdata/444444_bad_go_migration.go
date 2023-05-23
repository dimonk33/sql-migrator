package main

import (
	"database/sql"
)

const (
	tableName = "test_go_migration"
)

func up(tx *sql.Tx) error {
	_, err := tx.Exec(`
	CREATE TABLE
	    ` + tableName + `(
			id integer,
			name varchar(255)
		)
	`)
	if err != nil {
		return err
	}

	return tx.Commit()
}
