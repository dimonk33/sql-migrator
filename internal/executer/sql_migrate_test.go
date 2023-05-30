package executer

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	migfile "github.com/dimonk33/sql-migrator/internal/file"
	"github.com/stretchr/testify/require"
)

const (
	testGoodSQLFile = "111111_sql_migration.sql"
	testBadSQLFile  = "555555_bad_sql_migration.sql"
)

func TestSQLMigrate_parseFile(t *testing.T) {
	var (
		srcPath      string
		testGoodData []byte
		err          error
	)

	srcPath, err = os.Getwd()
	require.NoError(t, err)

	absTestDataPath := filepath.Join(srcPath, testDataPath)

	testGoodData, err = os.ReadFile(filepath.Join(absTestDataPath, testGoodSQLFile))
	require.NoError(t, err)

	testGoodDataStr := string(testGoodData)
	downPartStart := strings.Index(testGoodDataStr, migfile.SQLDownPartID)

	type args struct {
		path string
		dir  int
	}

	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "parse file Up ok",
			args: args{
				path: filepath.Join(absTestDataPath, testGoodSQLFile),
				dir:  UpDirection,
			},
			want:    strings.TrimPrefix(testGoodDataStr[:downPartStart], migfile.SQLUpPartID),
			wantErr: false,
		},
		{
			name: "parse file Down ok",
			args: args{
				path: filepath.Join(absTestDataPath, testGoodSQLFile),
				dir:  DownDirection,
			},
			want:    strings.TrimPrefix(testGoodDataStr[downPartStart:], migfile.SQLDownPartID),
			wantErr: false,
		},
		{
			name: "parse file without direction",
			args: args{
				path: filepath.Join(absTestDataPath, testGoodSQLFile),
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "parse file fail",
			args: args{
				path: filepath.Join(absTestDataPath, testBadSQLFile),
				dir:  UpDirection,
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &SQLMigrate{}
			got, err := sm.parseFile(tt.args.path, tt.args.dir)
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

//nolint:lll
func TestSQLMigrate_extractSQLRequest(t *testing.T) {
	type args struct {
		text string
	}

	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "test 1",
			args: args{
				text: `CREATE TABLE test_sql_migration
(
    id         SERIAL PRIMARY KEY,
    name       varchar(255) NOT NULL,
    created_at timestamp    NOT NULL default now()
);
DROP TABLE test_sql_migration;
SELECT * FROM test_sql_migration;
`,
			},
			want: []string{
				"CREATE TABLE test_sql_migration(    id         SERIAL PRIMARY KEY,    name       varchar(255) NOT NULL,    created_at timestamp    NOT NULL default now())",
				"DROP TABLE test_sql_migration",
				"SELECT * FROM test_sql_migration",
			},
		},
		{
			name: "test 2",
			args: args{
				text: `
DROP TABLE test_sql_migration;;;;;
SELECT * FROM test_sql_migration;;;;;
`,
			},
			want: []string{
				"DROP TABLE test_sql_migration",
				"SELECT * FROM test_sql_migration",
			},
		},
		{
			name: "test 3",
			args: args{
				text: `;;;;;`,
			},
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &SQLMigrate{}
			if got := sm.extractSQLRequest(tt.args.text); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractSQLRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}
