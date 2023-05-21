package executer

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	migfile "github.com/dimonk33/sql-migrator/internal/file"
)

const (
	prefixErrMigrateDelete = "Удаление записи о миграции "
)

type GoMigrate struct {
	db     DBGo
	logger migfile.Logger
}

type DBGo interface {
	GetConnString() string
	Create(ctx context.Context, name string) error
	Exec(ctx context.Context, sql string) error
	SetApplied(ctx context.Context, name string) error
	Delete(ctx context.Context, name string) error
}

func NewGoMigrate(db DBGo, l migfile.Logger) *GoMigrate {
	return &GoMigrate{
		db:     db,
		logger: l,
	}
}

func (sm *GoMigrate) exec(mFuncName string, mpath string) error {
	var (
		mFuncContent string
		mDirPath     string
		err          error
	)

	mFuncContent, err = sm.parseFile(mpath)
	if err != nil {
		return fmt.Errorf("ошибка парсинга файла: %w", err)
	}

	mDirPath, err = os.MkdirTemp("", strings.TrimRight(filepath.Base(mpath), "."+migfile.GoFile))
	if err != nil {
		return fmt.Errorf("ошибка создания каталога: %w", err)
	}

	_, err = sm.genMainFile(mDirPath, mFuncContent, mFuncName)
	if err != nil {
		return fmt.Errorf("ошибка создания main файла: %w", err)
	}

	err = sm.execMigration(mDirPath)

	sm.logger.Info("удаление каталога миграции:", os.RemoveAll(mDirPath))

	if err != nil {
		return fmt.Errorf("запуск миграции: %w", err)
	}

	return nil
}

func (sm *GoMigrate) UpExec(ctx context.Context, mpath string) error {
	mName := filepath.Base(mpath)

	if err := sm.db.Create(ctx, mName); err != nil {
		return fmt.Errorf("регистрация миграции: %w", err)
	}

	if err := sm.exec(migfile.GoUpFuncName, mpath); err != nil {
		sm.logger.Warning(prefixErrMigrateDelete, sm.db.Delete(ctx, mName))
		return fmt.Errorf("применение миграции: %w", err)
	}

	if err := sm.db.SetApplied(ctx, mName); err != nil {
		return fmt.Errorf("закрытие миграции: %w", err)
	}

	return nil
}

func (sm *GoMigrate) DownExec(ctx context.Context, mpath string) error {
	mName := filepath.Base(mpath)

	if err := sm.exec(migfile.GoDownFuncName, mpath); err != nil {
		return fmt.Errorf("откат миграции: %w", err)
	}

	if err := sm.db.Delete(ctx, mName); err != nil {
		return fmt.Errorf("удаление записи о миграции: %w", err)
	}

	return nil
}

func (sm *GoMigrate) parseFile(path string) (string, error) {
	fileContent, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("ошибка открытия файла: %w", err)
	}

	fileStr := string(fileContent)

	upStartIndex := strings.Index(fileStr, migfile.GoUpPartID)
	downStartIndex := strings.Index(fileStr, migfile.GoDownPartID)

	if upStartIndex == -1 || downStartIndex == -1 {
		return "", ErrWrongFileFormat
	}

	return fileStr, nil
}

func (sm *GoMigrate) genMainFile(
	dirPath string,
	migrateContent string,
	callFuncName string,
) (string, error) {
	const prefixErrMsg = "генерация main файла"

	var (
		mainFilePath string
		err          error
	)

	t := migfile.NewTemplate(sm.logger, dirPath)
	mainFilePath, err = t.CreateGoMain(migrateContent, callFuncName, sm.db.GetConnString())
	if err != nil {
		return "", fmt.Errorf("%s: %w", prefixErrMsg, err)
	}

	return mainFilePath, nil
}

func (sm *GoMigrate) execMigration(srcDirPath string) error {
	t := migfile.NewTemplate(sm.logger, srcDirPath)
	runFilePath, err := t.CreateRunSh(srcDirPath)
	if err != nil {
		return fmt.Errorf("генерация run файла: %w", err)
	}
	cmdOutput := &bytes.Buffer{}
	cmd := exec.Command(runFilePath)
	cmd.Stdout = cmdOutput
	err = cmd.Run()
	if err != nil {
		_, errO := os.Stderr.WriteString(err.Error())
		if errO != nil {
			sm.logger.Warning(errO)
		}
		sm.logger.Error(err)
		return fmt.Errorf("сборка миграции: %w", err)
	}
	fmt.Print(cmdOutput.String())

	return nil
}
