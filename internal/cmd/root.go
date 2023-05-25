package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/dimonk33/sql-migrator/internal/logger"
	"github.com/dimonk33/sql-migrator/pkg/gomigrator"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	// Путь до миграций по умолчанию.
	defaultMigrateDir = "./migrations"

	// Тип миграций по умолчанию.
	defaultMigrateType = "sql"

	// Уровень логирования по умолчанию.
	defaultLoggerLevel = logger.LevelError

	// Путь до файла конфигурации.
	defaultConfigFilename = "./config/migrator"

	// Префикс для переменных среды.
	envPrefix = "GM"

	// Замена дефисных названий на camelCase для конфигурационного файла.
	replaceHyphenWithCamelCase = false
)

var (
	migrateDir string
	dbParam    gomigrator.DBConnParam
	logLevel   string
	logg       *logger.Logger
)

// rootCmd базовая команда.
var rootCmd = &cobra.Command{
	Use:   "migrator",
	Short: "Migrator - программа для изменения схемы БД Postgresql",
	Long: `Программа для гибкого управления структурой БД,
разрабатываемая в рамках обучения в OTUS`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initializeConfig(cmd)
	},
}

// Execute выполнение дочерних команд.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dbParam.Host, "db-host", "localhost", "Хост для БД")
	rootCmd.PersistentFlags().StringVar(&dbParam.Port, "db-port", "5432", "Порт для БД")
	rootCmd.PersistentFlags().StringVar(&dbParam.Name, "db-name", "", "Имя БД")
	rootCmd.PersistentFlags().StringVar(&dbParam.User, "db-user", "", "Пользователь для БД")
	rootCmd.PersistentFlags().StringVar(&dbParam.Password, "db-password", "", "Пароль для БД")
	rootCmd.PersistentFlags().StringVar(&dbParam.SSL, "db-ssl", "disable", "Включение SSL для БД")
	rootCmd.PersistentFlags().StringVar(&migrateDir, "migrate", defaultMigrateDir, "Путь до каталога с миграциями")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", defaultLoggerLevel, "Уровень логирования")

	logg = logger.New(logLevel)
}

func initializeConfig(cmd *cobra.Command) error {
	v := viper.New()

	v.SetConfigName(defaultConfigFilename)
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil {
		// It's okay if there isn't a config file
		var _t0 viper.ConfigFileNotFoundError
		if !errors.Is(err, _t0) {
			return err
		}
	}

	v.SetEnvPrefix(envPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()
	bindFlags(cmd, v)

	return nil
}

// Связывание флагов cobra с настройками из viper.
func bindFlags(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		configName := f.Name
		if replaceHyphenWithCamelCase {
			configName = strings.ReplaceAll(f.Name, "-", "")
		}

		if !f.Changed && v.IsSet(configName) {
			val := v.Get(configName)
			_ = cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
		}
	})
}
