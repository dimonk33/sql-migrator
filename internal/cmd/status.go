package cmd

import (
	"fmt"
	"strings"

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

		var list []gomigrator.MigrateStatus

		if list, err = m.Status(); err != nil {
			return fmt.Errorf("%s%w", errStatusPrefix, err)
		}

		builder := strings.Builder{}
		builder.WriteString(`
Идентификатор миграции                  дата применения
-------------------------------------------------------------------------------
`)
		for _, item := range list {
			builder.WriteString(fmt.Sprintf(
				"%s - %s\n",
				item.Name,
				item.UpdatedAt.Format("02/01/2006 15:04:05"),
			))
		}

		fmt.Print(builder.String())

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
