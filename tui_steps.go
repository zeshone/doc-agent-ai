package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ---------------------------------------------------------------------------
// TUI step definitions
// ---------------------------------------------------------------------------

// Step represents the current active step in the install wizard.
type Step int

const (
	// stepWelcome is the new first step: Zeen block-art lockup + agent description +
	// [Continue][Quit] buttons. Added in slice 1; all subsequent step values
	// shift by 1 — there are NO raw integer comparisons against Step values.
	stepWelcome Step = iota

	// stepPlatformSelect is the second step: checkbox list of available platforms.
	stepPlatformSelect

	// stepOverwriteConfirm is the third step: shown when one or more selected
	// platforms already have agents installed. The user is prompted per-platform
	// to confirm overwrite or skip. This step is skipped on fresh installs.
	stepOverwriteConfirm

	// stepDocsMode is the fourth step: choose vault vs in-project.
	stepDocsMode

	// stepPath is the fifth step: vault base path entry (skipped for in-project).
	stepPath

	// stepConfirm is the sixth step: review choices and confirm.
	stepConfirm

	// stepProgress is the seventh step: running the install engine.
	stepProgress

	// stepDone is the final step: success summary.
	stepDone
)

// ---------------------------------------------------------------------------
// focusZone — tracks which widget owns keyboard focus on a screen
// ---------------------------------------------------------------------------

// focusZone describes which UI region currently owns keyboard focus.
// Screens with multiple interactive regions (e.g. platform-select has a
// checkbox list and a button row) use this to determine how to route keys.
type focusZone int

const (
	// focusZoneList means the content list (checkbox list, radio list, etc.)
	// currently has focus. Up/down/j/k/space operate on the list.
	focusZoneList focusZone = iota

	// focusZoneButtons means the button row currently has focus.
	// Tab/left/right/enter operate on the buttonRow.
	focusZoneButtons
)

// ---------------------------------------------------------------------------
// Navigation helpers — pure step transition functions
// ---------------------------------------------------------------------------

// nextStep returns the forward transition from the given step.
// Welcome → PlatformSelect → DocsMode → … (slices 3-5 extend this table).
func nextStep(s Step) Step {
	switch s {
	case stepWelcome:
		return stepPlatformSelect
	case stepPlatformSelect:
		return stepDocsMode
	default:
		return s + 1
	}
}

// prevStep returns the backward (BACK) transition from the given step.
// BACK from PlatformSelect → Welcome; BACK from DocsMode → PlatformSelect;
// slices 3-5 extend this table as each screen is implemented.
func prevStep(s Step) Step {
	switch s {
	case stepPlatformSelect:
		return stepWelcome
	case stepDocsMode:
		return stepPlatformSelect
	default:
		if s > stepWelcome {
			return s - 1
		}
		return stepWelcome
	}
}

// ---------------------------------------------------------------------------
// Platform checkbox item
// ---------------------------------------------------------------------------

// platformItem represents one platform in the platform-select checkbox list.
type platformItem struct {
	// id is the canonical platform ID (e.g. "opencode").
	id string

	// label is the human-readable display name shown in the TUI.
	label string

	// detected is true if this platform was found on the current system.
	detected bool

	// selected is true if the user has toggled this platform on.
	selected bool

	// alreadyInstalled is true when checkAlreadyInstalled found existing agents
	// for this platform. Used to render the "already installed" tag.
	alreadyInstalled bool
}

// ---------------------------------------------------------------------------
// Install result (collected by the TUI Reporter during stepProgress)
// ---------------------------------------------------------------------------

// installResultMsg is sent on the Bubbletea message bus when executeInstall
// completes (either with success or an error). The TUI Update method receives
// this message and transitions from stepProgress → stepDone.
// progressLines carries all Reporter output collected during the install so
// the event-loop model can display them in the done step.
type installResultMsg struct {
	// err is nil on success, non-nil on failure.
	err error

	// progressLines holds the formatted output lines collected by collectingReporter.
	progressLines []string
}

// ---------------------------------------------------------------------------
// buttonRow — focusable footer button primitive
// ---------------------------------------------------------------------------

// buttonRow is a lightweight focusable button row used as the navigation footer
// on each wizard screen. It is a value type; handle uses a pointer receiver to
// allow mutation via a direct field access on the model.
type buttonRow struct {
	labels []string
	focus  int
}

// move returns a new buttonRow with focus shifted by delta positions (wrapping).
func (b buttonRow) move(delta int) buttonRow {
	if len(b.labels) == 0 {
		return b
	}
	n := len(b.labels)
	b.focus = ((b.focus+delta)%n + n) % n
	return b
}

// focused returns the label of the currently focused button.
func (b buttonRow) focused() string {
	if len(b.labels) == 0 || b.focus < 0 || b.focus >= len(b.labels) {
		return ""
	}
	return b.labels[b.focus]
}

// handle processes a key string and updates focus or activates the focused button.
//   - Tab / right  → move focus +1; returns ("", false)
//   - shift+tab / left → move focus -1; returns ("", false)
//   - enter         → returns (labels[focus], true)
//   - unknown        → returns ("", false)
func (b *buttonRow) handle(key string) (activated string, handled bool) {
	switch strings.ToLower(key) {
	case "tab", "right":
		*b = b.move(1)
	case "shift+tab", "left":
		*b = b.move(-1)
	case "enter":
		return b.focused(), true
	}
	return "", false
}

// render returns a string representation of the button row.
// The focused button is styled as an inverse chip in color mode or wrapped in
// square brackets in plain mode. Unfocused buttons use the Dim style.
func (b buttonRow) render(s Styles) string {
	if len(b.labels) == 0 {
		return ""
	}
	parts := make([]string, 0, len(b.labels))
	for i, label := range b.labels {
		if i == b.focus {
			if s.Plain {
				parts = append(parts, "["+label+"]")
			} else {
				chip := lipgloss.NewStyle().
					Foreground(lipgloss.Color(brandDark)).
					Background(lipgloss.Color(brandCyan)).
					Render(" " + label + " ")
				parts = append(parts, chip)
			}
		} else {
			if s.Plain {
				parts = append(parts, " "+label+" ")
			} else {
				parts = append(parts, s.Dim.Render(" "+label+" "))
			}
		}
	}
	return strings.Join(parts, "  ")
}
