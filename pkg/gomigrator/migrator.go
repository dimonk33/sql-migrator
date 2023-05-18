package gomigrator

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	migdb "github.com/dimonk33/sql-migrator/internal/db"
	"github.com/dimonk33/sql-migrator/internal/executer"
	migfile "github.com/dimonk33/sql-migrator/internal/file"
)

type Migrator struct {
	logger  Logger
	dirPath string
	db      DB
	finder  *migfile.Finder
}

type Logger interface {
	Info(v ...any)
	Error(v ...any)
	Warning(v ...any)
	Debug(v ...any)
}

type DB interface {
	executer.DB
	Find(ctx context.Context, name string) (int, error)
	FindLast(ctx context.Context) (string, error)
	FindAllApplied(ctx context.Context) ([]migdb.MigrateInfo, error)
}

type MigrateExec interface {
	UpExec(ctx context.Context, path string) error
	DownExec(ctx context.Context, path string) error
}

var ErrNoMigrations = errors.New("отсутствуют миграции для применения")

func New(l Logger, dir string, dbConn *migdb.ConnParam) (*Migrator, error) {
	m := &Migrator{
		logger:  l,
		dirPath: dir,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := migdb.NewPgMigrator(ctx, dbConn)
	if err != nil {
		return nil, err
	}
	m.db = db

	m.finder, err = migfile.NewFileFinder()
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (m *Migrator) Status() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	list, err := m.db.FindAllApplied(ctx)
	if err != nil {
		return err
	}
	for _, item := range list {
		fmt.Printf("%s - %s\n", item.Name, item.UpdatedAt)
	}
	return nil
}

func (m *Migrator) Create(migrateName string, migrateType string) error {
	migrateType = strings.ToLower(strings.Trim(migrateType, " \t\r\n"))
	if migrateType != migfile.SQLFile && migrateType != migfile.GoFile {
		return errors.New("неверный тип миграции: " + migrateType)
	}

	err := os.MkdirAll(m.dirPath, 0o750)
	if err != nil {
		return fmt.Errorf("ошибка создания каталога для миграций: %w", err)
	}

	t := migfile.NewTemplate(m.logger, m.dirPath)
	err = t.Create(migrateName, migrateType)
	if err != nil {
		return fmt.Errorf("ошибка создания миграции: %w", err)
	}

	return nil
}

func (m *Migrator) Up() error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	flist, err := m.finder.ScanDir(ctx, m.dirPath)
	if err != nil {
		return fmt.Errorf("ошибка поиска миграций в каталоге: %w", err)
	}
	m.logger.Info("Cписок миграций:\n", flist)

	appliedMigrations, err := m.db.FindAllApplied(ctx)
	if err != nil {
		return fmt.Errorf("ошибка получения миграций из базы: %w", err)
	}

	for _, am := range appliedMigrations {
		_, ok := flist[am.Name]
		if ok {
			delete(flist, am.Name)
		}
	}

	m.logger.Info("Cписок миграций для применения:\n", flist)

	if len(flist) == 0 {
		return ErrNoMigrations
	}

	var mExecuter MigrateExec
	for _, f := range flist {
		m.logger.Info("Применение миграции", f)
		ext := strings.Trim(filepath.Ext(f), ".")
		switch ext {
		case migfile.SQLFile:
			mExecuter = executer.NewSqlMigrate(m.db)
		case migfile.GoFile:
			mExecuter = executer.NewGoMigrate(m.db)
		}

		err = mExecuter.UpExec(ctx, f)
		if err != nil {
			m.logger.Error("Миграция", f, "ошибка:", err)
			return fmt.Errorf("ошибка применения миграции %s: %w", f, err)
		}
		m.logger.Info("Миграция", f, "применена")
	}

	return nil
}

func (m *Migrator) Down() error {
	return nil
}

func (m *Migrator) Redo() error {
	return nil
}
