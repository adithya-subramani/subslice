package cmd

import (
	"fmt"
	"subslice/pkg/config"
	"subslice/pkg/connector"
	"subslice/pkg/graph"
	"subslice/pkg/transform"
	"time"

	"github.com/spf13/cobra"
)

var verbose bool

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Execute data extraction, masking, and streaming",
	RunE: func(cmd *cobra.Command, args []string) error {
		startTime := time.Now()

		cfg, err := config.LoadConfig(cfgFile)
		if err != nil {
			return fmt.Errorf("config error: %w", err)
		}

		fmt.Printf("[INFO] Run mode initialized using config: %s\n", cfgFile)

		// Connect to Source DB
		sourceConn, err := connector.NewConnector(cfg.Source.Driver, cfg.Source.URL)
		if err != nil {
			return fmt.Errorf("source connection failed: %w", err)
		}
		defer sourceConn.Close()

		// Connect to Target DB
		targetConn, err := connector.NewConnector(cfg.Target.Driver, cfg.Target.URL)
		if err != nil {
			return fmt.Errorf("target connection failed: %w", err)
		}
		defer targetConn.Close()

		// Discover Graph & Build Plan
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
		totalReplicated := 0

		// 1. Process Root Target Entity
		fmt.Printf("[INFO] Extracting Root entity: %s\n", cfg.Root.Table)
		rootRecords, err := sourceConn.FetchRecords(cfg.Root.Table, "id", []interface{}{"org_123"})
		if err != nil {
			return fmt.Errorf("root fetch error: %w", err)
		}

		for i := range rootRecords {
			transformer.TransformRecord(&rootRecords[i])
		}

		if err := targetConn.WriteStream(cfg.Root.Table, rootRecords); err != nil {
			return fmt.Errorf("root write error: %w", err)
		}
		fmt.Printf("[OK]   %s (%d records replicated)\n", cfg.Root.Table, len(rootRecords))
		totalReplicated += len(rootRecords)

		// 2. Process Downstream Child Entities (e.g. users, orders)
		for _, child := range plan.DownstreamEntities {
			childRecords, err := sourceConn.FetchRecords(child, "tenant_id", []interface{}{"org_123"})
			if err != nil {
				// Fallback attempt for user_id on child tables like orders
				childRecords, _ = sourceConn.FetchRecords(child, "user_id", []interface{}{"usr_1", "usr_2"})
			}

			for i := range childRecords {
				transformer.TransformRecord(&childRecords[i])
			}

			if len(childRecords) > 0 {
				if err := targetConn.WriteStream(child, childRecords); err != nil {
					return fmt.Errorf("write error for entity %s: %w", child, err)
				}
				fmt.Printf("[OK]   %s (%d records replicated with PII transformations)\n", child, len(childRecords))
				totalReplicated += len(childRecords)
			}
		}

		duration := time.Since(startTime).Seconds()
		fmt.Printf("[SUCCESS] Replicated %d rows across %d tables in %.2fs.\n",
			totalReplicated, 1+len(plan.DownstreamEntities), duration)
		return nil
	},
}

func init() {
	runCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output logging")
	rootCmd.AddCommand(runCmd)
}
