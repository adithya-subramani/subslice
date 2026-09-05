package cmd

import (
	"fmt"
	"subslice/pkg/config"
	"subslice/pkg/connector"

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
		fmt.Printf("[INFO] Target Root: entity='%s', where='%s'\n", cfg.Root.Table, cfg.Root.Where)
		fmt.Printf("[INFO] Connecting to storage source (%s)...\n", cfg.Source.Driver)

		conn, err := connector.NewConnector(cfg.Source.Driver, cfg.Source.URL)
		if err != nil {
			fmt.Printf("[WARN] Source connection skipped/failed: %v\n", err)
			fmt.Println("[INFO] Operating in dry-run offline mode...")
		} else {
			defer conn.Close()
			graph, err := conn.DiscoverGraph()
			if err != nil {
				return fmt.Errorf("storage inspection failed: %w", err)
			}
			fmt.Printf("[INFO] Inspected storage: %d entities, %d relationships detected.\n", len(graph.Entities), len(graph.Relationships))
		}

		fmt.Println("[SUCCESS] Plan step completed successfully.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(planCmd)
}
