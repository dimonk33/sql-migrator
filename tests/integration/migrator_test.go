//go:build integration

package integration_test

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/dimonk33/sql-migrator/internal/logger"
	"github.com/dimonk33/sql-migrator/pkg/gomigrator"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type MigratorSuite struct {
	suite.Suite
	ctx           context.Context
	migrator      *gomigrator.Migrator
	migrationPath string
	dbConn        gomigrator.DBConnParam
}

func TestMigratorSuite(t *testing.T) {
	suite.Run(t, new(MigratorSuite))
}

func (m *MigratorSuite) SetupSuite() {
	mpath := os.Getenv("TESTDATA_PATH")
	if mpath == "" {
		flag.StringVar(&mpath, "migratePath", "./testdata", "Path to migrations")
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

	logg := logger.New(logger.LevelDebug)

	m.migrator, err = gomigrator.New(logg, m.migrationPath, &m.dbConn)
	require.NoError(m.T(), err)

	m.ctx = context.Background()
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

func (m *MigratorSuite) TestCreateUpSuccess() {
}

func (m *MigratorSuite) TestCreateDownSuccess() {
}

func (m *MigratorSuite) TestCreateRedoSuccess() {
}

func (m *MigratorSuite) TestStatusSuccess() {
}

func (m *MigratorSuite) TestDBVersionSuccess() {
}
