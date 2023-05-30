package migdb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // Инициализация драйвера Postgresql
)

const (
	serviceTableName = "gomigrate_info"
	enumTableName    = "gomigrate_enum"
	statusProcessing = "processing"
	statusApplied    = "applied"

	logPrefixApplyMigration = "применение миграции:"
)

type Pg struct {
	connStr string
	conn    *sqlx.DB
	logger  Logger
}

type Logger interface {
	Error(v ...any)
	Warning(v ...any)
}

type ConnParam struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
	SSL      string
}

type MigrateInfo struct {
	Name      string    `db:"name"`
	UpdatedAt time.Time `db:"updated_at"`
}

func NewPgMigrator(ctx context.Context, dbConn *ConnParam, l Logger) (*Pg, error) {
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
	b := &Pg{
		conn:    c,
		logger:  l,
		connStr: connStr,
	}
	err = b.initTable(ctx)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (b *Pg) initTable(ctx context.Context) error {
	const logInitPrefix = "создание служебных таблиц: "

	var (
		tx  *sql.Tx
		err error
	)

	tx, err = b.conn.BeginTx(ctx, nil)
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
		b.txRollback(tx, logInitPrefix)
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
		b.txRollback(tx, logInitPrefix)
		return err
	}

	_, err = tx.ExecContext(ctx,
		"CREATE UNIQUE INDEX IF NOT EXISTS name_uniq_idx ON "+serviceTableName+"(name);",
	)
	if err != nil {
		b.txRollback(tx, logInitPrefix)
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (b *Pg) GetConnString() string {
	return b.connStr
}

func (b *Pg) Lock(ctx context.Context, sign string) bool {
	var lock bool
	sqlReq := `SELECT pg_try_advisory_lock(
		('x' || md5('` + sign + `'))::bit(64)::bigint
	);`

	err := b.conn.GetContext(ctx, &lock, sqlReq)
	if err != nil {
		b.logger.Error("блокировка базы:", err)
		b.Unlock(ctx, sign)
		return false
	}
	return lock
}

func (b *Pg) Unlock(ctx context.Context, sign string) bool {
	var lock bool
	sqlReq := `SELECT pg_advisory_unlock(
		('x' || md5('` + sign + `'))::bit(64)::bigint
	);`

	err := b.conn.GetContext(ctx, &lock, sqlReq)
	if err != nil {
		b.logger.Error("разблокировка базы:", err)
		return false
	}
	return lock
}

func (b *Pg) Find(ctx context.Context, name string) (int, error) {
	sqlReq := "SELECT id FROM " + serviceTableName + " WHERE name = $1"
	var id int
	err := b.conn.GetContext(ctx, &id, sqlReq, name)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (b *Pg) FindLast(ctx context.Context) (string, error) {
	sqlReq := "SELECT name FROM " + serviceTableName + " WHERE status = 'applied' ORDER BY created_at DESC LIMIT 1"
	var name string
	err := b.conn.GetContext(ctx, &name, sqlReq)
	if err != nil {
		return name, err
	}

	return name, nil
}

func (b *Pg) FindAllApplied(ctx context.Context) ([]MigrateInfo, error) {
	sqlReq := "SELECT name, updated_at FROM " + serviceTableName + " WHERE status = 'applied' ORDER BY created_at DESC"
	data := make([]MigrateInfo, 1)
	err := b.conn.SelectContext(ctx, &data, sqlReq)
	if err != nil {
		return data, err
	}

	return data, nil
}

func (b *Pg) ApplyTx(ctx context.Context, name string, sqlPool []string) error {
	if err := b.Create(ctx, name); err != nil {
		return fmt.Errorf("создание записи в базе: %w", err)
	}

	tx, err := b.conn.BeginTx(ctx, nil)
	if err != nil {
		b.txRollback(tx, logPrefixApplyMigration)
		b.deleteMigrate(ctx, name)
		return err
	}

	for i, s := range sqlPool {
		_, err = tx.ExecContext(ctx, s)
		if err != nil {
			b.txRollback(tx, logPrefixApplyMigration)
			b.deleteMigrate(ctx, name)
			return fmt.Errorf("выполнение запроса %d: %w", i, err)
		}
	}

	s := "UPDATE " + serviceTableName + " SET status = $2 WHERE name = $1"
	_, err = tx.ExecContext(ctx, s, name, statusApplied)
	if err != nil {
		b.txRollback(tx, logPrefixApplyMigration)
		b.deleteMigrate(ctx, name)
		return fmt.Errorf("изменение статуса миграции: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		b.txRollback(tx, logPrefixApplyMigration)
		b.deleteMigrate(ctx, name)
		return fmt.Errorf("ошибка закрытия транзакции: %w", err)
	}

	return nil
}

func (b *Pg) RevertTx(ctx context.Context, name string, sqlPool []string) error {
	const logPrefixRevertMigration = "откат миграции:"

	tx, err := b.conn.BeginTx(ctx, nil)
	if err != nil {
		b.txRollback(tx, logPrefixRevertMigration)
		return err
	}

	for i, s := range sqlPool {
		_, err = tx.ExecContext(ctx, s)
		if err != nil {
			b.txRollback(tx, logPrefixRevertMigration)
			return fmt.Errorf("выполнение запроса %d: %w", i, err)
		}
	}

	s := "DELETE FROM " + serviceTableName + " WHERE name = $1"
	_, err = tx.ExecContext(ctx, s, name)
	if err != nil {
		b.txRollback(tx, logPrefixRevertMigration)
		return fmt.Errorf("удаление миграции: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		b.txRollback(tx, logPrefixRevertMigration)
		return fmt.Errorf("ошибка закрытия транзакции: %w", err)
	}

	return nil
}

func (b *Pg) Create(ctx context.Context, name string) error {
	sqlReq := "INSERT INTO " + serviceTableName + " (name, status) VALUES($1, $2)"
	_, err := b.conn.ExecContext(ctx, sqlReq, name, statusProcessing)
	return err
}

func (b *Pg) Exec(ctx context.Context, sqlReq string) error {
	_, err := b.conn.ExecContext(ctx, sqlReq)
	return err
}

func (b *Pg) SetApplied(ctx context.Context, name string) error {
	sqlReq := "UPDATE " + serviceTableName + " SET status = $2 WHERE name = $1"
	_, err := b.conn.ExecContext(ctx, sqlReq, name, statusApplied)
	return err
}

func (b *Pg) Delete(ctx context.Context, name string) error {
	sqlReq := "DELETE FROM " + serviceTableName + " WHERE name = $1"
	_, err := b.conn.ExecContext(ctx, sqlReq, name)
	return err
}

func (b *Pg) txRollback(tx *sql.Tx, logPrefix string) {
	if err := tx.Rollback(); err != nil {
		b.logger.Error(logPrefix, err)
	}
}

func (b *Pg) deleteMigrate(ctx context.Context, name string) {
	if err := b.Delete(ctx, name); err != nil {
		b.logger.Error(logPrefixApplyMigration, err)
	}
}
