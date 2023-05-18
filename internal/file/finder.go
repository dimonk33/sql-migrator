package migfile

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

const (
	SQLFile = "sql"
	GoFile  = "go"
)

type Finder struct{}

func NewFileFinder() (*Finder, error) {
	return &Finder{}, nil
}

func (ff *Finder) ScanDir(ctx context.Context, path string) (map[string]string, error) {
	list := make(map[string]string)

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
				list[e.Name()] = filepath.Join(path, e.Name())
			}
		}
	}

	return list, nil
}

func (ff *Finder) validateEntry(e os.DirEntry) bool {
	if e.IsDir() {
		return false
	}
	ext := strings.ReplaceAll(filepath.Ext(e.Name()), ".", "")
	return ext == SQLFile || ext == GoFile
}
