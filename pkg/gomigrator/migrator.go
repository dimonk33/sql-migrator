package gomigrator

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	migdb "github.com/dimonk33/sql-migrator/internal/db"
	"github.com/dimonk33/sql-migrator/internal/executer"
	migfile "github.com/dimonk33/sql-migrator/internal/file"
)

const (
	SQLMigration MigrateType = "sql"
	GoMigration  MigrateType = "go"

	opTimeout = 60 * time.Minute
)

type MigrateType = string

func Validate(mt MigrateType) error {
	if mt != SQLMigration && mt != GoMigration {
		return errors.New("неизвестный тип миграции")
	}

	return nil
}

type Migrator struct {
	logger  Logger
	dirPath string
	db      DB
	finder  *migfile.Finder
}

type DBConnParam = migdb.ConnParam

type MigrateStatus = migdb.MigrateInfo

type Logger interface {
	Info(v ...any)
	Error(v ...any)
	Warning(v ...any)
	Debug(v ...any)
}

type DB interface {
	executer.DBSQL
	executer.DBGo
	Lock(ctx context.Context, sign string) bool
	Unlock(ctx context.Context, sign string) bool
	Find(ctx context.Context, name string) (int, error)
	FindLast(ctx context.Context) (string, error)
	FindAllApplied(ctx context.Context) ([]migdb.MigrateInfo, error)
}

type MigrateExec interface {
	UpExec(ctx context.Context, path string) error
	DownExec(ctx context.Context, path string) error
}

var ErrNoMigrations = errors.New("отсутствуют миграции для применения")

func New(l Logger, dir string, dbConn *DBConnParam) (*Migrator, error) {
	m := &Migrator{
		logger:  l,
		dirPath: dir,
	}

	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()

	db, err := migdb.NewPgMigrator(ctx, dbConn, l)
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

func (m *Migrator) Status() ([]MigrateStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()

	list, err := m.db.FindAllApplied(ctx)
	if err != nil {
		return nil, err
	}

	return list, nil
}

func (m *Migrator) Version() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()
	v, err := m.db.FindLast(ctx)
	if err != nil {
		return "", err
	}

	return v, nil
}

func (m *Migrator) Create(migrateName string, migrateType MigrateType) (string, error) {
	if err := Validate(migrateType); err != nil {
		return "", err
	}

	if migrateType != migfile.SQLFile && migrateType != migfile.GoFile {
		return "", errors.New("неверный тип миграции: " + migrateType)
	}

	err := os.MkdirAll(m.dirPath, 0o750)
	if err != nil {
		return "", fmt.Errorf("ошибка создания каталога для миграций: %w", err)
	}

	t := migfile.NewTemplate(m.logger, m.dirPath)

	var fname string
	if fname, err = t.Create(migrateName, migrateType); err != nil {
		return "", fmt.Errorf("ошибка создания миграции: %w", err)
	}

	return fname, nil
}

func (m *Migrator) Up() error {
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
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

	keys := make([]string, 0, len(flist))

	for key := range flist {
		keys = append(keys, key)
	}
	sort.SliceStable(keys, func(i, j int) bool {
		return flist[keys[i]] < flist[keys[j]]
	})

	var mExecuter MigrateExec
	for _, k := range keys {
		f := flist[k]
		m.logger.Info("Применение миграции", k)

		dbSign := filepath.Base(f)
		if !m.db.Lock(ctx, dbSign) {
			m.logger.Warning("Миграция", dbSign, "заблокирована")
			continue
		}

		ext := strings.Trim(filepath.Ext(f), ".")

		switch ext {
		case migfile.SQLFile:
			mExecuter = executer.NewSQLMigrate(m.db)
		case migfile.GoFile:
			mExecuter = executer.NewGoMigrate(m.db, m.logger)
		}

		err = mExecuter.UpExec(ctx, f)
		if err != nil {
			if !m.db.Unlock(ctx, dbSign) {
				m.logger.Error("ошибка разблокировки миграции ", dbSign)
			}

			return fmt.Errorf("ошибка применения миграции %s: %w", f, err)
		}

		if !m.db.Unlock(ctx, dbSign) {
			m.logger.Error("ошибка разблокировки миграции ", dbSign)
		}

		m.logger.Info("Миграция", f, "применена")
	}

	return nil
}

func (m *Migrator) Down() error {
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()

	lastMigrationName, lastMigrationPath, err := m.getLastMigration(ctx)
	if err != nil {
		return err
	}

	if !m.db.Lock(ctx, lastMigrationName) {
		return fmt.Errorf("миграция %s заблокирована", lastMigrationName)
	}

	defer func() {
		if !m.db.Unlock(ctx, lastMigrationName) {
			m.logger.Error("ошибка разблокировки миграции ", lastMigrationName)
		}
	}()

	var mExecuter MigrateExec

	m.logger.Info("Откат миграции", lastMigrationName)
	ext := strings.Trim(filepath.Ext(lastMigrationName), ".")

	switch ext {
	case migfile.SQLFile:
		mExecuter = executer.NewSQLMigrate(m.db)
	case migfile.GoFile:
		mExecuter = executer.NewGoMigrate(m.db, m.logger)
	}

	err = mExecuter.DownExec(ctx, lastMigrationPath)
	if err != nil {
		return fmt.Errorf("ошибка отмены миграции %s: %w", lastMigrationName, err)
	}

	m.logger.Info("Миграция", lastMigrationName, "отменена")

	return nil
}

func (m *Migrator) Redo() error {
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()

	lastMigrationName, lastMigrationPath, err := m.getLastMigration(ctx)
	if err != nil {
		return err
	}

	if !m.db.Lock(ctx, lastMigrationName) {
		return fmt.Errorf("миграция %s заблокирована", lastMigrationName)
	}

	defer func() {
		if !m.db.Unlock(ctx, lastMigrationName) {
			m.logger.Error("ошибка разблокировки миграции ", lastMigrationName)
		}
	}()

	var mExecuter MigrateExec

	m.logger.Info("Откат миграции", lastMigrationName)
	ext := strings.Trim(filepath.Ext(lastMigrationName), ".")

	switch ext {
	case migfile.SQLFile:
		mExecuter = executer.NewSQLMigrate(m.db)
	case migfile.GoFile:
		mExecuter = executer.NewGoMigrate(m.db, m.logger)
	}

	err = mExecuter.DownExec(ctx, lastMigrationPath)
	if err != nil {
		return fmt.Errorf("ошибка отмены миграции %s: %w", lastMigrationName, err)
	}

	err = mExecuter.UpExec(ctx, lastMigrationPath)
	if err != nil {
		return fmt.Errorf("ошибка применения миграции %s: %w", lastMigrationName, err)
	}

	m.logger.Info("Миграция", lastMigrationName, "применена повторно")

	return nil
}

func (m *Migrator) getLastMigration(ctx context.Context) (string, string, error) {
	flist, err := m.finder.ScanDir(ctx, m.dirPath)
	if err != nil {
		return "", "", fmt.Errorf("ошибка поиска миграций в каталоге: %w", err)
	}
	m.logger.Info("Cписок миграций:\n", flist)

	var lastMigrationName string

	if lastMigrationName, err = m.db.FindLast(ctx); err != nil {
		return "", "", fmt.Errorf("ошибка получения последней миграции из базы: %w", err)
	}

	path, ok := flist[lastMigrationName]
	if !ok {
		return "", "", fmt.Errorf("миграция %s отсутствует на диске", lastMigrationName)
	}

	return lastMigrationName, path, nil
}
