// Package tui provides a terminal UI for Queen migrations.
package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/honeynil/queen"
)

type ViewMode int

const (
	ViewMigrations ViewMode = iota
	ViewGaps
	ViewHelp
)

type Model struct {
	queen       *queen.Queen
	ctx         context.Context
	migrations  []queen.MigrationStatus
	gaps        []queen.Gap
	cursor      int
	viewMode    ViewMode
	message     string
	messageType MessageType
	loading     bool
	width       int
	height      int
	err         error
	quitting    bool
}

type MessageType int

const (
	MessageInfo MessageType = iota
	MessageSuccess
	MessageWarning
	MessageError
)

func NewModel(q *queen.Queen, ctx context.Context) *Model {
	return &Model{
		queen:    q,
		ctx:      ctx,
		cursor:   0,
		viewMode: ViewMigrations,
		loading:  true,
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadMigrations(),
		m.loadGaps(),
	)
}

func (m *Model) loadMigrations() tea.Cmd {
	return func() tea.Msg {
		statuses, err := m.queen.Status(m.ctx)
		if err != nil {
			return errMsg{err}
		}
		return migrationsLoadedMsg{statuses}
	}
}

func (m *Model) loadGaps() tea.Cmd {
	return func() tea.Msg {
		gaps, err := m.queen.DetectGaps(m.ctx)
		if err != nil {
			return errMsg{err}
		}
		return gapsLoadedMsg{gaps}
	}
}

type migrationsLoadedMsg struct {
	migrations []queen.MigrationStatus
}

type gapsLoadedMsg struct {
	gaps []queen.Gap
}

type errMsg struct {
	err error
}

type operationCompleteMsg struct {
	message     string
	messageType MessageType
}

// Update handles messages and updates the model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case migrationsLoadedMsg:
		m.migrations = msg.migrations
		m.loading = false
		return m, nil

	case gapsLoadedMsg:
		m.gaps = msg.gaps
		return m, nil

	case errMsg:
		m.err = msg.err
		m.message = fmt.Sprintf("Error: %v", msg.err)
		m.messageType = MessageError
		m.loading = false
		return m, nil

	case operationCompleteMsg:
		m.message = msg.message
		m.messageType = msg.messageType
		m.loading = false
		// Reload data after operation
		return m, tea.Batch(m.loadMigrations(), m.loadGaps())
	}

	return m, nil
}

// handleKeyPress handles keyboard input.
func (m *Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.loading {
		return m, nil
	}

	switch msg.String() {
	case "ctrl+c", "q":
		m.quitting = true
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		maxCursor := 0
		switch m.viewMode {
		case ViewMigrations:
			maxCursor = len(m.migrations) - 1
		case ViewGaps:
			maxCursor = len(m.gaps) - 1
		}
		if m.cursor < maxCursor {
			m.cursor++
		}

	case "g":
		m.cursor = 0

	case "G":
		switch m.viewMode {
		case ViewMigrations:
			if len(m.migrations) > 0 {
				m.cursor = len(m.migrations) - 1
			}
		case ViewGaps:
			if len(m.gaps) > 0 {
				m.cursor = len(m.gaps) - 1
			}
		}

	case "1":
		m.viewMode = ViewMigrations
		m.cursor = 0
		m.message = ""

	case "2":
		m.viewMode = ViewGaps
		m.cursor = 0
		m.message = ""

	case "3", "?":
		m.viewMode = ViewHelp
		m.cursor = 0
		m.message = ""

	case "r":
		m.loading = true
		m.message = "Reloading..."
		m.messageType = MessageInfo
		return m, tea.Batch(m.loadMigrations(), m.loadGaps())

	case "enter":
		return m.handleAction()

	case "u":
		if m.viewMode == ViewMigrations {
			return m.applyMigration()
		}

	case "d":
		if m.viewMode == ViewMigrations {
			return m.rollbackMigration()
		}

	case "f":
		if m.viewMode == ViewGaps {
			return m.fillGap()
		}

	case "i":
		if m.viewMode == ViewGaps {
			return m.ignoreGap()
		}
	}

	return m, nil
}

// handleAction handles enter key press based on context.
func (m *Model) handleAction() (tea.Model, tea.Cmd) {
	switch m.viewMode {
	case ViewMigrations:
		if len(m.migrations) == 0 || m.cursor >= len(m.migrations) {
			return m, nil
		}
		migration := m.migrations[m.cursor]
		if migration.Status == queen.StatusPending {
			return m.applyMigration()
		} else {
			return m.rollbackMigration()
		}

	case ViewGaps:
		return m.fillGap()
	}

	return m, nil
}

// applyMigration applies the selected migration.
func (m *Model) applyMigration() (tea.Model, tea.Cmd) {
	if len(m.migrations) == 0 || m.cursor >= len(m.migrations) {
		return m, nil
	}

	migration := m.migrations[m.cursor]
	if migration.Status != queen.StatusPending {
		m.message = "Migration already applied"
		m.messageType = MessageWarning
		return m, nil
	}

	m.loading = true
	return m, func() tea.Msg {
		// Count steps to this migration
		steps := 0
		for i := 0; i <= m.cursor; i++ {
			if m.migrations[i].Status == queen.StatusPending {
				steps++
			}
		}

		if err := m.queen.UpSteps(m.ctx, steps); err != nil {
			return operationCompleteMsg{
				message:     fmt.Sprintf("Failed to apply migration: %v", err),
				messageType: MessageError,
			}
		}

		return operationCompleteMsg{
			message:     fmt.Sprintf("Applied migration %s", migration.Version),
			messageType: MessageSuccess,
		}
	}
}

// rollbackMigration rolls back the selected migration.
func (m *Model) rollbackMigration() (tea.Model, tea.Cmd) {
	if len(m.migrations) == 0 || m.cursor >= len(m.migrations) {
		return m, nil
	}

	migration := m.migrations[m.cursor]
	if migration.Status != queen.StatusApplied {
		m.message = "Migration not applied yet"
		m.messageType = MessageWarning
		return m, nil
	}

	m.loading = true
	return m, func() tea.Msg {
		// Count steps from this migration
		steps := 0
		for i := m.cursor; i < len(m.migrations); i++ {
			if m.migrations[i].Status == queen.StatusApplied {
				steps++
			}
		}

		if err := m.queen.Down(m.ctx, steps); err != nil {
			return operationCompleteMsg{
				message:     fmt.Sprintf("Failed to rollback migration: %v", err),
				messageType: MessageError,
			}
		}

		return operationCompleteMsg{
			message:     fmt.Sprintf("Rolled back %d migration(s)", steps),
			messageType: MessageSuccess,
		}
	}
}

// fillGap fills the selected gap.
func (m *Model) fillGap() (tea.Model, tea.Cmd) {
	if len(m.gaps) == 0 || m.cursor >= len(m.gaps) {
		return m, nil
	}

	gap := m.gaps[m.cursor]
	m.loading = true

	return m, func() tea.Msg {
		// Get migration statuses
		statuses, err := m.queen.Status(m.ctx)
		if err != nil {
			return operationCompleteMsg{
				message:     fmt.Sprintf("Failed to get status: %v", err),
				messageType: MessageError,
			}
		}

		targetIndex := -1
		for i, s := range statuses {
			if s.Version == gap.Version {
				targetIndex = i
				break
			}
		}

		if targetIndex == -1 {
			return operationCompleteMsg{
				message:     fmt.Sprintf("Migration %s not found", gap.Version),
				messageType: MessageError,
			}
		}

		// Count steps to apply
		stepsToApply := 0
		for i := 0; i <= targetIndex; i++ {
			if statuses[i].Status == queen.StatusPending {
				stepsToApply++
			}
		}

		if err := m.queen.UpSteps(m.ctx, stepsToApply); err != nil {
			return operationCompleteMsg{
				message:     fmt.Sprintf("Failed to fill gap: %v", err),
				messageType: MessageError,
			}
		}

		return operationCompleteMsg{
			message:     fmt.Sprintf("Filled gap: %s", gap.Version),
			messageType: MessageSuccess,
		}
	}
}

// ignoreGap ignores the selected gap.
func (m *Model) ignoreGap() (tea.Model, tea.Cmd) {
	if len(m.gaps) == 0 || m.cursor >= len(m.gaps) {
		return m, nil
	}

	gap := m.gaps[m.cursor]
	m.loading = true

	return m, func() tea.Msg {
		qi, err := queen.LoadQueenIgnore()
		if err != nil {
			// Create new one if doesn't exist
			qi = &queen.QueenIgnore{}
		}

		if err := qi.AddIgnore(gap.Version, gap.Description, "tui"); err != nil {
			return operationCompleteMsg{
				message:     fmt.Sprintf("Failed to ignore gap: %v", err),
				messageType: MessageError,
			}
		}

		return operationCompleteMsg{
			message:     fmt.Sprintf("Ignored gap: %s", gap.Version),
			messageType: MessageSuccess,
		}
	}
}

// View renders the UI.
func (m *Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	switch m.viewMode {
	case ViewMigrations:
		return m.renderMigrationsView()
	case ViewGaps:
		return m.renderGapsView()
	case ViewHelp:
		return m.renderHelpView()
	}

	return ""
}
