package main

// ---------------------------------------------------------------------------
// TUI step definitions
// ---------------------------------------------------------------------------

// Step represents the current active step in the install wizard.
type Step int

const (
	// stepPlatformSelect is the first step: checkbox list of available platforms.
	stepPlatformSelect Step = iota

	// stepOverwriteConfirm is the second step: shown when one or more selected
	// platforms already have agents installed. The user is prompted per-platform
	// to confirm overwrite or skip. This step is skipped on fresh installs.
	stepOverwriteConfirm

	// stepDocsMode is the third step: choose vault vs in-project.
	stepDocsMode

	// stepPath is the fourth step: vault base path entry (skipped for in-project).
	stepPath

	// stepConfirm is the fifth step: review choices and confirm.
	stepConfirm

	// stepProgress is the sixth step: running the install engine.
	stepProgress

	// stepDone is the final step: success summary.
	stepDone
)

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
