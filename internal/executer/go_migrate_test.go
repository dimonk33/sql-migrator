package executer

import (
	"os"
	"path/filepath"
	"testing"

	migfile "github.com/dimonk33/sql-migrator/internal/file"
	"github.com/dimonk33/sql-migrator/internal/logger"
	"github.com/stretchr/testify/require"
)

const (
	testDataPath   = "../../testdata"
	testGoodGoFile = "222222_go_migration.go"
	testBadGoFile  = "444444_bad_go_migration.go"
)

func TestGoMigrate_parseFile(t *testing.T) {
	type fields struct {
		logger migfile.Logger
	}

	ff := fields{
		logger: logger.New(logger.LevelDebug),
	}

	var (
		srcPath      string
		testGoodData []byte
		err          error
	)

	srcPath, err = os.Getwd()
	require.NoError(t, err)

	absTestDataPath := filepath.Join(srcPath, testDataPath)

	testGoodData, err = os.ReadFile(filepath.Join(absTestDataPath, testGoodGoFile))
	require.NoError(t, err)

	type args struct {
		path string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name:   "parse file ok",
			fields: ff,
			args: args{
				path: filepath.Join(absTestDataPath, testGoodGoFile),
			},
			want:    string(testGoodData),
			wantErr: false,
		},
		{
			name:   "parse file fail",
			fields: ff,
			args: args{
				path: filepath.Join(absTestDataPath, testBadGoFile),
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &GoMigrate{
				logger: tt.fields.logger,
			}

			var got string

			got, err = sm.parseFile(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseFile() got = %v, want %v", got, tt.want)
			}
		})
	}
}
