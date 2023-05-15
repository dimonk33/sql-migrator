package cmd

import (
	"fmt"

	"github.com/dimonk33/sql-migrator/pkg/gomigrator"

	"github.com/spf13/cobra"
)

const ErrWrapPrefix = "создание миграции: %w"

var migrateType string

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Создание шаблона для написания миграции",
	Args:  cobra.MinimumNArgs(1),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Parent().PersistentPreRunE(cmd.Parent(), args)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("create called")

		m, err := gomigrator.New(logg, migrateDir, &dbParam)
		if err != nil {
			return fmt.Errorf(ErrWrapPrefix, err)
		}

		err = m.Create(args[0], migrateType)
		if err != nil {
			return fmt.Errorf(ErrWrapPrefix, err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.Flags().StringVar(&migrateType, "migrateType", defaultMigrateType, "Тип миграции (sql/go)")
}
