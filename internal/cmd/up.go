package cmd

import (
	"fmt"

	"github.com/dimonk33/sql-migrator/pkg/gomigrator"
	"github.com/spf13/cobra"
)

const ErrUpPrefix = "применение миграций: "

// upCmd команда для применения транзакций.
var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Применение миграций",
	Args:  cobra.MinimumNArgs(0),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Parent().PersistentPreRunE(cmd.Parent(), args)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		m, err := gomigrator.New(logg, migrateDir, &dbParam)
		if err != nil {
			return fmt.Errorf("%s%w", ErrUpPrefix, err)
		}

		err = m.Up()
		if err != nil {
			return fmt.Errorf("%s%w", ErrUpPrefix, err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(upCmd)
}
