package main

import "github.com/charmbracelet/lipgloss"

// ---------------------------------------------------------------------------
// Styles — lipgloss style definitions for the installer TUI
// ---------------------------------------------------------------------------

// Styles holds all lipgloss styles used by the TUI. Using a struct instead of
// package-level globals makes it possible to inject a no-color variant in tests
// for deterministic golden output.
type Styles struct {
	// Banner is used for the doc-agent-ai header line.
	Banner lipgloss.Style

	// Title is used for step titles / section headings.
	Title lipgloss.Style

	// Subtitle is a dimmed line used for descriptions under a title.
	Subtitle lipgloss.Style

	// SelectedItem is used to highlight the focused item in a list.
	SelectedItem lipgloss.Style

	// NormalItem is used for list items that are not focused.
	NormalItem lipgloss.Style

	// CheckedItem is used for a checked (selected) list item.
	CheckedItem lipgloss.Style

	// Ok is used for success messages in the progress and done steps.
	Ok lipgloss.Style

	// Warning is used for warning messages in the progress and done steps.
	Warning lipgloss.Style

	// ErrStyle is used for error messages.
	ErrStyle lipgloss.Style

	// Dim is used for de-emphasized informational lines.
	Dim lipgloss.Style

	// Notice is used for the mode-switch notice box.
	Notice lipgloss.Style

	// Confirm is used for the final confirmation prompt line.
	Confirm lipgloss.Style
}

// NewStyles returns the default production styles with full ANSI colour.
func NewStyles() Styles {
	return Styles{
		Banner:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")), // cyan
		Title:        lipgloss.NewStyle().Bold(true),
		Subtitle:     lipgloss.NewStyle().Foreground(lipgloss.Color("8")),            // gray
		SelectedItem: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2")), // green
		NormalItem:   lipgloss.NewStyle(),
		CheckedItem:  lipgloss.NewStyle().Foreground(lipgloss.Color("2")),            // green
		Ok:           lipgloss.NewStyle().Foreground(lipgloss.Color("2")),            // green
		Warning:      lipgloss.NewStyle().Foreground(lipgloss.Color("3")),            // yellow
		ErrStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("1")),            // red
		Dim:          lipgloss.NewStyle().Foreground(lipgloss.Color("8")),            // gray
		Notice:       lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true), // yellow bold
		Confirm:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3")), // yellow bold
	}
}

// NoColor returns Styles with all colour removed for deterministic golden tests.
// Width/height and structural formatting (bold via plain prefix) are preserved so
// golden snapshots reflect real layout without ANSI escape sequences.
func NoColor() Styles {
	plain := lipgloss.NewStyle()
	return Styles{
		Banner:       plain,
		Title:        plain,
		Subtitle:     plain,
		SelectedItem: plain,
		NormalItem:   plain,
		CheckedItem:  plain,
		Ok:           plain,
		Warning:      plain,
		ErrStyle:     plain,
		Dim:          plain,
		Notice:       plain,
		Confirm:      plain,
	}
}
