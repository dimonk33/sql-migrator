package cmd

import (
	"fmt"

	"github.com/dimonk33/sql-migrator/pkg/gomigrator"
	"github.com/spf13/cobra"
)

// downCmd команда для отката миграции.
var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Откат последней миграции",
	Args:  cobra.MinimumNArgs(0),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Parent().PersistentPreRunE(cmd.Parent(), args)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		const errDownPrefix = "откат миграции: "

		m, err := gomigrator.New(logg, migrateDir, &dbParam)
		if err != nil {
			return fmt.Errorf("%s%w", errDownPrefix, err)
		}

		err = m.Down()
		if err != nil {
			return fmt.Errorf("%s%w", errDownPrefix, err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(downCmd)
}
