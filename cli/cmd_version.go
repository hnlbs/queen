package cli

import (
	"context"
	"fmt"

	"github.com/honeynil/queen"
	"github.com/spf13/cobra"
)

func (app *App) versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show current migration version",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			q, err := app.setupQueen(ctx)
			if err != nil {
				return err
			}
			defer func() { _ = q.Close() }()

			statuses, err := q.Status(ctx)
			if err != nil {
				return fmt.Errorf("failed to get migration status: %w", err)
			}

			var latestVersion string
			var latestName string
			for _, s := range statuses {
				if s.Status == queen.StatusApplied {
					latestVersion = s.Version
					latestName = s.Name
				}
			}

			if latestVersion == "" {
				fmt.Println("No migrations have been applied yet")
			} else {
				fmt.Printf("Current version: %s (%s)\n", latestVersion, latestName)
			}

			return nil
		},
	}
}
