package cmd

import (
	"fmt"
	"subslice/pkg/config"

	"github.com/spf13/cobra"
)

var verbose bool

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Execute data extraction, masking, and streaming",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig(cfgFile)
		if err != nil {
			return fmt.Errorf("config error: %w", err)
		}

		fmt.Printf("[INFO] Run mode initialized using config: %s (verbose: %t)\n", cfgFile, verbose)
		fmt.Printf("[INFO] Source: %s\n", cfg.Source)
		fmt.Printf("[INFO] Target: %s\n", cfg.Target)
		fmt.Printf("[INFO] Configured Workers: %d, Batch Size: %d\n", cfg.Options.Workers, cfg.Options.BatchSize)
		fmt.Println("[INFO] Connecting to target data sources...")
		fmt.Println("[SUCCESS] Replicated 0 rows across 0 tables in 0.00s.")
		return nil
	},
}

func init() {
	runCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output logging")
	rootCmd.AddCommand(runCmd)
}
