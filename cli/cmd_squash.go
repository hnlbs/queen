package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func (app *App) squashCmd() *cobra.Command {
	var (
		into    string
		fromVer string
		toVer   string
		dryRun  bool
	)

	cmd := &cobra.Command{
		Use:   "squash VERSIONS...",
		Short: "Combine multiple migrations into one",
		RunE: func(cmd *cobra.Command, args []string) error {
			var versions []string
			if len(args) > 0 {
				versions = strings.Split(args[0], ",")
			} else if fromVer != "" && toVer != "" {
				ctx := context.Background()
				q, err := app.setupQueen(ctx)
				if err != nil {
					return err
				}
				defer func() { _ = q.Close() }()

				statuses, err := q.Status(ctx)
				if err != nil {
					return fmt.Errorf("failed to get status: %w", err)
				}

				inRange := false
				for _, s := range statuses {
					if s.Version == fromVer {
						inRange = true
					}
					if inRange {
						versions = append(versions, s.Version)
					}
					if s.Version == toVer {
						break
					}
				}
			} else {
				return fmt.Errorf("provide either VERSIONS or use --from/--to flags")
			}

			if len(versions) < 2 {
				return fmt.Errorf("need at least 2 migrations to squash")
			}

			if into == "" {
				return fmt.Errorf("--into flag is required")
			}

			if dryRun {
				fmt.Println("Dry run mode - no changes will be made")
				fmt.Println()
			}

			fmt.Printf("Squashing %d migrations into '%s':\n", len(versions), into)
			for _, v := range versions {
				fmt.Printf("  • %s\n", v)
			}
			fmt.Println()

			if dryRun {
				fmt.Println("Steps that would be performed:")
				fmt.Println("  1. Analyze SQL from all migrations")
				fmt.Println("  2. Combine into single migration")
				fmt.Println("  3. Create new migration file")
				fmt.Println("  4. Archive old migration files")
				fmt.Println()
				fmt.Println("Run without --dry-run to execute")
				return nil
			}

			return fmt.Errorf("squash command not yet fully implemented")
		},
	}

	cmd.Flags().StringVar(&into, "into", "", "Name for the squashed migration (required)")
	cmd.Flags().StringVar(&fromVer, "from", "", "Start version for range squash")
	cmd.Flags().StringVar(&toVer, "to", "", "End version for range squash")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview squash without making changes")

	return cmd
}
