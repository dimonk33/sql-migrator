package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"strings"
)

var cmdUp = &cobra.Command{
	Use:   "up",
	Short: "up - команда для применения миграций",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("cmd up " + strings.Join(args, " "))
	},
}
