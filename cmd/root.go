package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "subslice",
	Short: "subslice is a partial data replicator and TDM engine",
	Long:  `A deterministic CLI tool to extract, mask, and stream database sub-slices.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "subslice.yaml", "path to config file")
}
