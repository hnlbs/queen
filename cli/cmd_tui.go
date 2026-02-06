package cli

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/honeynil/queen/cli/tui"
	"github.com/spf13/cobra"
)

func (app *App) tuiCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Launch interactive Terminal UI",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			q, err := app.setupQueen(ctx)
			if err != nil {
				return err
			}
			defer func() { _ = q.Close() }()

			// Create TUI model
			model := tui.NewModel(q, ctx)

			// Start TUI program
			p := tea.NewProgram(
				model,
				tea.WithAltScreen(),
				tea.WithMouseCellMotion(),
			)

			if _, err := p.Run(); err != nil {
				return fmt.Errorf("failed to start TUI: %w", err)
			}

			return nil
		},
	}

	return cmd
}
