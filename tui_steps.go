package main

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// ---------------------------------------------------------------------------
// TUI step definitions
// ---------------------------------------------------------------------------

// Step represents the current active step in the install wizard.
// Final chain (ADR-4 / ADR-5 slice 4): Welcome → PlatformSelect → DocsMode →
// Path → Overwrite → Progress → Done.
// stepConfirm and stepOverwriteConfirm have been removed. There are NO raw
// integer comparisons against Step values; all transitions use prevStep/nextStep
// or explicit case labels.
type Step int

const (
	// stepWelcome is the first step: Zeen block-art lockup + agent description +
	// [Continue][Quit] buttons.
	stepWelcome Step = iota

	// stepPlatformSelect is the second step: checkbox list of available platforms.
	stepPlatformSelect

	// stepDocsMode is the third step: choose vault vs in-project.
	stepDocsMode

	// stepPath is the fourth step: vault base path entry (skipped for in-project).
	stepPath

	// stepOverwrite is the fifth step: consolidated overwrite screen. Shown only
	// when at least one selected platform is already installed. The user chooses
	// between "Overwrite all" and "Install only missing". Skipped on fresh installs.
	stepOverwrite

	// stepProgress is the sixth step: running the install engine.
	stepProgress

	// stepDone is the final step: success or nothing-to-do summary.
	stepDone
)

// ---------------------------------------------------------------------------
// Navigation helpers — pure step transition functions
// ---------------------------------------------------------------------------

// nextStep returns the forward transition from the given step.
// Final chain: Welcome → PlatformSelect → DocsMode → Path → Overwrite → Progress → Done.
// Note: stepOverwrite is conditional (skipped when no platforms already installed);
// the routing logic in updateDocsMode/updatePath handles that skip directly.
// This table reflects the physical enum order used for default fallthrough.
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
// Final chain (ADR-4): PlatformSelect → Welcome; DocsMode → PlatformSelect;
// Path → DocsMode; Overwrite → Path (vault default; in-project uses DocsMode
// directly via the mode-aware BACK handler, not this table).
func prevStep(s Step) Step {
	switch s {
	case stepPlatformSelect:
		return stepWelcome
	case stepDocsMode:
		return stepPlatformSelect
	case stepPath:
		return stepDocsMode
	case stepOverwrite:
		return stepPath
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
// Animated progress checklist — slice-5 types (ADR-6)
// ---------------------------------------------------------------------------

// stepDelay is the per-platform tick delay for the animated progress checklist.
// Drives the tea.Tick cadence; named so tests can assert its value and keep
// teatest timeout expectations bounded.
const stepDelay = 150 * time.Millisecond

// checklistState describes the animation state of one platform entry in the
// progress checklist. Transitions: Pending → Installing → Done (or Skipped).
type checklistState int

const (
	// stateChecklist_Pending means this platform has not started yet.
	stateChecklist_Pending checklistState = iota

	// stateChecklist_Installing means the tick cursor is on this platform.
	stateChecklist_Installing

	// stateChecklist_Done means this platform completed installation.
	stateChecklist_Done

	// stateChecklist_Skipped means this platform was already installed and
	// the user chose "install only missing". It never transitions to Done.
	stateChecklist_Skipped
)

// checklistItem is one row in the animated progress checklist.
type checklistItem struct {
	// platformID is the canonical platform ID (e.g. "opencode").
	platformID string

	// state is the current animation state of this row.
	state checklistState
}

// tickMsg is fired by tickCmd on each tea.Tick to advance the checklist cursor.
// It carries no payload — the model advances its own internal cursor.
type tickMsg struct{}

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
// Tab and shift+tab are NOT handled here; button focus is moved by ←/→ only.
//   - right  → move focus +1; returns ("", false)
//   - left   → move focus -1; returns ("", false)
//   - enter  → returns (labels[focus], true)
//   - unknown → returns ("", false)
func (b *buttonRow) handle(key string) (activated string, handled bool) {
	switch strings.ToLower(key) {
	case "right":
		*b = b.move(1)
	case "left":
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
