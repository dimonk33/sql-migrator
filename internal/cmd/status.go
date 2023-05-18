package cmd

import (
	"fmt"

	"github.com/dimonk33/sql-migrator/pkg/gomigrator"
	"github.com/spf13/cobra"
)

// statusCmd состояние миграций.
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Статус применения миграций",
	Args:  cobra.MinimumNArgs(0),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Parent().PersistentPreRunE(cmd.Parent(), args)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		const errStatusPrefix = "статус: "
		m, err := gomigrator.New(logg, migrateDir, &dbParam)
		if err != nil {
			return fmt.Errorf("%s%w", errStatusPrefix, err)
		}

		err = m.Status()
		if err != nil {
			return fmt.Errorf("%s%w", errStatusPrefix, err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
