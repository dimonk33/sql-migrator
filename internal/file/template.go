package migfile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

const (
	SQLUpPartID   = "-- ===gm Up==="
	SQLDownPartID = "-- ===gm Down==="
)

type Template struct {
	tmplDirPath string
	f           *os.File
	logger      Logger
}

type Logger interface {
	Info(v ...any)
	Error(v ...any)
	Warning(v ...any)
	Debug(v ...any)
}

type tmplVars struct {
	Name string
}

var sqlMigrateTemplate = template.Must(template.New("gm.sql-migration").Parse(
	SQLUpPartID + `
CREATE 'up SQL query';

` + SQLDownPartID + `
DROP 'down SQL query';
`))

var goMigrateTemplate = template.Must(template.New("gm.go-migration").Parse(
	`package main

import (
	"database/sql"
)

func up(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return nil
}

func down(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}
`))

func NewTemplate(logg Logger, dir string) *Template {
	return &Template{
		tmplDirPath: dir,
		logger:      logg,
	}
}

func (t *Template) Create(name string, tType string) error {
	fname := time.Now().Format("20060102150405") + "_" + name + "." + tType
	path := filepath.Join(t.tmplDirPath, fname)

	_, err := os.Stat(path)
	if !os.IsNotExist(err) {
		return fmt.Errorf("ошибка создания файла: %w", err)
	}

	t.f, err = os.Create(filepath.Join(t.tmplDirPath, fname))
	if err != nil {
		return fmt.Errorf("ошибка создания файла: %w", err)
	}

	defer func() {
		err = t.f.Close()
		if err != nil {
			t.logger.Warning("ошибка закрытия файла: " + err.Error())
		}
	}()

	tv := tmplVars{
		Name: strings.ReplaceAll(name, " ", ""),
	}

	switch tType {
	case SQLFile:
		err = sqlMigrateTemplate.Execute(t.f, tv)
	case GoFile:
		err = goMigrateTemplate.Execute(t.f, tv)
	default:
		err = errors.New("неподдерживаемый тип миграций")
	}

	if err != nil {
		return fmt.Errorf("ошибка генерации шаблона: %w", err)
	}

	return nil
}
