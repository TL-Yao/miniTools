package commands

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "minitool",
	Short: "A collection of CLI utilities",
	Long:  `minitool is a CLI toolkit with various utilities for daily development tasks.`,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(tsCmd)
	rootCmd.AddCommand(padAddrCmd)
	rootCmd.AddCommand(lowerCmd)
	rootCmd.AddCommand(trCmd)
	rootCmd.AddCommand(dbCmd)
}
