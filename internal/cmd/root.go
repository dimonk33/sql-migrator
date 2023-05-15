package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/dimonk33/sql-migrator/internal/logger"

	"github.com/dimonk33/sql-migrator/internal/db"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/spf13/cobra"
)

const (
	// Путь до миграций по умолчанию
	defaultMigrateDir = "./migrations"

	// Путь до миграций по умолчанию
	defaultMigrateType = "sql"

	// Путь до миграций по умолчанию
	defaultLoggerLevel = logger.LevelError

	// Путь до файла конфигурации
	defaultConfigFilename = "./config/migrator"

	// Префикс для переменных среды
	envPrefix = "GM"

	// Замена дефисных названий на camelCase для конфигурационного файла
	replaceHyphenWithCamelCase = false
)

var (
	migrateDir string
	dbParam    db.ConnParam
	logLevel   string
	logg       *logger.Logger
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "migrator",
	Short: "Migrator - программа для изменения схемы БД Postgresql",
	Long: `Программа для гибкого управления структурой БД,
разрабатываемая в рамках обучения в OTUS`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// You can bind cobra and viper in a few locations, but PersistencePreRunE on the root command works well
		return initializeConfig(cmd)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
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

	// Set the base name of the config file, without the file extension.
	v.SetConfigName(defaultConfigFilename)

	// Set as many paths as you like where viper should look for the
	// config file. We are only looking in the current working directory.
	v.AddConfigPath(".")

	// Attempt to read the config file, gracefully ignoring errors
	// caused by a config file not being found. Return an error
	// if we cannot parse the config file.
	if err := v.ReadInConfig(); err != nil {
		// It's okay if there isn't a config file
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	// When we bind flags to environment variables expect that the
	// environment variables are prefixed, e.g. a flag like --number
	// binds to an environment variable STING_NUMBER. This helps
	// avoid conflicts.
	v.SetEnvPrefix(envPrefix)

	// Environment variables can't have dashes in them, so bind them to their equivalent
	// keys with underscores, e.g. --favorite-color to STING_FAVORITE_COLOR
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	// Bind to environment variables
	// Works great for simple config names, but needs help for names
	// like --favorite-color which we fix in the bindFlags function
	v.AutomaticEnv()

	// Bind the current command's flags to viper
	bindFlags(cmd, v)

	return nil
}

// Bind each cobra flag to its associated viper configuration (config file and environment variable)
func bindFlags(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// Determine the naming convention of the flags when represented in the config file
		configName := f.Name
		// If using camelCase in the config file, replace hyphens with a camelCased string.
		// Since viper does case-insensitive comparisons, we don't need to bother fixing the case, and only need to remove the hyphens.
		if replaceHyphenWithCamelCase {
			configName = strings.ReplaceAll(f.Name, "-", "")
		}

		// Apply the viper config value to the flag when the flag is not set and viper has a value
		if !f.Changed && v.IsSet(configName) {
			val := v.Get(configName)
			_ = cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
		}
	})
}
