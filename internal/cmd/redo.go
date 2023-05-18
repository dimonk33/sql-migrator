package cmd

import (
	"fmt"

	"github.com/dimonk33/sql-migrator/pkg/gomigrator"
	"github.com/spf13/cobra"
)

const ErrRedoPrefix = "повтор миграций: "

// redoCmd повтор последней миграции.
var redoCmd = &cobra.Command{
	Use:   "redo",
	Short: "Повтор последней миграции",
	Args:  cobra.MinimumNArgs(0),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Parent().PersistentPreRunE(cmd.Parent(), args)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		m, err := gomigrator.New(logg, migrateDir, &dbParam)
		if err != nil {
			return fmt.Errorf("%s%w", ErrRedoPrefix, err)
		}

		err = m.Redo()
		if err != nil {
			return fmt.Errorf("%s%w", ErrRedoPrefix, err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(redoCmd)
}
