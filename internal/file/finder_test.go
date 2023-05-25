package migfile

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testDataPath = "../../tests/testdata"
)

func TestFinder_ScanDir(t *testing.T) {
	type args struct {
		ctx  context.Context
		path string
	}

	var (
		err     error
		srcPath string
	)

	srcPath, err = os.Getwd()
	require.NoError(t, err)

	absTestDataPath := filepath.Join(srcPath, testDataPath)

	tests := []struct {
		name    string
		args    args
		want    map[string]string
		wantErr bool
	}{
		{
			name: "test ok",
			args: args{
				ctx:  context.Background(),
				path: absTestDataPath,
			},
			want: map[string]string{
				"111111_sql_migration.sql":     filepath.Join(absTestDataPath, "111111_sql_migration.sql"),
				"222222_go_migration.go":       filepath.Join(absTestDataPath, "222222_go_migration.go"),
				"444444_bad_go_migration.go":   filepath.Join(absTestDataPath, "444444_bad_go_migration.go"),
				"555555_bad_sql_migration.sql": filepath.Join(absTestDataPath, "555555_bad_sql_migration.sql"),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ff := &Finder{}
			got, err := ff.ScanDir(tt.args.ctx, tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ScanDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ScanDir() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFinder_validateEntry(t *testing.T) {
	type args struct {
		e os.DirEntry
	}

	var (
		err     error
		srcPath string
		dir     []os.DirEntry
	)

	srcPath, err = os.Getwd()
	require.NoError(t, err)

	dir, err = os.ReadDir(filepath.Join(srcPath, testDataPath))
	require.NoError(t, err)
	require.Equal(t, 5, len(dir))

	var testFileID [3]int
	for i, e := range dir {
		switch filepath.Ext(e.Name()) {
		case ".sql":
			testFileID[0] = i
		case ".go":
			testFileID[1] = i
		default:
			testFileID[2] = i
		}
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "test sql",
			args: args{
				e: dir[testFileID[0]],
			},
			want: true,
		},
		{
			name: "test go",
			args: args{
				e: dir[testFileID[1]],
			},
			want: true,
		},
		{
			name: "test txt",
			args: args{
				e: dir[testFileID[2]],
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ff := &Finder{}
			if got := ff.validateEntry(tt.args.e); got != tt.want {
				t.Errorf("validateEntry() = %v, want %v", got, tt.want)
			}
		})
	}
}
