// Package tui provides a terminal UI for Queen migrations.
package tui

import "github.com/charmbracelet/lipgloss"

var (
	primaryColor   = lipgloss.Color("#7D56F4")
	secondaryColor = lipgloss.Color("#04B575")
	warningColor   = lipgloss.Color("#FFAA00")
	errorColor     = lipgloss.Color("#FF0000")
	grayColor      = lipgloss.Color("#666666")
	whiteColor     = lipgloss.Color("#FFFFFF")

	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	AppliedStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true)

	PendingStyle = lipgloss.NewStyle().
			Foreground(warningColor).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	SelectedStyle = lipgloss.NewStyle().
			Background(primaryColor).
			Foreground(whiteColor).
			Bold(true).
			Padding(0, 1)

	NormalStyle = lipgloss.NewStyle().
			Padding(0, 1)

	HelpStyle = lipgloss.NewStyle().
			Foreground(grayColor).
			MarginTop(1)

	InfoBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2).
			MarginTop(1)

	GapWarningStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(warningColor).
			Padding(0, 1).
			MarginTop(1)

	GapErrorStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(errorColor).
			Padding(0, 1).
			MarginTop(1)
)
