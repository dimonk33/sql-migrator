package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

const (
	serviceTableName = "gomigrate_info"
	enumTableName    = "gomigrate_enum"
	statusProcessing = "processing"
	statusApplied    = "applied"
)

type PgMigrator struct {
	conn *sqlx.DB
}

type ConnParam struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
	SSL      string
}

func NewPgMigrator(ctx context.Context, dbConn *ConnParam) (*PgMigrator, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password='%s' sslmode=%s",
		dbConn.Host,
		dbConn.Port,
		dbConn.Name,
		dbConn.User,
		dbConn.Password,
		dbConn.SSL,
	)
	c, err := sqlx.ConnectContext(ctx, "postgres", connStr)
	if err != nil {
		return nil, err
	}
	m := &PgMigrator{
		conn: c,
	}
	err = m.initTable(ctx)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (m *PgMigrator) initTable(ctx context.Context) error {
	var (
		tx  *sql.Tx
		err error
	)

	tx, err = m.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx,
		`DO $$
			BEGIN
				IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = '`+enumTableName+`') THEN
					CREATE TYPE `+enumTableName+` AS ENUM('processing', 'applied');
				END IF;
			END
			$$;`)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx,
		`CREATE TABLE IF NOT EXISTS `+serviceTableName+`(
			    id SERIAL PRIMARY KEY,
				name varchar(255) NOT NULL,
				status `+enumTableName+` NOT NULL,
				created_at timestamp NOT NULL default now(),
				updated_at timestamp NOT NULL default now()
			)`,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx,
		`CREATE UNIQUE INDEX IF NOT EXISTS name_uniq_idx ON `+serviceTableName+`(name);`,
	)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (m *PgMigrator) Lock(ctx context.Context, sign string) bool {
	var lock bool
	sqlReq := `SELECT pg_try_advisory_lock(
		('x' || md5('` + sign + `'))::bit(64)::bigint
	);`

	err := m.conn.GetContext(ctx, &lock, sqlReq)
	if err != nil {
		return false
	}
	return lock
}

func (m *PgMigrator) Unlock(ctx context.Context, sign string) bool {
	var lock bool
	sqlReq := `SELECT pg_advisory_unlock(
		('x' || md5('` + sign + `'))::bit(64)::bigint
	);`

	err := m.conn.GetContext(ctx, &lock, sqlReq)
	if err != nil {
		return false
	}
	return lock
}

func (m *PgMigrator) Apply(ctx context.Context, sqlReq string) error {
	_, err := m.conn.ExecContext(ctx, sqlReq)
	return err
}

func (m *PgMigrator) Create(ctx context.Context, name string) error {
	sqlReq := `
		INSERT INTO $1 (name, status) VALUES($2, $3)`
	_, err := m.conn.ExecContext(ctx, sqlReq, serviceTableName, name, statusProcessing)
	return err
}

func (m *PgMigrator) SetApplied(ctx context.Context, name string) error {
	sqlReq := `
		UPDATE $1 SET status = $3 WHERE name = $2`
	_, err := m.conn.ExecContext(ctx, sqlReq, serviceTableName, name, statusApplied)
	return err
}

func (m *PgMigrator) Delete(ctx context.Context, name string) error {
	sqlReq := `
		DELETE FROM $1 WHERE name = $2`
	_, err := m.conn.ExecContext(ctx, sqlReq, serviceTableName, name)
	return err
}

func (m *PgMigrator) Find(ctx context.Context, name string) (int, error) {
	sqlReq := `SELECT id FROM $1 WHERE name = $2`
	var id int
	err := m.conn.GetContext(ctx, &id, sqlReq, serviceTableName, name)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (m *PgMigrator) FindLast(ctx context.Context) (string, error) {
	sqlReq := `SELECT name FROM $1 WHERE status = 'applied' ORDER BY created_at DESC LIMIT 1`
	var name string
	err := m.conn.GetContext(ctx, &name, sqlReq, serviceTableName)
	if err != nil {
		return name, err
	}

	return name, nil
}

func (m *PgMigrator) FindAllApplied(ctx context.Context) ([][2]string, error) {
	sqlReq := `SELECT name, updated_at FROM $1 WHERE status = 'applied' ORDER BY created_at DESC`
	var data [][2]string
	err := m.conn.SelectContext(ctx, &data, sqlReq, serviceTableName)
	if err != nil {
		return data, err
	}

	return data, nil
}
