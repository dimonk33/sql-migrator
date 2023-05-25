package testdata

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

func down(tx *sql.Tx) error {
	_, err := tx.Exec("DROP TABLE " + tableName)
	if err != nil {
		return err
	}

	return tx.Commit()
}
