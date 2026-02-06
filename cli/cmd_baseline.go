package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// baselineCmd returns the baseline command.
func (app *App) baselineCmd() *cobra.Command {
	var (
		name   string
		at     string
		dryRun bool
	)

	cmd := &cobra.Command{
		Use:   "baseline",
		Short: "Create baseline migration from current schema",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				name = "baseline"
			}

			ctx := context.Background()
			q, err := app.setupQueen(ctx)
			if err != nil {
				return err
			}
			defer func() { _ = q.Close() }()

			if dryRun {
				fmt.Println("Dry run mode - no changes will be made")
				fmt.Println()
			}

			fmt.Printf("Creating baseline migration: %s\n", name)
			if at != "" {
				fmt.Printf("At version: %s\n", at)
			}
			fmt.Println()

			if dryRun {
				fmt.Println("Steps that would be performed:")
				fmt.Println("  1. Extract current database schema")
				fmt.Println("  2. Generate migration file")
				if at != "" {
					fmt.Printf("  3. Insert baseline record at version %s\n", at)
				} else {
					fmt.Println("  3. Insert baseline record")
				}
				fmt.Println("  4. Mark previous migrations as applied")
				fmt.Println()
				fmt.Println("Run without --dry-run to execute")
				return nil
			}

			return fmt.Errorf("baseline command not yet fully implemented")
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Name for the baseline migration")
	cmd.Flags().StringVar(&at, "at", "", "Version number for baseline (optional)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview baseline without making changes")

	return cmd
}
