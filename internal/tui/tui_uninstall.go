package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	installpkg "github.com/zeshone/doc-agent-ai/internal/install"
)

// ---------------------------------------------------------------------------
// Uninstall wizard step definitions
// ---------------------------------------------------------------------------

// uninstallStep represents the current active step in the uninstall wizard.
type uninstallStep int

const (
	// uninstallStepConfirm shows what will be removed and asks for confirmation.
	uninstallStepConfirm uninstallStep = iota

	// uninstallStepProgress shows removal output while the engine runs.
	uninstallStepProgress

	// uninstallStepDone shows the completion summary.
	uninstallStepDone
)

// ---------------------------------------------------------------------------
// Uninstall result message
// ---------------------------------------------------------------------------

// uninstallResultMsg is sent on the Bubbletea bus when the uninstall engine
// finishes. The TUI Update method uses it to transition to uninstallStepDone.
// progressLines carries the output collected by the engine closure — the model
// copy the closure captured is dead by then, so lines must travel in the msg.
type uninstallResultMsg struct {
	err           error
	progressLines []string
}

// ---------------------------------------------------------------------------
// UninstallModel — Bubbletea model for the uninstall wizard
// ---------------------------------------------------------------------------

// UninstallModel is the Bubbletea model for the uninstall wizard.
// Steps: confirm → progress → done
type UninstallModel struct {
	// step is the currently active wizard step.
	step uninstallStep

	// installed is the list of platforms that have doc-agent-ai artifacts.
	installed []installpkg.InstalledDetails

	// manifest is the rendered install manifest for uninstall detection/removal.
	manifest installpkg.DistManifest

	// progressLines collects output during uninstallStepProgress.
	progressLines []string

	// err holds any error returned by the uninstall engine.
	err error

	// styles holds the lipgloss styles for this session.
	styles Styles

	// width / height are the terminal dimensions.
	width  int
	height int

	// quitting is set to true immediately before the model returns tea.Quit,
	// so RootModel can detect the exit request and return to Home instead of
	// propagating the quit to the top-level Bubbletea program. This preserves
	// standalone behavior: when UninstallModel is run directly via RunUninstallTUI,
	// the tea.Quit is never intercepted so the program exits normally.
	quitting bool
}

// newUninstallModel constructs an UninstallModel ready for use.
func newUninstallModel(installed []installpkg.InstalledDetails, manifest installpkg.DistManifest, styles Styles) UninstallModel {
	return UninstallModel{
		step:      uninstallStepConfirm,
		installed: installed,
		manifest:  manifest,
		styles:    styles,
		width:     80,
		height:    24,
	}
}

// ---------------------------------------------------------------------------
// tea.Model interface
// ---------------------------------------------------------------------------

func (m UninstallModel) Init() tea.Cmd {
	return nil
}

func (m UninstallModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case uninstallResultMsg:
		m.err = msg.err
		m.progressLines = append(m.progressLines, msg.progressLines...)
		m.step = uninstallStepDone
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m UninstallModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.step {
	case uninstallStepConfirm:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "y", "Y":
			m.step = uninstallStepProgress
			return m, m.runUninstall()
		case "n", "N":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			// Default = no (destructive action requires explicit y).
			m.quitting = true
			return m, tea.Quit
		}

	case uninstallStepProgress:
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}

	case uninstallStepDone:
		m.quitting = true
		return m, tea.Quit
	}

	return m, nil
}

// runUninstall drives the existing uninstallPlatform engine for each detected
// installation. Output is accumulated locally and returned in the result msg —
// never appended through the model, whose copy is dead once the Cmd runs.
func (m UninstallModel) runUninstall() tea.Cmd {
	installed := m.installed
	manifest := m.manifest

	return func() tea.Msg {
		var lines []string
		for _, details := range installed {
			lines = append(lines, "  Removing from "+details.Platform.ID()+"...")
			installpkg.UninstallPlatform(details, manifest)
		}
		return uninstallResultMsg{err: nil, progressLines: lines}
	}
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (m UninstallModel) View() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(renderCompactHeader(m.styles))
	sb.WriteString("\n")

	switch m.step {
	case uninstallStepConfirm:
		m.viewConfirm(&sb)
	case uninstallStepProgress:
		m.viewProgress(&sb)
	case uninstallStepDone:
		m.viewDone(&sb)
	}

	return sb.String()
}

func (m UninstallModel) viewConfirm(sb *strings.Builder) {
	sb.WriteString(m.styles.Title.Render("  The following will be removed:") + "\n")
	sb.WriteString(m.styles.Dim.Render("  ─────────────────────────────────") + "\n\n")

	for _, details := range m.installed {
		sb.WriteString(m.styles.Subtitle.Render("  "+details.Platform.ID()+":") + "\n")

		if len(details.Skills) > 0 {
			sb.WriteString(m.styles.Dim.Render("    → Skills: "+strings.Join(details.Skills, ", ")) + "\n")
		}
		if len(details.Prompts) > 0 {
			sb.WriteString(m.styles.Dim.Render("    → Prompts: prompts/doc/") + "\n")
		}
		if len(details.Commands) > 0 {
			cmdNames := make([]string, len(details.Commands))
			for i, id := range details.Commands {
				cmdNames[i] = "/" + id
			}
			sb.WriteString(m.styles.Dim.Render("    → Commands: "+strings.Join(cmdNames, ", ")) + "\n")
		}
		if len(details.Agents) > 0 {
			sb.WriteString(m.styles.Dim.Render("    → Agents: "+strings.Join(details.Agents, ", ")) + "\n")
		}
		if details.Registry {
			sb.WriteString(m.styles.Dim.Render("    → Registry: .atl/skill-registry.md") + "\n")
		}
	}

	sb.WriteString("\n")
	sb.WriteString(m.styles.Warning.Render("  ! Your documentation files are NOT affected.") + "\n\n")
	sb.WriteString(m.styles.Confirm.Render("  Uninstall from all detected platforms? (y/N) "))
}

func (m UninstallModel) viewProgress(sb *strings.Builder) {
	sb.WriteString(m.styles.Title.Render("  Uninstalling...") + "\n\n")
	for _, line := range m.progressLines {
		sb.WriteString(line + "\n")
	}
	sb.WriteString("\n")
	sb.WriteString(m.styles.Dim.Render("  Please wait...") + "\n")
}

func (m UninstallModel) viewDone(sb *strings.Builder) {
	if m.err != nil {
		sb.WriteString(m.styles.ErrStyle.Render("  ✖ Uninstall failed: "+m.err.Error()) + "\n")
		return
	}
	for _, line := range m.progressLines {
		sb.WriteString(m.styles.Dim.Render(line) + "\n")
	}
	if len(m.progressLines) > 0 {
		sb.WriteString("\n")
	}
	sb.WriteString(m.styles.Ok.Render("  ✔ Uninstall complete!") + "\n\n")
	sb.WriteString(m.styles.Dim.Render("  Restart your AI tool if it is currently running.") + "\n")
	sb.WriteString("\n")
	sb.WriteString(m.styles.Dim.Render("  Press any key to return to the menu.") + "\n")
}
