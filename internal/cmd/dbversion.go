package cmd

import (
	"fmt"
	"strings"

	"github.com/dimonk33/sql-migrator/pkg/gomigrator"
	"github.com/spf13/cobra"
)

// statusCmd состояние миграций.
var versionCmd = &cobra.Command{
	Use:   "dbversion",
	Short: "Версия базы данных",
	Args:  cobra.MinimumNArgs(0),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Parent().PersistentPreRunE(cmd.Parent(), args)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		const errVersionPrefix = "версия: "

		m, err := gomigrator.New(logg, migrateDir, &dbParam)
		if err != nil {
			return fmt.Errorf("%s%w", errVersionPrefix, err)
		}

		var v string

		if v, err = m.Version(); err != nil {
			return fmt.Errorf("%s%w", errVersionPrefix, err)
		}

		builder := strings.Builder{}
		builder.WriteString("Версия базы данных: ")
		builder.WriteString(v)

		fmt.Print(builder.String())

		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
