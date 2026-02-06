// Package tui provides a terminal UI for Queen migrations.
package tui

import (
	"fmt"
	"strings"

	"github.com/honeynil/queen"
)

func (m *Model) renderMigrationsView() string {
	var s strings.Builder

	s.WriteString(TitleStyle.Render("🗂️  Queen Migrations"))
	s.WriteString("\n\n")

	if m.loading {
		s.WriteString("Loading migrations...\n")
		return s.String()
	}

	if m.err != nil {
		s.WriteString(ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		s.WriteString("\n")
		return s.String()
	}

	applied := 0
	pending := 0
	for _, migration := range m.migrations {
		if migration.Status == queen.StatusApplied {
			applied++
		} else {
			pending++
		}
	}

	statsBox := InfoBoxStyle.Render(fmt.Sprintf(
		"Total: %d  •  Applied: %s  •  Pending: %s",
		len(m.migrations),
		AppliedStyle.Render(fmt.Sprintf("%d", applied)),
		PendingStyle.Render(fmt.Sprintf("%d", pending)),
	))
	s.WriteString(statsBox)
	s.WriteString("\n\n")

	if len(m.migrations) == 0 {
		s.WriteString("No migrations found\n")
	} else {
		for i, migration := range m.migrations {
			cursor := "  "
			if i == m.cursor {
				cursor = "❯ "
			}

			statusIcon := "○"
			statusStyle := PendingStyle

			if migration.Status == queen.StatusApplied {
				statusIcon = "✓"
				statusStyle = AppliedStyle
			}

			line := fmt.Sprintf("%s %s %s - %s",
				cursor,
				statusStyle.Render(statusIcon),
				migration.Version,
				migration.Name,
			)

			if i == m.cursor {
				line = SelectedStyle.Render(line)
			} else {
				line = NormalStyle.Render(line)
			}

			s.WriteString(line)

			if i == m.cursor && migration.Status == queen.StatusApplied && migration.AppliedAt != nil {
				info := fmt.Sprintf("\n     Applied at: %s", migration.AppliedAt.Format("2006-01-02 15:04:05"))
				s.WriteString(HelpStyle.Render(info))
			}

			s.WriteString("\n")
		}
	}

	if m.message != "" {
		s.WriteString("\n")
		switch m.messageType {
		case MessageSuccess:
			s.WriteString(AppliedStyle.Render("✓ " + m.message))
		case MessageWarning:
			s.WriteString(PendingStyle.Render("⚠ " + m.message))
		case MessageError:
			s.WriteString(ErrorStyle.Render("✗ " + m.message))
		default:
			s.WriteString(m.message)
		}
		s.WriteString("\n")
	}

	s.WriteString("\n")
	s.WriteString(HelpStyle.Render(
		"↑/↓: navigate  •  enter: apply/rollback  •  u: up  •  d: down  •  1: migrations  •  2: gaps  •  3: help  •  r: refresh  •  q: quit",
	))

	return s.String()
}

func (m *Model) renderGapsView() string {
	var s strings.Builder

	s.WriteString(TitleStyle.Render("🔍 Gap Detection"))
	s.WriteString("\n\n")

	if m.loading {
		s.WriteString("Detecting gaps...\n")
		return s.String()
	}

	if m.err != nil {
		s.WriteString(ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		s.WriteString("\n")
		return s.String()
	}

	warnings := 0
	errors := 0
	for _, gap := range m.gaps {
		switch gap.Severity {
		case "warning":
			warnings++
		case "error":
			errors++
		}
	}

	statsContent := fmt.Sprintf("Total: %d  •  Warnings: %s  •  Errors: %s",
		len(m.gaps),
		PendingStyle.Render(fmt.Sprintf("%d", warnings)),
		ErrorStyle.Render(fmt.Sprintf("%d", errors)),
	)

	if len(m.gaps) == 0 {
		statsContent = AppliedStyle.Render("✓ No gaps detected - migrations are clean!")
	}

	statsBox := InfoBoxStyle.Render(statsContent)
	s.WriteString(statsBox)
	s.WriteString("\n\n")

	if len(m.gaps) > 0 {
		for i, gap := range m.gaps {
			cursor := "  "
			if i == m.cursor {
				cursor = "❯ "
			}

			icon := "⚠"
			style := PendingStyle

			if gap.Severity == "error" {
				icon = "✗"
				style = ErrorStyle
			}

			typeLabel := string(gap.Type)
			switch gap.Type {
			case queen.GapTypeNumbering:
				typeLabel = "numbering"
			case queen.GapTypeApplication:
				typeLabel = "application"
			case queen.GapTypeUnregistered:
				typeLabel = "unregistered"
			}

			line := fmt.Sprintf("%s %s [%s] %s",
				cursor,
				style.Render(icon),
				typeLabel,
				gap.Version,
			)

			if gap.Name != "" {
				line += fmt.Sprintf(" - %s", gap.Name)
			}

			if i == m.cursor {
				line = SelectedStyle.Render(line)
			} else {
				line = NormalStyle.Render(line)
			}

			s.WriteString(line)

			if i == m.cursor {
				desc := fmt.Sprintf("\n     %s", gap.Description)
				if len(gap.BlockedBy) > 0 {
					desc += fmt.Sprintf("\n     Blocks: %s", strings.Join(gap.BlockedBy, ", "))
				}
				s.WriteString(HelpStyle.Render(desc))
			}

			s.WriteString("\n")
		}
	}

	if m.message != "" {
		s.WriteString("\n")
		switch m.messageType {
		case MessageSuccess:
			s.WriteString(AppliedStyle.Render("✓ " + m.message))
		case MessageWarning:
			s.WriteString(PendingStyle.Render("⚠ " + m.message))
		case MessageError:
			s.WriteString(ErrorStyle.Render("✗ " + m.message))
		default:
			s.WriteString(m.message)
		}
		s.WriteString("\n")
	}

	s.WriteString("\n")
	s.WriteString(HelpStyle.Render(
		"↑/↓: navigate  •  enter/f: fill gap  •  i: ignore gap  •  1: migrations  •  2: gaps  •  3: help  •  r: refresh  •  q: quit",
	))

	return s.String()
}

func (m *Model) renderHelpView() string {
	var s strings.Builder

	s.WriteString(TitleStyle.Render("❓ Help"))
	s.WriteString("\n\n")

	helpContent := `Queen TUI - Interactive Migration Manager

Navigation:
  ↑/k          Move cursor up
  ↓/j          Move cursor down
  g            Jump to top
  G            Jump to bottom

Views:
  1            Migrations view
  2            Gaps detection view
  3/?          Help view

Actions (Migrations View):
  enter        Apply pending migration / Rollback applied migration
  u            Apply migration up to cursor
  d            Rollback migration from cursor

Actions (Gaps View):
  enter/f      Fill the selected gap
  i            Ignore the selected gap (add to .queenignore)

General:
  r            Refresh data
  q/Ctrl+C     Quit

Tips:
  • Applied migrations are shown with ✓ in green
  • Pending migrations are shown with ○ in yellow
  • Gaps are detected automatically and shown in the gaps view
  • Use 'i' to ignore gaps that are intentional
  • Use 'f' to fill application gaps by applying the missing migrations
`

	s.WriteString(InfoBoxStyle.Render(helpContent))
	s.WriteString("\n\n")
	s.WriteString(HelpStyle.Render("Press 1 to return to migrations view  •  Press 2 for gaps view  •  Press q to quit"))

	return s.String()
}
