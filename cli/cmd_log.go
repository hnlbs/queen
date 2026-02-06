package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/honeynil/queen"
	naturalsort "github.com/honeynil/queen/internal/sort"
	"github.com/spf13/cobra"
)

func (app *App) logCmd() *cobra.Command {
	var (
		last         int
		since        string
		withDuration bool
		withMeta     bool
		reverse      bool
	)

	cmd := &cobra.Command{
		Use:   "log",
		Short: "Show migration history",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			q, err := app.setupQueen(ctx)
			if err != nil {
				return err
			}
			defer func() { _ = q.Close() }()

			// Get all applied migrations from database
			applied, err := q.Driver().GetApplied(ctx)
			if err != nil {
				return fmt.Errorf("failed to get applied migrations: %w", err)
			}

			if len(applied) == 0 {
				if !app.config.JSON {
					fmt.Println("No migrations applied yet")
				}
				return nil
			}

			// Sort by version using natural sort
			sort.Slice(applied, func(i, j int) bool {
				return naturalsort.Compare(applied[i].Version, applied[j].Version) < 0
			})

			// Filter by date if --since is provided
			if since != "" {
				sinceTime, err := time.Parse("2006-01-02", since)
				if err != nil {
					return fmt.Errorf("invalid date format for --since (expected YYYY-MM-DD): %w", err)
				}

				filtered := make([]queen.Applied, 0)
				for _, a := range applied {
					if a.AppliedAt.After(sinceTime) || a.AppliedAt.Equal(sinceTime) {
						filtered = append(filtered, a)
					}
				}
				applied = filtered
			}

			// Apply --reverse flag
			if reverse {
				for i, j := 0, len(applied)-1; i < j; i, j = i+1, j-1 {
					applied[i], applied[j] = applied[j], applied[i]
				}
			}

			// Apply --last limit
			if last > 0 && last < len(applied) {
				applied = applied[len(applied)-last:]
			}

			// Output
			if app.config.JSON {
				return outputLogJSON(applied)
			}

			outputLogTable(applied, withDuration, withMeta)
			return nil
		},
	}

	cmd.Flags().IntVar(&last, "last", 0, "Show last N migrations")
	cmd.Flags().StringVar(&since, "since", "", "Show migrations since date (YYYY-MM-DD)")
	cmd.Flags().BoolVar(&withDuration, "with-duration", false, "Show execution duration")
	cmd.Flags().BoolVar(&withMeta, "with-meta", false, "Show all metadata (applied_by, hostname, etc.)")
	cmd.Flags().BoolVar(&reverse, "reverse", false, "Show in reverse order (newest first)")

	return cmd
}

// outputLogTable prints migrations in a formatted table.
func outputLogTable(applied []queen.Applied, withDuration, withMeta bool) {
	// Determine column widths
	maxVersion := len("VERSION")
	maxName := len("NAME")
	for _, a := range applied {
		if len(a.Version) > maxVersion {
			maxVersion = len(a.Version)
		}
		if len(a.Name) > maxName {
			maxName = len(a.Name)
		}
	}

	// Print header
	header := fmt.Sprintf("%-*s  %-*s  %s", maxVersion, "VERSION", maxName, "NAME", "APPLIED AT")
	if withDuration {
		header += "  DURATION"
	}
	if withMeta {
		header += "  APPLIED BY  HOSTNAME  ENV"
	}
	fmt.Println(header)
	fmt.Println(strings.Repeat("-", len(header)+20))

	// Print rows
	for _, a := range applied {
		appliedAt := a.AppliedAt.Format("2006-01-02 15:04:05")
		row := fmt.Sprintf("%-*s  %-*s  %s", maxVersion, a.Version, maxName, a.Name, appliedAt)

		if withDuration {
			if a.DurationMS > 0 {
				row += fmt.Sprintf("  %dms", a.DurationMS)
			} else {
				row += "  -"
			}
		}

		if withMeta {
			appliedBy := a.AppliedBy
			if appliedBy == "" {
				appliedBy = "-"
			}
			hostname := a.Hostname
			if hostname == "" {
				hostname = "-"
			}
			env := a.Environment
			if env == "" {
				env = "-"
			}
			row += fmt.Sprintf("  %s  %s  %s", appliedBy, hostname, env)
		}

		fmt.Println(row)
	}

	fmt.Printf("\nTotal: %d migrations\n", len(applied))
}

// outputLogJSON prints migrations in JSON format.
func outputLogJSON(applied []queen.Applied) error {
	data, err := json.MarshalIndent(applied, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}
