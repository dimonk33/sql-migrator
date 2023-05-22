package migfile

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"text/template"

	"github.com/dimonk33/sql-migrator/internal/logger"
	"github.com/stretchr/testify/require"
)

const (
	tmplDirPath = "./test"
)

func TestTemplate_Create(t1 *testing.T) {
	type fields struct {
		tmplDirPath string
		logger      Logger
	}
	type args struct {
		name  string
		tType string
	}

	require.NoError(t1, os.MkdirAll(tmplDirPath, 0o750))

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "create sql ok",
			fields: fields{
				tmplDirPath: tmplDirPath,
				logger:      logger.New(logger.LevelDebug),
			},
			args: args{
				name:  "test_sql",
				tType: SQLFile,
			},
			wantErr: false,
		},
		{
			name: "create go ok",
			fields: fields{
				tmplDirPath: tmplDirPath,
				logger:      logger.New(logger.LevelDebug),
			},
			args: args{
				name:  "test_go",
				tType: GoFile,
			},
			wantErr: false,
		},
		{
			name: "create txt fail",
			fields: fields{
				tmplDirPath: tmplDirPath,
				logger:      logger.New(logger.LevelDebug),
			},
			args: args{
				name:  "test_txt",
				tType: "txt",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := NewTemplate(tt.fields.logger, tt.fields.tmplDirPath)
			if err := t.Create(tt.args.name, tt.args.tType); (err != nil) != tt.wantErr {
				t1.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				dir, err := os.ReadDir(tt.fields.tmplDirPath)
				require.NoError(t1, err)
				var ok bool
				fileMask := tt.args.name + "." + tt.args.tType
				for _, f := range dir {
					if strings.Contains(f.Name(), fileMask) {
						ok = true
						break
					}
				}
				require.True(t1, ok)
			}
		})
	}

	require.NoError(t1, os.RemoveAll(tmplDirPath))
}

func TestTemplate_CreateGoMain(t1 *testing.T) {
	type fields struct {
		tmplDirPath string
		logger      Logger
	}

	testDirName, err := os.MkdirTemp("", "TestTemplate_CreateGoMain")
	require.NoError(t1, err)

	logg := logger.New(logger.LevelDebug)

	testContent := `import (
	"database/sql"
)

const (
	tableName = "test_go_migration"
)

func up(tx *sql.Tx) error {
	return tx.Commit()
}

func down(tx *sql.Tx) error {
	return tx.Commit()
}`
	testControlContent := `	"database/sql"
)

const (
	tableName = "test_go_migration"
)

func up(tx *sql.Tx) error {
	return tx.Commit()
}

func down(tx *sql.Tx) error {
	return tx.Commit()
}`
	type args struct {
		content      string
		callFuncName string
		dbConn       string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "test ok",
			fields: fields{
				tmplDirPath: testDirName,
				logger:      logg,
			},
			args: args{
				content:      testContent,
				callFuncName: "up",
				dbConn:       "test connection",
			},
			want:    filepath.Join(testDirName, "main.go"),
			wantErr: false,
		},
		{
			name: "test fail",
			fields: fields{
				tmplDirPath: testDirName,
				logger:      logg,
			},
			args: args{
				content:      "test content",
				callFuncName: "up",
				dbConn:       "test connection",
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Template{
				tmplDirPath: tt.fields.tmplDirPath,
				logger:      tt.fields.logger,
			}

			var got string
			got, err = t.CreateGoMain(tt.args.content, tt.args.callFuncName, tt.args.dbConn)
			if (err != nil) != tt.wantErr {
				t1.Errorf("CreateGoMain() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t1.Errorf("CreateGoMain() got = %v, want %v", got, tt.want)
				return
			}

			if tt.wantErr {
				require.Error(t1, err)
			} else {
				var fileContent []byte
				fileContent, err = os.ReadFile(got)
				require.NoError(t1, err)
				tv := goTmplVars{
					MigrateCode: testControlContent,
					MigrateFunc: tt.args.callFuncName,
					DBConn:      tt.args.dbConn,
				}

				builder := strings.Builder{}
				err = goMainTemplate.Execute(&builder, tv)
				require.NoError(t1, err)
				require.Equal(t1, builder.String(), string(fileContent))
			}
		})
	}

	require.NoError(t1, os.RemoveAll(testDirName))
}

func TestTemplate_CreateRunSh(t1 *testing.T) {
	type fields struct {
		tmplDirPath string
		logger      Logger
	}

	testDirName, err := os.MkdirTemp("", "TestTemplate_CreateGoMain")
	require.NoError(t1, err)

	logg := logger.New(logger.LevelDebug)

	type args struct{}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "test ok",
			fields: fields{
				tmplDirPath: testDirName,
				logger:      logg,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Template{
				tmplDirPath: tt.fields.tmplDirPath,
				logger:      tt.fields.logger,
			}

			var got string

			got, err = t.CreateRunSh()
			if (err != nil) != tt.wantErr {
				t1.Errorf("CreateRunSh() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got == "" {
				t1.Errorf("CreateRunSh() got = %v, want %v", got, tt.want)
				return
			}

			if tt.wantErr {
				require.Error(t1, err)
			} else {
				var fileContent []byte
				fileContent, err = os.ReadFile(got)
				require.NoError(t1, err)

				var tmpl *template.Template

				switch runtime.GOOS {
				case "linux":
					tmpl = shRunTemplate
				case "windows":
					tmpl = batRunTemplate
				}

				tv := shTmplVars{
					SrcDir: tt.fields.tmplDirPath,
				}

				builder := strings.Builder{}
				err = tmpl.Execute(&builder, tv)
				require.NoError(t1, err)
				require.Equal(t1, builder.String(), string(fileContent))
			}
		})
	}

	require.NoError(t1, os.RemoveAll(testDirName))
}
