//go:build integration

package integration_test

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dimonk33/sql-migrator/internal/logger"
	"github.com/dimonk33/sql-migrator/pkg/gomigrator"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	SQLMigrateName = "111111_sql_migration.sql"
	GoMigrateName  = "222222_go_migration.go"

	SQLMigrationTestTable = "test_sql_migration"
	GoMigrationTestTable  = "test_go_migration"
)

type MigratorSuite struct {
	suite.Suite
	ctx           context.Context
	migrator      *gomigrator.Migrator
	migrationPath string
	dbConn        gomigrator.DBConnParam
	conn          *sqlx.DB
}

func TestMigratorSuite(t *testing.T) {
	suite.Run(t, new(MigratorSuite))
}

func (m *MigratorSuite) SetupSuite() {
	mpath := os.Getenv("TESTDATA_PATH")
	if mpath == "" {
		flag.StringVar(&mpath, "migratePath", "./migrations", "Path to migrations")
		flag.Parse()
	}

	srcPath, err := os.Getwd()
	require.NoError(m.T(), err)

	m.migrationPath = filepath.Join(srcPath, mpath)

	m.dbConn = gomigrator.DBConnParam{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		Name:     os.Getenv("DB_NAME"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		SSL:      os.Getenv("DB_SSL"),
	}

	if m.dbConn.Host == "" {
		m.dbConn = gomigrator.DBConnParam{
			Host:     "localhost",
			Port:     "5432",
			Name:     "migrator",
			User:     "migrator",
			Password: "migrator",
			SSL:      "disable",
		}
	}

	m.ctx = context.Background()

	connStr := fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password='%s' sslmode=%s",
		m.dbConn.Host,
		m.dbConn.Port,
		m.dbConn.Name,
		m.dbConn.User,
		m.dbConn.Password,
		m.dbConn.SSL,
	)

	m.conn, err = sqlx.ConnectContext(m.ctx, "postgres", connStr)
	require.NoError(m.T(), err)

	logg := logger.New(logger.LevelDebug)

	m.migrator, err = gomigrator.New(logg, m.migrationPath, &m.dbConn)
	require.NoError(m.T(), err)
}

func (m *MigratorSuite) SetupTest() {
}

func (m *MigratorSuite) TestCreateEventSuccess() {
	const migrationName = "test_create_sql_migration"

	fname, err := m.migrator.Create(migrationName, gomigrator.SQLMigration)
	require.NoError(m.T(), err)
	require.NotEmpty(m.T(), fname)

	fpath := filepath.Join(m.migrationPath, fname)
	_, err = os.Stat(fpath)
	require.NoError(m.T(), err)

	err = os.RemoveAll(fpath)
	require.NoError(m.T(), err)
}

func (m *MigratorSuite) TestCreateEventFail() {
	const migrationName = "test_create_go_migration"

	fname, err := m.migrator.Create(migrationName, "txt")
	require.Error(m.T(), err)
	require.Empty(m.T(), fname)
}

func (m *MigratorSuite) TestUpDownSuccess() {
	err := m.migrator.Up()
	if err != nil && !errors.Is(err, gomigrator.ErrNoMigrations) {
		require.NoError(m.T(), err)
	}

	require.True(m.T(), m.testTable(SQLMigrationTestTable))
	require.True(m.T(), m.testTable(GoMigrationTestTable))

	var mlist []gomigrator.MigrateStatus
	mlist, err = m.migrator.Status()
	require.NoError(m.T(), err)
	require.Equal(m.T(), 2, len(mlist))

	for i, item := range mlist {
		switch i {
		case 0:
			require.Equal(m.T(), GoMigrateName, item.Name)
		case 1:
			require.Equal(m.T(), SQLMigrateName, item.Name)
		}
	}

	var dbversion string
	dbversion, err = m.migrator.Version()
	require.NoError(m.T(), err)
	require.Equal(m.T(), GoMigrateName, dbversion)

	err = m.migrator.Down()
	require.NoError(m.T(), err)
	require.True(m.T(), m.testTable(SQLMigrationTestTable))
	require.False(m.T(), m.testTable(GoMigrationTestTable))

	mlist, err = m.migrator.Status()
	require.NoError(m.T(), err)
	require.Equal(m.T(), 1, len(mlist))
	require.Equal(m.T(), SQLMigrateName, mlist[0].Name)

	err = m.migrator.Down()
	require.NoError(m.T(), err)

	mlist, err = m.migrator.Status()
	require.NoError(m.T(), err)
	require.Equal(m.T(), 0, len(mlist))

	require.False(m.T(), m.testTable(SQLMigrationTestTable))
	require.False(m.T(), m.testTable(GoMigrationTestTable))
}

func (m *MigratorSuite) TestCreateRedoSuccess() {
	err := m.migrator.Up()
	if err != nil && !errors.Is(err, gomigrator.ErrNoMigrations) {
		require.NoError(m.T(), err)
	}

	require.True(m.T(), m.testTable(SQLMigrationTestTable))
	require.True(m.T(), m.testTable(GoMigrationTestTable))

	var mlist1 []gomigrator.MigrateStatus
	mlist1, err = m.migrator.Status()
	require.NoError(m.T(), err)
	require.Equal(m.T(), 2, len(mlist1))

	time.Sleep(5 * time.Second)

	err = m.migrator.Redo()
	require.NoError(m.T(), err)
	require.True(m.T(), m.testTable(SQLMigrationTestTable))
	require.True(m.T(), m.testTable(GoMigrationTestTable))

	var mlist2 []gomigrator.MigrateStatus
	mlist2, err = m.migrator.Status()
	require.NoError(m.T(), err)
	require.Equal(m.T(), 2, len(mlist2))

	require.Greater(m.T(), mlist2[0].UpdatedAt.Unix(), mlist1[0].UpdatedAt.Unix())

	err = m.migrator.Down()
	require.NoError(m.T(), err)
	require.True(m.T(), m.testTable(SQLMigrationTestTable))
	require.False(m.T(), m.testTable(GoMigrationTestTable))

	err = m.migrator.Down()
	require.NoError(m.T(), err)
	require.False(m.T(), m.testTable(SQLMigrationTestTable))
	require.False(m.T(), m.testTable(GoMigrationTestTable))
}

func (m *MigratorSuite) testTable(name string) bool {
	sqlReq := `SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE lower(table_name) = lower($1))`

	var exists bool

	err := m.conn.GetContext(m.ctx, &exists, sqlReq, name)
	require.NoError(m.T(), err)

	return exists
}
