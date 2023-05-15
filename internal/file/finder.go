package file

import (
	"context"
	"errors"
	"os"
	"path/filepath"
)

const (
	SqlFile = "sql"
	GoFile  = "go"
)

type Finder struct {
	fileType string
}

func NewFileFinder(ft string) (*Finder, error) {
	if ft != SqlFile && ft != GoFile {
		return nil, errors.New("неподдерживаемый тип файлов")
	}
	return &Finder{
		fileType: ft,
	}, nil
}

func (ff *Finder) scanDir(ctx context.Context, path string) ([]string, error) {
	var list []string

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			if ff.validateEntry(e) {
				list = append(list, path+"/"+e.Name())
			}
		}
	}

	return list, nil
}

func (ff *Finder) validateEntry(e os.DirEntry) bool {
	if e.IsDir() {
		return false
	}
	ext := filepath.Ext(e.Name())
	if ext != ff.fileType {
		return false
	}
	return true
}
