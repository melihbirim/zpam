package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "zpo",
	Short: "ZPO - Fast spam filter (Baby Donkey)",
	Long: `ZPO is a lightning-fast spam filter that processes emails in under 5ms.
It rates emails from 1-5 and automatically moves spam (4-5 rating) to spam folder.

Named after baby donkey - free, fast, and reliable.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("ZPO - Baby Donkey Spam Filter")
		fmt.Println("Use 'zpo --help' for usage information")
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(filterCmd)
	rootCmd.AddCommand(testCmd)
	rootCmd.AddCommand(configCmd)
} 