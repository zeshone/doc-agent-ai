package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ---------------------------------------------------------------------------
// Brand palette
// ---------------------------------------------------------------------------

const (
	brandCyan = "#00A5E7" // Zeen blue — primary accent
	brandDark = "#0D1012" // near-black — dark square background
	brandGrey = "#E5E8EA" // light grey — wordmark / subtle text
)

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

	// Plain signals that banner rendering must emit raw glyphs only (no ANSI
	// color codes). Set to true by NoColor(); used by renderBanner.
	Plain bool
}

// NewStyles returns the default production styles with full ANSI colour and
// the Zeen brand palette applied to chrome elements.
func NewStyles() Styles {
	return Styles{
		// Banner: bold cyan wordmark accent.
		Banner: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(brandCyan)),
		// Title: bold + brand cyan heading.
		Title: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(brandCyan)),
		// Subtitle / Dim: muted grey.
		Subtitle: lipgloss.NewStyle().Foreground(lipgloss.Color(brandGrey)),
		// Focused / selected item: inverse chip (dark-on-cyan).
		SelectedItem: lipgloss.NewStyle().
			Foreground(lipgloss.Color(brandDark)).
			Background(lipgloss.Color(brandCyan)),
		NormalItem: lipgloss.NewStyle(),
		CheckedItem: lipgloss.NewStyle().
			Foreground(lipgloss.Color(brandCyan)),
		// Status colors stay conventional for accessibility — not brand palette.
		Ok:       lipgloss.NewStyle().Foreground(lipgloss.Color("2")), // green
		Warning:  lipgloss.NewStyle().Foreground(lipgloss.Color("3")), // yellow
		ErrStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("1")), // red
		Dim:      lipgloss.NewStyle().Foreground(lipgloss.Color(brandGrey)),
		Notice:   lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true), // yellow bold
		Confirm:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3")), // yellow bold
		Plain:    false,
	}
}

// NoColor returns Styles with all colour removed for deterministic golden tests.
// Width/height and structural formatting are preserved so golden snapshots
// reflect real layout without ANSI escape sequences. Plain is set to true so
// renderBanner strips truecolor codes and emits raw block glyphs only.
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
		Plain:        true,
	}
}

// ---------------------------------------------------------------------------
// Banner rendering
// ---------------------------------------------------------------------------

// renderBanner renders a baked cell grid as a multi-line string.
//
// When s.Plain is true the function emits the raw glyph character (▀/▄/space)
// without any Foreground or Background lipgloss wrappers, producing output
// that is byte-identical across calls and free of ANSI escape sequences.
// This is required for deterministic golden tests.
//
// When s.Plain is false each cell is wrapped in a lipgloss truecolor style
// using the FgIdx / BgIdx indices into bannerPalette.
func renderBanner(cells [][]bannerCell, s Styles) string {
	var sb strings.Builder
	for _, row := range cells {
		for _, cell := range row {
			if s.Plain {
				sb.WriteRune(cell.Char)
			} else {
				st := lipgloss.NewStyle()
				if cell.FgIdx >= 0 && int(cell.FgIdx) < len(bannerPalette) {
					st = st.Foreground(lipgloss.Color(bannerPalette[cell.FgIdx]))
				}
				if cell.BgIdx >= 0 && int(cell.BgIdx) < len(bannerPalette) {
					st = st.Background(lipgloss.Color(bannerPalette[cell.BgIdx]))
				}
				sb.WriteString(st.Render(string(cell.Char)))
			}
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// renderCompactHeader returns a one-line branded header for inner screens:
// the compact Z mark followed by the cyan "Zeen" wordmark.
func renderCompactHeader(s Styles) string {
	mark := renderBanner(compactBanner, s)
	// Remove trailing newline from banner before joining with wordmark.
	mark = strings.TrimRight(mark, "\n")

	if s.Plain {
		return mark + "  Zeen\n"
	}
	wordmark := lipgloss.NewStyle().
		Foreground(lipgloss.Color(brandCyan)).
		Bold(true).
		Render("  Zeen")
	return mark + wordmark + "\n"
}

// renderZeenLockup composes the icon Z mark (welcomeBanner, 12 rows) and the
// "een" block-art wordmark (wordmarkEen, 4 rows) side by side into a single
// multi-line string. The "een" grid is vertically centered against the taller
// icon: it is padded with blank lines above and below so that icon-Z + "een"
// reads as one "Zeen" lockup.
//
// In Plain mode (NoColor / s.Plain==true) both halves are rendered as raw block
// glyphs — no ANSI codes — keeping goldens byte-stable.
func renderZeenLockup(s Styles) string {
	iconRows := len(welcomeBanner) // 12
	eenRows := len(wordmarkEen)    // 4
	topPad := (iconRows - eenRows) / 2

	eenWidth := 0
	if len(wordmarkEen) > 0 {
		eenWidth = len(wordmarkEen[0])
	}

	var sb strings.Builder
	for i := 0; i < iconRows; i++ {
		// Render the icon Z cell for this row.
		for _, cell := range welcomeBanner[i] {
			if s.Plain {
				sb.WriteRune(cell.Char)
			} else {
				st := lipgloss.NewStyle()
				if cell.FgIdx >= 0 && int(cell.FgIdx) < len(bannerPalette) {
					st = st.Foreground(lipgloss.Color(bannerPalette[cell.FgIdx]))
				}
				if cell.BgIdx >= 0 && int(cell.BgIdx) < len(bannerPalette) {
					st = st.Background(lipgloss.Color(bannerPalette[cell.BgIdx]))
				}
				sb.WriteString(st.Render(string(cell.Char)))
			}
		}

		// Two-space gap between icon and wordmark.
		sb.WriteString("  ")

		// Render the "een" row (with vertical centering via topPad).
		eenIdx := i - topPad
		if eenIdx >= 0 && eenIdx < eenRows {
			for _, cell := range wordmarkEen[eenIdx] {
				if s.Plain {
					sb.WriteRune(cell.Char)
				} else {
					st := lipgloss.NewStyle()
					if cell.FgIdx >= 0 && int(cell.FgIdx) < len(bannerPalette) {
						st = st.Foreground(lipgloss.Color(bannerPalette[cell.FgIdx]))
					}
					if cell.BgIdx >= 0 && int(cell.BgIdx) < len(bannerPalette) {
						st = st.Background(lipgloss.Color(bannerPalette[cell.BgIdx]))
					}
					sb.WriteString(st.Render(string(cell.Char)))
				}
			}
		} else {
			// Blank padding rows to keep column widths consistent.
			for j := 0; j < eenWidth; j++ {
				sb.WriteByte(' ')
			}
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}
