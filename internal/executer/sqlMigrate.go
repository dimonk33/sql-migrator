package executer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	migfile "github.com/dimonk33/sql-migrator/internal/file"
)

const (
	UpDirection   = 1
	DownDirection = 2
)

type SQLMigrate struct {
	db DB
}

type DB interface {
	ApplyTx(ctx context.Context, name string, sqlPool []string) error
	RevertTx(ctx context.Context, name string, sqlPool []string) error
	Create(ctx context.Context, name string) error
	Exec(ctx context.Context, sql string) error
	SetApplied(ctx context.Context, name string) error
	Delete(ctx context.Context, name string) error
}

var (
	ErrWrongDirection  = errors.New("неизвестное направление миграции")
	ErrWrongFileFormat = errors.New("неверный формат файла")
	ErrNoData          = errors.New("запросы не найдены")
)

func NewSqlMigrate(db DB) *SQLMigrate {
	return &SQLMigrate{
		db: db,
	}
}

func (sm *SQLMigrate) UpExec(ctx context.Context, path string) error {
	text, err := sm.parseFile(path, UpDirection)
	if err != nil {
		return fmt.Errorf("ошибка парсинга файла: %w", err)
	}

	sqls := sm.extractSqlRequest(text)
	if len(sqls) == 0 {
		return ErrNoData
	}

	name := filepath.Base(path)
	err = sm.db.ApplyTx(ctx, name, sqls)
	if err != nil {
		return fmt.Errorf("ошибка применения миграций из файла %s: %w", path, err)
	}

	return nil
}

//nolint:all
func (sm *SQLMigrate) DownExec(ctx context.Context, path string) error {
	return nil
}

func (sm *SQLMigrate) parseFile(path string, dir int) (string, error) {
	fileContent, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("open file error: %w", err)
	}

	upStartIndex := strings.Index(string(fileContent), migfile.SQLUpPartID) + len(migfile.SQLUpPartID)
	upEndIndex := strings.Index(string(fileContent), migfile.SQLDownPartID)
	downStartIndex := upEndIndex + len(migfile.SQLDownPartID)
	downEndIndex := len(fileContent) - 1

	if upStartIndex < len(migfile.SQLUpPartID) || upEndIndex < upStartIndex {
		return "", ErrWrongFileFormat
	}

	switch dir {
	case UpDirection:
		return string(fileContent)[upStartIndex:upEndIndex], nil
	case DownDirection:
		return string(fileContent)[downStartIndex:downEndIndex], nil
	}

	return "", ErrWrongDirection
}

func (sm *SQLMigrate) extractSqlRequest(text string) []string {
	t1 := strings.Trim(strings.ReplaceAll(strings.ReplaceAll(
		text, "\r", "",
	), "\n", ""),
		" \t\r\n")
	sql := strings.Split(t1, ";")
	return sql
}

func (sm *SQLMigrate) execSqlRequest(sql []string) error {
	return nil
}
