package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "zpam",
	Short: "ZPAM - Fast spam filter (Baby Donkey)",
	Long: `ZPAM is a lightning-fast spam filter that processes emails in under 5ms.
It rates emails from 1-5 and automatically moves spam (4-5 rating) to spam folder.

Named after baby donkey - free, fast, and reliable.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("ZPAM - Baby Donkey Spam Filter")
		fmt.Println("Use 'zpam --help' for usage information")
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(filterCmd)
	rootCmd.AddCommand(testCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(benchmarkCmd)
	rootCmd.AddCommand(trainCmd)
} 