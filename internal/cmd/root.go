package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var rootCmd = &cobra.Command{
	Use:   "migrator",
	Short: "Migrator - программа для изменения схемы БД Postgresql",
	Long: `Программа для гибкого управления структурой БД,
разрабатываемая в рамках обучения в OTUS`,
}

func init() {
	rootCmd.AddCommand(cmdUp)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
