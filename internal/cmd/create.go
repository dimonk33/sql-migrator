package cmd

import (
	"fmt"
	"strings"

	"github.com/dimonk33/sql-migrator/pkg/gomigrator"
	"github.com/spf13/cobra"
)

var migrateType string

// createCmd команда для создания миграций.
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Создание шаблона для написания миграции",
	Args:  cobra.MinimumNArgs(1),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Parent().PersistentPreRunE(cmd.Parent(), args)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		const errCreatePrefix = "создание миграции: "

		m, err := gomigrator.New(logg, migrateDir, &dbParam)
		if err != nil {
			return fmt.Errorf("%s%w", errCreatePrefix, err)
		}

		mt := gomigrator.MigrateType(strings.ToLower(strings.TrimSpace(migrateType)))
		if err = mt.Validate(); err != nil {
			return fmt.Errorf("%s%w", errCreatePrefix, err)
		}

		var fname string
		if fname, err = m.Create(args[0], mt); err != nil {
			return fmt.Errorf("%s%w", errCreatePrefix, err)
		}

		fmt.Printf("Создан файл миграции: %s", fname)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.Flags().StringVar(&migrateType, "migrate-type", defaultMigrateType, "Тип миграции (sql/go)")
}
