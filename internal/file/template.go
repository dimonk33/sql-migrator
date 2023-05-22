package migfile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
	"time"
)

const (
	SQLUpPartID   = "-- ===gm Up==="
	SQLDownPartID = "-- ===gm Down==="

	GoUpFuncName   = "up"
	GoDownFuncName = "down"

	GoUpPartID   = "func " + GoUpFuncName + "(tx *sql.Tx) error {"
	GoDownPartID = "func " + GoDownFuncName + "(tx *sql.Tx) error {"
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
	MainFunc string
}

type goTmplVars struct {
	MigrateCode string
	MigrateFunc string
	DBConn      string
}

type shTmplVars struct {
	SrcDir string
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

` + GoUpPartID + `
	// Здесь располагается код, который будет выполнен при применении миграции
	return nil
}

` + GoDownPartID + `
	// Здесь располагается код, который будет выполнен при откате миграции
	return nil
}
`))

var goMainTemplate = template.Must(template.New("gm.go-main").Parse(
	`package main

import (
	"log"
	"context"
	"time"
	_ "github.com/lib/pq"
{{.MigrateCode}}

func main() {
	var (
		db	*sql.DB
		tx	*sql.Tx
		err error
	)
	db, err = sql.Open("postgres", "{{.DBConn}}")
	if err != nil {
		log.Fatal(err)
	}
	db.SetConnMaxLifetime(0)
	db.SetMaxIdleConns(1)
	db.SetMaxOpenConns(1)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tx, err = db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		log.Fatal(err)
	}

	err = {{.MigrateFunc}}(tx)
	if err != nil {
		log.Fatal(err)
	}
}
`))

var shRunTemplate = template.Must(template.New("gm.go-main").Parse(
	`#!/bin/bash

cd {{.SrcDir}} &&
go mod init migrate &&
go mod tidy
go run .
`))

var batRunTemplate = template.Must(template.New("gm.go-main").Parse(
	`cd /d {{.SrcDir}}
go mod init migrate
go mod tidy
go run .
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

	t.f, err = os.Create(path)
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
		MainFunc: "",
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

func (t *Template) CreateGoMain(content string, callFuncName string, dbConn string) (string, error) {
	const importPrefix = "import ("
	_, err := os.Stat(t.tmplDirPath)
	if err != nil {
		return "", fmt.Errorf("ошибка наличия каталога: %w", err)
	}

	importPos := strings.Index(content, importPrefix)
	if importPos == -1 {
		return "", fmt.Errorf("неверный формат")
	}

	validContent := strings.TrimLeft(content[importPos+len(importPrefix):], "\r\n")

	fname := filepath.Join(t.tmplDirPath, "main.go")

	t.f, err = os.Create(fname)
	if err != nil {
		return "", fmt.Errorf("ошибка создания файла: %w", err)
	}

	defer func() {
		err = t.f.Close()
		if err != nil {
			t.logger.Warning("ошибка закрытия файла: " + err.Error())
		}
	}()

	tv := goTmplVars{
		MigrateCode: validContent,
		MigrateFunc: callFuncName,
		DBConn:      dbConn,
	}

	err = goMainTemplate.Execute(t.f, tv)

	if err != nil {
		return "", fmt.Errorf("ошибка генерации шаблона: %w", err)
	}

	return fname, nil
}

func (t *Template) CreateRunSh() (string, error) {
	_, err := os.Stat(t.tmplDirPath)
	if err != nil {
		return "", fmt.Errorf("ошибка наличия каталога: %w", err)
	}

	var fname string
	var tmpl *template.Template
	switch runtime.GOOS {
	case "linux":
		fname = filepath.Join(t.tmplDirPath, "run.sh")
		tmpl = shRunTemplate
	case "windows":
		fname = filepath.Join(t.tmplDirPath, "run.bat")
		tmpl = batRunTemplate
	default:
		return "", fmt.Errorf("неподдерживаемая система: %s", runtime.GOOS)
	}

	t.f, err = os.Create(fname)
	if err != nil {
		return "", fmt.Errorf("ошибка создания файла: %w", err)
	}

	defer func() {
		err = t.f.Close()
		if err != nil {
			t.logger.Warning("ошибка закрытия файла: " + err.Error())
		}
	}()

	tv := shTmplVars{
		SrcDir: t.tmplDirPath,
	}

	err = tmpl.Execute(t.f, tv)

	if err != nil {
		return "", fmt.Errorf("ошибка генерации шаблона: %w", err)
	}

	return fname, nil
}
