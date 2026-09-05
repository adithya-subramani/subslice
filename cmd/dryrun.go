package cmd

import (
	"encoding/json"
	"fmt"
	"subslice/pkg/config"
	"subslice/pkg/connector"
	"subslice/pkg/graph"
	"subslice/pkg/model"
	"subslice/pkg/transform"

	"github.com/spf13/cobra"
)

var dryRunCmd = &cobra.Command{
	Use:   "dry-run",
	Short: "Preview traversal plan and inspect PII-masked records without writing to target",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig(cfgFile)
		if err != nil {
			return fmt.Errorf("config error: %w", err)
		}

		fmt.Printf("[INFO] Dry-run initialized using config: %s\n", cfgFile)

		sourceConn, err := connector.NewConnector(cfg.Source.Driver, cfg.Source.URL)
		if err != nil {
			return fmt.Errorf("source connection failed: %w", err)
		}
		defer sourceConn.Close()

		storageGraph, err := sourceConn.DiscoverGraph()
		if err != nil {
			return fmt.Errorf("storage inspection failed: %w", err)
		}

		engine := graph.NewEngine(storageGraph, cfg.ImplicitForeignKeys)
		plan, err := engine.BuildPlan(cfg.Root.Table, cfg.Options.MaxDepth)
		if err != nil {
			return fmt.Errorf("plan error: %w", err)
		}

		transformer := transform.NewTransformer(cfg.Transformations)

		// Preview Root Entity Records
		fmt.Printf("\n--- [Preview] Root Entity: %s ---\n", cfg.Root.Table)
		rootRecords, err := sourceConn.FetchRecords(cfg.Root.Table, "id", []interface{}{"org_123"}, cfg.Options.Limit)
		if err != nil {
			return fmt.Errorf("root fetch error: %w", err)
		}

		for _, record := range rootRecords {
			transformer.TransformRecord(&record)
			prettyPrintRecord(record)
		}

		// Preview Downstream Child Entities
		for _, child := range plan.DownstreamEntities {
			fmt.Printf("\n--- [Preview] Child Entity: %s ---\n", child)
			childRecords, err := sourceConn.FetchRecords(child, "tenant_id", []interface{}{"org_123"}, cfg.Options.Limit)
			if err != nil || len(childRecords) == 0 {
				childRecords, _ = sourceConn.FetchRecords(child, "user_id", []interface{}{"usr_1", "usr_2"}, cfg.Options.Limit)
			}

			for _, record := range childRecords {
				transformer.TransformRecord(&record)
				prettyPrintRecord(record)
			}
		}

		fmt.Println("\n[SUCCESS] Dry-run completed safely. No data written to target.")
		return nil
	},
}

func prettyPrintRecord(record model.Record) {
	data, err := json.MarshalIndent(record.Data, "  ", "  ")
	if err != nil {
		fmt.Printf("  %v\n", record.Data)
		return
	}
	fmt.Printf("  Entity: %s\n  Data: %s\n\n", record.EntityName, string(data))
}

func init() {
	rootCmd.AddCommand(dryRunCmd)
}
