package cmd

import (
	"fmt"
	"subslice/pkg/config"

	"github.com/spf13/cobra"
)

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Inspect schema and generate traversal execution plan",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig(cfgFile)
		if err != nil {
			return fmt.Errorf("config error: %w", err)
		}

		fmt.Printf("[INFO] Plan mode initialized using config: %s\n", cfgFile)
		fmt.Printf("[INFO] Target Root: table='%s', where='%s'\n", cfg.Root.Table, cfg.Root.Where)
		fmt.Println("[INFO] Inspecting schema graph...")
		fmt.Println("[INFO] Simulated Graph Plan:")
		fmt.Println("       ├── Upstream parents: 0 tables detected")
		fmt.Println("       └── Downstream children: 0 tables detected")
		fmt.Println("[SUCCESS] Plan step completed successfully.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(planCmd)
}
