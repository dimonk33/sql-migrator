package gomigrator

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	db2 "github.com/dimonk33/sql-migrator/internal/db"
	"github.com/dimonk33/sql-migrator/internal/file"
)

type Migrator struct {
	logger  Logger
	dirPath string
	db      DB
	finder  *file.Finder
}

type Logger interface {
	Info(msg string)
	Error(msg string)
	Warning(msg string)
	Debug(msg string)
}

type DB interface {
	Create(ctx context.Context, name string) error
	Apply(ctx context.Context, sql string) error
	SetApplied(ctx context.Context, name string) error
	Delete(ctx context.Context, name string) error
	Find(ctx context.Context, name string) (int, error)
	FindLast(ctx context.Context) (string, error)
	FindAllApplied(ctx context.Context) ([][2]string, error)
}

func New(l Logger, dir string, dbConn *db2.ConnParam) (*Migrator, error) {
	m := &Migrator{
		logger:  l,
		dirPath: dir,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := db2.NewPgMigrator(ctx, dbConn)
	if err != nil {
		return nil, err
	}
	m.db = db

	m.finder, err = file.NewFileFinder(file.SqlFile)
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
		fmt.Printf("%s - %s\n", item[0], item[1])
	}
	return nil
}

func (m *Migrator) Create(name string, mtype string) error {
	mtype = strings.ToLower(strings.Trim(mtype, " \t\r\n"))
	if mtype != file.SqlFile && mtype != file.GoFile {
		return errors.New("неверный тип миграции: " + mtype)
	}

	err := os.MkdirAll(m.dirPath, 0750)
	if err != nil {
		return fmt.Errorf("ошибка создания каталога для миграций: %w", err)
	}

	t := file.NewTemplate(m.logger, m.dirPath)
	err = t.Create(name, mtype)
	if err != nil {
		return fmt.Errorf("ошибка создания миграции: %w", err)
	}

	return nil
}

func (m *Migrator) Up(mtype string) error {
	ff, err := file.NewFileFinder(mtype)
	if err != nil {
		return fmt.Errorf("ошибка поиска миграций: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var flist []string
	flist, err = ff.ScanDir(ctx, m.dirPath)
	if err != nil {
		return fmt.Errorf("ошибка поиска миграций: %w", err)
	}
	m.logger.Info(strings.Join(flist, "\n"))

	return nil
}

func (m *Migrator) Down(count int) error {
	return nil
}

func (m *Migrator) Redo() error {
	return nil
}
