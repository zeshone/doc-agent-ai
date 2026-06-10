package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// ---------------------------------------------------------------------------
// Install wizard Model
// ---------------------------------------------------------------------------

// InstallModel is the Bubbletea model for the install wizard.
// Steps: platformSelect → docsMode → path (vault only) → confirm → progress → done
type InstallModel struct {
	// step is the currently active wizard step.
	step Step

	// platforms is the checkbox list for platform selection.
	platforms []platformItem

	// cursor is the focused index in the platforms list.
	cursor int

	// mode is the chosen documentation mode.
	mode DocsMode

	// modeCursor is the focused index in the mode selection list (0=vault, 1=in-project).
	modeCursor int

	// pathInput is the textinput for vault base path entry.
	pathInput textinput.Model

	// cfg is the AppConfig loaded at startup; used for pre-fill defaults.
	cfg AppConfig

	// cfgExisted is true when a config file was found (enables PrevMode tracking).
	cfgExisted bool

	// manifest is the DistManifest loaded from dist/.
	manifest DistManifest

	// distDir is the path to the dist/ directory.
	distDir string

	// allPlatforms is the full detected platform list (passed to executeInstall).
	allPlatforms []Platform

	// progressLines collects Reporter output during stepProgress.
	progressLines []string

	// err is set when the install engine returns an error (displayed in stepDone).
	err error

	// styles holds the lipgloss styles for this session.
	styles Styles

	// width / height are the terminal dimensions (used for layout).
	width  int
	height int
}

// newInstallModel constructs an InstallModel with defaults pre-filled from cfg.
// allPlatforms is the detected platform list; it is passed through to executeInstall.
func newInstallModel(cfg AppConfig, cfgExisted bool, manifest DistManifest, distDir string, allPlatforms []Platform, styles Styles) InstallModel {
	// Build checkbox list: all detected platforms, pre-select from config if any.
	cfgPlatSet := make(map[string]bool, len(cfg.Platforms))
	for _, id := range cfg.Platforms {
		cfgPlatSet[id] = true
	}

	items := make([]platformItem, 0, len(allPlatforms))
	for _, p := range allPlatforms {
		sel := len(cfg.Platforms) == 0 || cfgPlatSet[p.ID()] // all if no saved prefs
		items = append(items, platformItem{
			id:       p.ID(),
			label:    platformDisplayName(p.ID()),
			detected: true,
			selected: sel,
		})
	}

	// Pre-fill mode from config (default vault if no prior config).
	mode := ModeVault
	if cfgExisted && cfg.Mode == string(ModeInProject) {
		mode = ModeInProject
	}

	modeCursor := 0
	if mode == ModeInProject {
		modeCursor = 1
	}

	// Build path textinput pre-filled from config.
	pi := textinput.New()
	pi.Placeholder = "/home/you/docs/"
	pi.CharLimit = 256
	if cfg.Path != "" {
		pi.SetValue(cfg.Path)
	}
	if mode == ModeVault {
		pi.Focus()
	}

	return InstallModel{
		step:         stepPlatformSelect,
		platforms:    items,
		cursor:       0,
		mode:         mode,
		modeCursor:   modeCursor,
		pathInput:    pi,
		cfg:          cfg,
		cfgExisted:   cfgExisted,
		manifest:     manifest,
		distDir:      distDir,
		allPlatforms: allPlatforms,
		styles:       styles,
		width:        80,
		height:       24,
	}
}

// ---------------------------------------------------------------------------
// tea.Model interface
// ---------------------------------------------------------------------------

func (m InstallModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m InstallModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case installResultMsg:
		m.err = msg.err
		m.step = stepDone
		return m, tea.Quit

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Forward messages to the active text input when on the path step.
	if m.step == stepPath {
		var cmd tea.Cmd
		m.pathInput, cmd = m.pathInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

// handleKey processes keyboard input for each wizard step.
func (m InstallModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.step {

	case stepPlatformSelect:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.platforms)-1 {
				m.cursor++
			}
		case " ":
			if len(m.platforms) > 0 {
				m.platforms[m.cursor].selected = !m.platforms[m.cursor].selected
			}
		case "enter":
			// At least one must be selected.
			anySelected := false
			for _, p := range m.platforms {
				if p.selected {
					anySelected = true
					break
				}
			}
			if anySelected {
				m.step = stepDocsMode
			}
		}

	case stepDocsMode:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.modeCursor > 0 {
				m.modeCursor--
			}
		case "down", "j":
			if m.modeCursor < 1 {
				m.modeCursor++
			}
		case "enter":
			if m.modeCursor == 0 {
				m.mode = ModeVault
			} else {
				m.mode = ModeInProject
			}
			if m.mode == ModeVault {
				m.pathInput.Focus()
				m.step = stepPath
			} else {
				m.step = stepConfirm
			}
		}
		return m, nil

	case stepPath:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			path := strings.TrimSpace(m.pathInput.Value())
			if path != "" {
				m.step = stepConfirm
			}
			return m, nil
		}
		// Forward to textinput.
		var cmd tea.Cmd
		m.pathInput, cmd = m.pathInput.Update(msg)
		return m, cmd

	case stepConfirm:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "y", "Y":
			m.step = stepProgress
			return m, m.runInstall()
		case "n", "N":
			return m, tea.Quit
		case "enter":
			// Default confirm = yes.
			m.step = stepProgress
			return m, m.runInstall()
		}
		return m, nil

	case stepProgress:
		// Wait for installResultMsg; ignore other keys.
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m, nil

	case stepDone:
		return m, tea.Quit
	}

	return m, nil
}

// runInstall builds an InstallPlan from the current model state and
// executes the install engine as a Bubbletea Cmd (runs off the UI goroutine).
func (m *InstallModel) runInstall() tea.Cmd {
	// Collect selected platform IDs.
	var selectedIDs []string
	for _, p := range m.platforms {
		if p.selected {
			selectedIDs = append(selectedIDs, p.id)
		}
	}

	// Determine prev mode (from config, zero-value = no prior install).
	prevMode := DocsMode(m.cfg.Mode)

	plan := InstallPlan{
		Platforms: selectedIDs,
		Mode:      m.mode,
		BasePath:  strings.TrimSpace(m.pathInput.Value()),
		PrevMode:  prevMode,
		Yes:       true, // user confirmed in stepConfirm
	}

	manifest := m.manifest
	distDir := m.distDir
	allPlatforms := m.allPlatforms

	// Capture progress lines via a collecting Reporter.
	collector := &collectingReporter{model: m}

	return func() tea.Msg {
		err := executeInstall(manifest, plan, distDir, allPlatforms, collector)
		return installResultMsg{err: err}
	}
}

// ---------------------------------------------------------------------------
// collectingReporter — routes engine output to the TUI progressLines slice
// ---------------------------------------------------------------------------

// collectingReporter is a Reporter that appends formatted lines to the
// InstallModel.progressLines slice. Because tea.Cmd runs on a separate
// goroutine, we collect into a local buffer and the final installResultMsg
// carries the model update.
//
// Note: this reporter stores a pointer to the InstallModel so it can append
// progressLines during the install. The tea.Cmd closure captures the model
// by pointer; Bubbletea handles the final state handoff via installResultMsg.
type collectingReporter struct {
	model *InstallModel
}

func (c *collectingReporter) Ok(msg string)     { c.model.progressLines = append(c.model.progressLines, "  ✔ "+msg) }
func (c *collectingReporter) Warn(msg string)   { c.model.progressLines = append(c.model.progressLines, "  ⚠  "+msg) }
func (c *collectingReporter) ErrOut(msg string) { c.model.progressLines = append(c.model.progressLines, "  ✖ "+msg) }
func (c *collectingReporter) Info(msg string)   { c.model.progressLines = append(c.model.progressLines, "  → "+msg) }
func (c *collectingReporter) Dim(msg string)    { c.model.progressLines = append(c.model.progressLines, "  "+msg) }
func (c *collectingReporter) Head(msg string)   { c.model.progressLines = append(c.model.progressLines, "\n  "+msg) }

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (m InstallModel) View() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(m.styles.Banner.Render("  doc-agent-ai") + "  " +
		m.styles.Dim.Render("v"+version+" — installer") + "\n\n")

	switch m.step {
	case stepPlatformSelect:
		m.viewPlatformSelect(&sb)
	case stepDocsMode:
		m.viewDocsMode(&sb)
	case stepPath:
		m.viewPath(&sb)
	case stepConfirm:
		m.viewConfirm(&sb)
	case stepProgress:
		m.viewProgress(&sb)
	case stepDone:
		m.viewDone(&sb)
	}

	return sb.String()
}

func (m InstallModel) viewPlatformSelect(sb *strings.Builder) {
	sb.WriteString(m.styles.Title.Render("  Select platforms") + "\n")
	sb.WriteString(m.styles.Subtitle.Render("  Space to toggle · Enter to continue") + "\n\n")

	for i, p := range m.platforms {
		cursor := "  "
		if i == m.cursor {
			cursor = m.styles.SelectedItem.Render("▶ ")
		}

		checkbox := "[ ]"
		if p.selected {
			checkbox = m.styles.CheckedItem.Render("[✔]")
		}

		label := p.label
		if i == m.cursor {
			label = m.styles.SelectedItem.Render(p.label)
		}

		sb.WriteString(cursor + checkbox + " " + label + "\n")
	}
}

func (m InstallModel) viewDocsMode(sb *strings.Builder) {
	sb.WriteString(m.styles.Title.Render("  Documentation mode") + "\n")
	sb.WriteString(m.styles.Subtitle.Render("  ↑/↓ to move · Enter to select") + "\n\n")

	modes := []struct {
		label string
		desc  string
	}{
		{"vault", "Save docs to a central vault path (e.g. ~/docs/)"},
		{"in-project", "Save docs under docs/doc-agent/ in each project"},
	}

	for i, mo := range modes {
		cursor := "  "
		if i == m.modeCursor {
			cursor = m.styles.SelectedItem.Render("▶ ")
		}

		radio := "( )"
		if i == m.modeCursor {
			radio = m.styles.CheckedItem.Render("(●)")
		}

		label := mo.label
		desc := m.styles.Dim.Render("  " + mo.desc)
		if i == m.modeCursor {
			label = m.styles.SelectedItem.Render(mo.label)
		}

		sb.WriteString(cursor + radio + " " + label + "\n")
		sb.WriteString(desc + "\n")
	}
}

func (m InstallModel) viewPath(sb *strings.Builder) {
	sb.WriteString(m.styles.Title.Render("  Vault base path") + "\n")
	sb.WriteString(m.styles.Subtitle.Render("  Enter the root folder for all your doc-agent-ai documentation") + "\n\n")
	sb.WriteString("  " + m.pathInput.View() + "\n\n")
	sb.WriteString(m.styles.Dim.Render("  Enter to continue") + "\n")
}

func (m InstallModel) viewConfirm(sb *strings.Builder) {
	sb.WriteString(m.styles.Title.Render("  Confirm installation") + "\n\n")

	// List selected platforms.
	sb.WriteString(m.styles.Subtitle.Render("  Platforms:") + "\n")
	for _, p := range m.platforms {
		if p.selected {
			sb.WriteString(m.styles.Ok.Render("    ✔ "+p.label) + "\n")
		}
	}

	// Mode.
	sb.WriteString("\n")
	sb.WriteString(m.styles.Subtitle.Render("  Mode: ") + string(m.mode) + "\n")

	// Path (vault only).
	if m.mode == ModeVault {
		sb.WriteString(m.styles.Subtitle.Render("  Path: ") + m.pathInput.Value() + "\n")
	}

	// Mode-switch notice when switching modes.
	if m.cfgExisted && m.cfg.Mode != "" && m.cfg.Mode != string(m.mode) {
		sb.WriteString("\n")
		sb.WriteString(m.styles.Notice.Render("  ! Mode change detected.") + "\n")
		sb.WriteString(m.styles.Dim.Render("    Existing documentation files are not automatically migrated.") + "\n")
	}

	sb.WriteString("\n")
	sb.WriteString(m.styles.Confirm.Render("  Install? (Y/n) "))
}

func (m InstallModel) viewProgress(sb *strings.Builder) {
	sb.WriteString(m.styles.Title.Render("  Installing...") + "\n\n")
	for _, line := range m.progressLines {
		sb.WriteString(line + "\n")
	}
	sb.WriteString("\n")
	sb.WriteString(m.styles.Dim.Render("  Please wait...") + "\n")
}

func (m InstallModel) viewDone(sb *strings.Builder) {
	if m.err != nil {
		sb.WriteString(m.styles.ErrStyle.Render("  ✖ Install failed") + "\n\n")
		sb.WriteString(m.styles.ErrStyle.Render("  "+m.err.Error()) + "\n\n")
		sb.WriteString(m.styles.Dim.Render("  Run with --help for headless flag usage.") + "\n")
		return
	}

	sb.WriteString(m.styles.Ok.Render("  ✔ Install complete!") + "\n\n")

	// Show progress log.
	for _, line := range m.progressLines {
		sb.WriteString(line + "\n")
	}

	sb.WriteString("\n")
	sb.WriteString(m.styles.Dim.Render("  Restart your AI tool if it is currently running.") + "\n")
}

// BuildPlan returns the InstallPlan the user confirmed. Call this after the
// wizard exits to extract the final plan for display or logging.
func (m InstallModel) BuildPlan() InstallPlan {
	var selectedIDs []string
	for _, p := range m.platforms {
		if p.selected {
			selectedIDs = append(selectedIDs, p.id)
		}
	}
	return InstallPlan{
		Platforms: selectedIDs,
		Mode:      m.mode,
		BasePath:  strings.TrimSpace(m.pathInput.Value()),
		PrevMode:  DocsMode(m.cfg.Mode),
		Yes:       true,
	}
}

// Summary returns a one-line summary of the completed install for display.
func (m InstallModel) Summary() string {
	var selectedIDs []string
	for _, p := range m.platforms {
		if p.selected {
			selectedIDs = append(selectedIDs, p.id)
		}
	}
	return fmt.Sprintf("platforms=%s mode=%s", strings.Join(selectedIDs, ","), m.mode)
}
