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
	db DBSQL
}

type DBSQL interface {
	ApplyTx(ctx context.Context, name string, sqlPool []string) error
	RevertTx(ctx context.Context, name string, sqlPool []string) error
}

var (
	ErrWrongDirection  = errors.New("неизвестное направление миграции")
	ErrWrongFileFormat = errors.New("неверный формат файла")
	ErrNoData          = errors.New("запросы не найдены")
)

func NewSQLMigrate(db DBSQL) *SQLMigrate {
	return &SQLMigrate{
		db: db,
	}
}

func (sm *SQLMigrate) UpExec(ctx context.Context, path string) error {
	text, err := sm.parseFile(path, UpDirection)
	if err != nil {
		return fmt.Errorf("ошибка парсинга файла: %w", err)
	}

	sqls := sm.extractSQLRequest(text)
	if len(sqls) == 0 {
		return ErrNoData
	}

	name := filepath.Base(path)
	err = sm.db.ApplyTx(ctx, name, sqls)
	if err != nil {
		return fmt.Errorf("ошибка применения миграции %s: %w", path, err)
	}

	return nil
}

func (sm *SQLMigrate) DownExec(ctx context.Context, path string) error {
	text, err := sm.parseFile(path, DownDirection)
	if err != nil {
		return fmt.Errorf("ошибка парсинга файла: %w", err)
	}

	sqlList := sm.extractSQLRequest(text)
	if len(sqlList) == 0 {
		return ErrNoData
	}

	name := filepath.Base(path)
	err = sm.db.RevertTx(ctx, name, sqlList)
	if err != nil {
		return fmt.Errorf("ошибка отката миграции %s: %w", path, err)
	}

	return nil
}

func (sm *SQLMigrate) parseFile(path string, dir int) (string, error) {
	fileContent, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("ошибка открытия файла: %w", err)
	}

	fileStr := string(fileContent)
	upStartIndex := strings.Index(fileStr, migfile.SQLUpPartID) + len(migfile.SQLUpPartID)
	upEndIndex := strings.Index(fileStr, migfile.SQLDownPartID)
	downStartIndex := upEndIndex + len(migfile.SQLDownPartID)
	downEndIndex := len(fileContent) - 1

	if upStartIndex < len(migfile.SQLUpPartID) || upEndIndex < upStartIndex {
		return "", ErrWrongFileFormat
	}

	switch dir {
	case UpDirection:
		return fileStr[upStartIndex:upEndIndex], nil
	case DownDirection:
		return fileStr[downStartIndex:downEndIndex], nil
	}

	return "", ErrWrongDirection
}

func (sm *SQLMigrate) extractSQLRequest(text string) []string {
	ts := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(text, "\r", ""), "\n", ""))
	sql := strings.Split(ts, ";")
	out := make([]string, 0)
	for _, s := range sql {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
