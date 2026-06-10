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
// Steps: welcome → platformSelect → docsMode → path? → confirm → progress → done
type InstallModel struct {
	// step is the currently active wizard step.
	step Step

	// welcomeButtons is the footer button row on the Welcome screen.
	// Initialized with ["Continue", "Quit"].
	welcomeButtons buttonRow

	// platformSelectButtons is the footer button row on the Platform Selection screen.
	// Initialized with ["Continue", "Back"].
	platformSelectButtons buttonRow

	// docsModeButtons is the footer button row on the Docs Mode screen.
	// Initialized with ["Continue", "Back"].
	docsModeButtons buttonRow

	// pathButtons is the footer button row on the Path Entry screen.
	// Initialized with ["Continue", "Back"].
	pathButtons buttonRow

	// focusZone tracks which UI region owns keyboard focus on the current screen.
	// On screens with a list + button row (e.g. stepPlatformSelect), Tab cycles
	// between focusZoneList and focusZoneButtons.
	focusZone focusZone

	// platforms is the checkbox list for platform selection.
	platforms []platformItem

	// cursor is the focused index in the platforms list.
	cursor int

	// alreadyInstalled is a map of platform ID → true for platforms that have
	// existing agent files detected by checkAlreadyInstalled at startup.
	// Read-only after construction; the slice-4 overwrite screen consults it.
	alreadyInstalled map[string]bool

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

	// overwriteQueue is the ordered list of platform IDs that have existing
	// installs and are awaiting overwrite confirmation (stepOverwriteConfirm).
	overwriteQueue []string

	// overwriteQueueIdx is the index into overwriteQueue of the platform
	// currently being shown in the overwrite confirmation step.
	overwriteQueueIdx int

	// overwriteConsent records the user's per-platform overwrite decision.
	// true = user consented to overwrite; false / missing = skip that platform.
	overwriteConsent map[string]bool

	// progressLines collects Reporter output during stepProgress.
	progressLines []string

	// err is set when the install engine returns an error (displayed in stepDone).
	err error

	// notice is a transient warning shown in stepPlatformSelect when the user
	// tries to continue with zero selections.
	notice string

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

	// Compute already-installed set once at startup using the engine helper.
	// This map is read-only after construction; it drives the already-installed
	// tag in viewPlatformSelect and will drive the overwrite screen in slice 4.
	alreadyInstalled := buildAlreadyInstalledMap(manifest, allPlatforms)

	items := make([]platformItem, 0, len(allPlatforms))
	for _, p := range allPlatforms {
		sel := len(cfg.Platforms) == 0 || cfgPlatSet[p.ID()] // all if no saved prefs
		items = append(items, platformItem{
			id:               p.ID(),
			label:            platformDisplayName(p.ID()),
			detected:         true,
			selected:         sel,
			alreadyInstalled: alreadyInstalled[p.ID()],
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
		step:                  stepWelcome, // wizard starts at the Welcome screen
		welcomeButtons:        buttonRow{labels: []string{"Continue", "Quit"}, focus: 0},
		platformSelectButtons: buttonRow{labels: []string{"Continue", "Back"}, focus: 0},
		docsModeButtons:       buttonRow{labels: []string{"Continue", "Back"}, focus: 0},
		pathButtons:           buttonRow{labels: []string{"Continue", "Back"}, focus: 0},
		focusZone:             focusZoneList,
		platforms:             items,
		cursor:                0,
		alreadyInstalled:      alreadyInstalled,
		mode:                  mode,
		modeCursor:            modeCursor,
		pathInput:             pi,
		cfg:                   cfg,
		cfgExisted:            cfgExisted,
		manifest:              manifest,
		distDir:               distDir,
		allPlatforms:          allPlatforms,
		overwriteConsent:      make(map[string]bool),
		styles:                styles,
		width:                 80,
		height:                24,
	}
}

// ---------------------------------------------------------------------------
// buildAlreadyInstalledMap — startup helper
// ---------------------------------------------------------------------------

// buildAlreadyInstalledMap calls checkAlreadyInstalled once per platform and
// returns a map of platformID → true for every platform that has existing agents.
// It is a pure read from the filesystem; it does NOT modify the manifest or
// any engine state. Call once at newInstallModel time.
func buildAlreadyInstalledMap(manifest DistManifest, allPlatforms []Platform) map[string]bool {
	result := make(map[string]bool, len(allPlatforms))
	for _, plat := range allPlatforms {
		if len(checkAlreadyInstalled(manifest, plat)) > 0 {
			result[plat.ID()] = true
		}
	}
	return result
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
		m.progressLines = msg.progressLines
		m.step = stepDone
		// Do NOT quit here — stay on the done screen so the user can read the
		// summary. Any key (handled in the stepDone case of handleKey) exits.
		return m, nil

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

	case stepWelcome:
		return m.updateWelcome(msg)

	case stepPlatformSelect:
		return m.updatePlatformSelect(msg)

	case stepOverwriteConfirm:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "y", "Y":
			platID := m.overwriteQueue[m.overwriteQueueIdx]
			m.overwriteConsent[platID] = true
			m = m.advanceOverwriteQueue()
		case "n", "N", "enter":
			// Default is skip (no overwrite). The platform remains selected in the
			// checkbox list but will be excluded from the plan via BuildPlan (overwriteConsent
			// is absent for it). Deselect it now to keep the UI consistent.
			platID := m.overwriteQueue[m.overwriteQueueIdx]
			for i := range m.platforms {
				if m.platforms[i].id == platID {
					m.platforms[i].selected = false
					break
				}
			}
			m = m.advanceOverwriteQueue()
		}
		return m, nil

	case stepDocsMode:
		return m.updateDocsMode(msg)

	case stepPath:
		return m.updatePath(msg)

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

// buildOverwriteQueue checks which selected platforms already have agents
// installed. Returns the next step (stepOverwriteConfirm or stepDocsMode),
// the queue of platform IDs needing confirmation, and starting index 0.
// For platforms with no existing install, no confirmation is needed.
func (m InstallModel) buildOverwriteQueue() (Step, []string, int) {
	var queue []string
	for _, p := range m.platforms {
		if !p.selected {
			continue
		}
		// Find the matching Platform from allPlatforms.
		for _, plat := range m.allPlatforms {
			if plat.ID() == p.id {
				existing := checkAlreadyInstalled(m.manifest, plat)
				if len(existing) > 0 {
					queue = append(queue, p.id)
				}
				break
			}
		}
	}
	if len(queue) > 0 {
		return stepOverwriteConfirm, queue, 0
	}
	return stepDocsMode, nil, 0
}

// anySelected reports whether at least one platform checkbox is selected.
func (m InstallModel) anySelected() bool {
	for _, p := range m.platforms {
		if p.selected {
			return true
		}
	}
	return false
}

// advanceOverwriteQueue moves to the next platform in the overwrite queue.
// When the queue is exhausted it transitions to stepDocsMode — unless declining
// overwrites deselected every platform, in which case continuing would hand the
// engine an empty plan (nil Platforms = "all detected", the opposite of what
// the user chose). In that case return to platform selection with a notice.
func (m InstallModel) advanceOverwriteQueue() InstallModel {
	m.overwriteQueueIdx++
	if m.overwriteQueueIdx >= len(m.overwriteQueue) {
		if !m.anySelected() {
			m.step = stepPlatformSelect
			m.notice = "All platforms were declined for overwrite. Reselect platforms or quit (q)."
			return m
		}
		m.step = stepDocsMode
	}
	return m
}

// runInstall builds an InstallPlan from the current model state and
// executes the install engine as a Bubbletea Cmd (runs off the UI goroutine).
// Progress lines are collected inside the Cmd closure and returned via
// installResultMsg so the Bubbletea event loop can apply them to the model.
func (m InstallModel) runInstall() tea.Cmd {
	// Collect selected platform IDs.
	var selectedIDs []string
	for _, p := range m.platforms {
		if p.selected {
			selectedIDs = append(selectedIDs, p.id)
		}
	}

	// Defense in depth: an empty selection must never reach the engine — a nil
	// Platforms list means "install to all detected", the opposite intent.
	if len(selectedIDs) == 0 {
		return func() tea.Msg {
			return installResultMsg{err: fmt.Errorf("no platforms selected — nothing to install")}
		}
	}

	// Determine prev mode (from config, zero-value = no prior install).
	prevMode := DocsMode(m.cfg.Mode)

	// Copy overwrite consent map into the plan so executeInstall can check it.
	overwrite := make(map[string]bool, len(m.overwriteConsent))
	for k, v := range m.overwriteConsent {
		overwrite[k] = v
	}

	plan := InstallPlan{
		Platforms: selectedIDs,
		Mode:      m.mode,
		BasePath:  strings.TrimSpace(m.pathInput.Value()),
		PrevMode:  prevMode,
		Yes:       true, // user confirmed in stepConfirm
		Overwrite: overwrite,
	}

	manifest := m.manifest
	distDir := m.distDir
	allPlatforms := m.allPlatforms

	return func() tea.Msg {
		// Collect progress lines into a local slice owned by this closure.
		// We do not store a pointer to the InstallModel here because tea.Cmd
		// runs on a separate goroutine; writing to the model copy in the event
		// loop's goroutine would cause a data race with no guarantee the lines
		// reach the rendered model. Instead, lines are returned inside
		// installResultMsg and the Update method merges them into the live model.
		var lines []string
		collector := &collectingReporter{lines: &lines}
		err := executeInstall(manifest, plan, distDir, allPlatforms, collector)
		return installResultMsg{err: err, progressLines: lines}
	}
}

// ---------------------------------------------------------------------------
// collectingReporter — collects engine output for display in the done step
// ---------------------------------------------------------------------------

// collectingReporter is a Reporter that appends formatted lines to a local
// []string slice owned by the tea.Cmd closure. The collected lines are
// returned via installResultMsg and merged into the live model by Update,
// ensuring they are displayed in the done step. This avoids writing to a
// detached model copy from the Cmd goroutine.
type collectingReporter struct {
	lines *[]string
}

func (c *collectingReporter) Ok(msg string)     { *c.lines = append(*c.lines, "  ✔ "+msg) }
func (c *collectingReporter) Warn(msg string)   { *c.lines = append(*c.lines, "  ⚠  "+msg) }
func (c *collectingReporter) ErrOut(msg string) { *c.lines = append(*c.lines, "  ✖ "+msg) }
func (c *collectingReporter) Info(msg string)   { *c.lines = append(*c.lines, "  → "+msg) }
func (c *collectingReporter) Dim(msg string)    { *c.lines = append(*c.lines, "  "+msg) }
func (c *collectingReporter) Head(msg string)   { *c.lines = append(*c.lines, "\n  "+msg) }

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (m InstallModel) View() string {
	var sb strings.Builder

	// Welcome screen renders its own full-width logo; inner screens use a
	// compact single-line header prepended here.
	if m.step == stepWelcome {
		switch m.step {
		case stepWelcome:
			m.viewWelcome(&sb)
		}
		return sb.String()
	}

	sb.WriteString(renderCompactHeader(m.styles))

	switch m.step {
	case stepPlatformSelect:
		m.viewPlatformSelect(&sb)
	case stepOverwriteConfirm:
		m.viewOverwriteConfirm(&sb)
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

	// Hint line changes based on active zone.
	if m.focusZone == focusZoneList {
		sb.WriteString(m.styles.Subtitle.Render("  ↑/↓ to move · Space to toggle · Tab for buttons") + "\n\n")
	} else {
		sb.WriteString(m.styles.Subtitle.Render("  ←/→ to move buttons · Enter to activate · Tab back to list") + "\n\n")
	}

	if m.notice != "" {
		sb.WriteString(m.styles.Warning.Render("  ! "+m.notice) + "\n\n")
	}

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

		// Already-installed tag: shown as a dim suffix on the label line.
		// Consult the model-level map (injected at startup via buildAlreadyInstalledMap)
		// rather than the per-item bool so tests can override it directly.
		installedTag := ""
		if m.alreadyInstalled[p.id] {
			installedTag = m.styles.Dim.Render("  (already installed)")
		}

		sb.WriteString(cursor + checkbox + " " + label + installedTag + "\n")
	}

	sb.WriteString("\n")
	sb.WriteString("  " + m.platformSelectButtons.render(m.styles) + "\n")
}

func (m InstallModel) viewOverwriteConfirm(sb *strings.Builder) {
	if m.overwriteQueueIdx >= len(m.overwriteQueue) {
		return
	}
	platID := m.overwriteQueue[m.overwriteQueueIdx]
	platLabel := platformDisplayName(platID)

	// Find which agents are already installed for this platform.
	var existing []string
	for _, plat := range m.allPlatforms {
		if plat.ID() == platID {
			existing = checkAlreadyInstalled(m.manifest, plat)
			break
		}
	}

	sb.WriteString(m.styles.Title.Render("  Overwrite existing installation?") + "\n\n")
	sb.WriteString(m.styles.Subtitle.Render("  "+platLabel+" already has agents installed:") + "\n")
	for _, id := range existing {
		sb.WriteString(m.styles.Dim.Render("    - "+id) + "\n")
	}
	sb.WriteString("\n")
	total := len(m.overwriteQueue)
	sb.WriteString(m.styles.Dim.Render(fmt.Sprintf("  Platform %d of %d", m.overwriteQueueIdx+1, total)) + "\n\n")
	sb.WriteString(m.styles.Confirm.Render("  Overwrite " + platLabel + "? (y/N) "))
}

func (m InstallModel) viewDocsMode(sb *strings.Builder) {
	sb.WriteString(m.styles.Title.Render("  Documentation mode") + "\n")

	// Hint line changes based on active focus zone.
	if m.focusZone == focusZoneList {
		sb.WriteString(m.styles.Subtitle.Render("  ↑/↓ to move · Tab for buttons") + "\n\n")
	} else {
		sb.WriteString(m.styles.Subtitle.Render("  ←/→ to move buttons · Enter to activate · Tab back to list") + "\n\n")
	}

	// Mode-switch notice: shown when there is a prior config and the currently
	// selected mode differs from the saved mode. Relocated here from stepConfirm
	// (slice 3 — the stepConfirm notice is kept on confirm; this is an additional
	// early signal on the docs-mode screen itself).
	selectedMode := ModeVault
	if m.modeCursor == 1 {
		selectedMode = ModeInProject
	}
	if m.cfgExisted && m.cfg.Mode != "" && m.cfg.Mode != string(selectedMode) {
		sb.WriteString(m.styles.Notice.Render("  ! Mode change detected.") + "\n")
		sb.WriteString(m.styles.Dim.Render("    Existing documentation files are not automatically migrated.") + "\n\n")
	}

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

	sb.WriteString("\n")
	sb.WriteString("  " + m.docsModeButtons.render(m.styles) + "\n")
}

func (m InstallModel) viewPath(sb *strings.Builder) {
	sb.WriteString(m.styles.Title.Render("  Vault base path") + "\n")
	sb.WriteString(m.styles.Subtitle.Render("  Enter the root folder for all your doc-agent-ai documentation") + "\n\n")
	sb.WriteString("  " + m.pathInput.View() + "\n\n")
	sb.WriteString("  " + m.pathButtons.render(m.styles) + "\n")
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
		sb.WriteString(m.styles.Dim.Render("  Run with --help for headless flag usage.") + "\n\n")
		sb.WriteString(m.styles.Dim.Render("  Press any key to exit.") + "\n")
		return
	}

	sb.WriteString(m.styles.Ok.Render("  ✔ Install complete!") + "\n\n")

	// Show progress log.
	for _, line := range m.progressLines {
		sb.WriteString(line + "\n")
	}

	sb.WriteString("\n")
	sb.WriteString(m.styles.Dim.Render("  Restart your AI tool if it is currently running.") + "\n\n")
	sb.WriteString(m.styles.Dim.Render("  Press any key to exit.") + "\n")
}

// ---------------------------------------------------------------------------
// Welcome screen — view and update
// ---------------------------------------------------------------------------

// viewWelcome renders the Welcome screen: the Zeen block-art lockup (icon Z +
// "een" wordmark composed horizontally), a concise agent description, and the
// [Continue][Quit] footer button row.
func (m InstallModel) viewWelcome(sb *strings.Builder) {
	sb.WriteString("\n")
	// Zeen lockup: icon Z mark (12 rows) + "een" wordmark (4 rows, vertically centered).
	sb.WriteString(renderZeenLockup(m.styles))
	sb.WriteString("\n")

	// Short agent description.
	sb.WriteString(m.styles.Subtitle.Render("  AI-powered documentation agent for your project.") + "\n")
	sb.WriteString(m.styles.Dim.Render("  Installs context-aware skills and roles into your AI tool.") + "\n")
	sb.WriteString("\n")

	// Footer button row.
	sb.WriteString("  " + m.welcomeButtons.render(m.styles) + "\n")
}

// updateWelcome handles key input on the Welcome screen.
// Tab/Shift+Tab/Left/Right move button focus; Enter activates:
//   - "Continue" → advance to stepPlatformSelect
//   - "Quit"     → tea.Quit (exit 0)
func (m InstallModel) updateWelcome(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "tab", "shift+tab", "left", "right":
		activated, _ := m.welcomeButtons.handle(msg.String())
		_ = activated // focus-move only
		return m, nil
	case "enter":
		activated, ok := m.welcomeButtons.handle("enter")
		if !ok {
			return m, nil
		}
		switch activated {
		case "Continue":
			m.step = stepPlatformSelect
			return m, nil
		case "Quit":
			return m, tea.Quit
		}
	}
	return m, nil
}

// updatePlatformSelect handles key input on the Platform Selection screen.
//
// focusZone determines key routing:
//   - focusZoneList: up/down/j/k move cursor; Space toggles checkbox; Tab → focusZoneButtons.
//   - focusZoneButtons: Tab/left/right move button focus; Enter activates;
//     up/down pass-through to the list (ergonomic: user may expect list navigation from buttons).
//
// Enter with focusZoneButtons:
//   - "Continue" → nextStep (stepDocsMode); blocked if zero selected (shows notice).
//   - "Back"     → prevStep (stepWelcome); state is preserved.
func (m InstallModel) updatePlatformSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "tab":
		// Cycle focus between list and button row.
		if m.focusZone == focusZoneList {
			m.focusZone = focusZoneButtons
		} else {
			m.focusZone = focusZoneList
		}
		return m, nil

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case "down", "j":
		if m.cursor < len(m.platforms)-1 {
			m.cursor++
		}
		return m, nil

	case " ":
		// Space toggles the checkbox only when the list zone is focused.
		if m.focusZone == focusZoneList && len(m.platforms) > 0 {
			m.platforms[m.cursor].selected = !m.platforms[m.cursor].selected
			m.notice = ""
		}
		return m, nil

	case "left", "shift+tab":
		if m.focusZone == focusZoneButtons {
			m.platformSelectButtons = m.platformSelectButtons.move(-1)
		}
		return m, nil

	case "right":
		if m.focusZone == focusZoneButtons {
			m.platformSelectButtons = m.platformSelectButtons.move(1)
		}
		return m, nil

	case "enter":
		if m.focusZone == focusZoneButtons {
			// Activate the focused button.
			activated := m.platformSelectButtons.focused()
			switch activated {
			case "Continue":
				if !m.anySelected() {
					m.notice = "Select at least one platform to continue."
					return m, nil
				}
				m.notice = ""
				m.step = nextStep(stepPlatformSelect)
				// Reset button focus and zone for the next screen.
				m.platformSelectButtons.focus = 0
				m.focusZone = focusZoneList
				return m, nil
			case "Back":
				m.step = prevStep(stepPlatformSelect)
				m.platformSelectButtons.focus = 0
				m.focusZone = focusZoneList
				return m, nil
			}
		} else {
			// Enter in list zone: same as Continue button shortcut.
			if m.anySelected() {
				m.notice = ""
				m.step = nextStep(stepPlatformSelect)
				m.focusZone = focusZoneList
				return m, nil
			}
		}
		return m, nil
	}

	return m, nil
}

// updateDocsMode handles key input on the Documentation Mode screen.
//
// focusZone determines key routing:
//   - focusZoneList: up/down/j/k move modeCursor; Tab → focusZoneButtons.
//   - focusZoneButtons: Tab/left/right move button focus; Enter activates:
//     Continue (sets mode from modeCursor, advances to stepPath or stepConfirm),
//     Back (prevStep = stepPlatformSelect).
//
// The mode-switch notice (non-migration warning) is rendered in viewDocsMode
// when cfgExisted=true and the selected mode differs from cfg.Mode.
func (m InstallModel) updateDocsMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "tab":
		// Cycle focus between radio list and button row.
		if m.focusZone == focusZoneList {
			m.focusZone = focusZoneButtons
		} else {
			m.focusZone = focusZoneList
		}
		return m, nil

	case "up", "k":
		if m.modeCursor > 0 {
			m.modeCursor--
		}
		return m, nil

	case "down", "j":
		if m.modeCursor < 1 {
			m.modeCursor++
		}
		return m, nil

	case "left", "shift+tab":
		if m.focusZone == focusZoneButtons {
			m.docsModeButtons = m.docsModeButtons.move(-1)
		}
		return m, nil

	case "right":
		if m.focusZone == focusZoneButtons {
			m.docsModeButtons = m.docsModeButtons.move(1)
		}
		return m, nil

	case "enter":
		if m.focusZone == focusZoneButtons {
			activated := m.docsModeButtons.focused()
			switch activated {
			case "Continue":
				// Commit the mode from the current cursor position.
				if m.modeCursor == 0 {
					m.mode = ModeVault
				} else {
					m.mode = ModeInProject
				}
				// Reset button focus and zone for the next screen.
				m.docsModeButtons.focus = 0
				m.pathButtons.focus = 0
				m.focusZone = focusZoneList
				if m.mode == ModeVault {
					m.pathInput.Focus()
					m.step = stepPath
				} else {
					m.step = stepConfirm
				}
				return m, nil
			case "Back":
				m.docsModeButtons.focus = 0
				m.focusZone = focusZoneList
				m.step = prevStep(stepDocsMode)
				return m, nil
			}
		} else {
			// Enter in list zone: treat as Continue shortcut.
			if m.modeCursor == 0 {
				m.mode = ModeVault
			} else {
				m.mode = ModeInProject
			}
			m.focusZone = focusZoneList
			if m.mode == ModeVault {
				m.pathInput.Focus()
				m.step = stepPath
			} else {
				m.step = stepConfirm
			}
			return m, nil
		}
	}

	return m, nil
}

// updatePath handles key input on the Vault Path Entry screen.
//
// focusZone determines key routing:
//   - focusZoneList: characters are forwarded to pathInput (so any path
//     character, including 'b', types normally); Tab → focusZoneButtons;
//     Esc is a Back shortcut (navigates to stepDocsMode).
//   - focusZoneButtons: Tab/left/right move button focus; Enter activates:
//     Continue (if path is non-empty, advances to stepConfirm),
//     Back (prevStep = stepDocsMode).
//
// Back uses Esc (not a letter) because this screen is a free-text input —
// an alphabetic Back shortcut would make paths containing that letter
// untypeable (e.g. C:\Users\bob\docs).
func (m InstallModel) updatePath(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "tab":
		if m.focusZone == focusZoneList {
			m.focusZone = focusZoneButtons
			m.pathInput.Blur()
		} else {
			m.focusZone = focusZoneList
			m.pathInput.Focus()
		}
		return m, nil

	case "esc":
		// Back shortcut (non-alphabetic so it never collides with path typing).
		m.pathButtons.focus = 0
		m.focusZone = focusZoneList
		m.pathInput.Focus()
		m.step = prevStep(stepPath)
		return m, nil

	case "enter":
		if m.focusZone == focusZoneButtons {
			activated := m.pathButtons.focused()
			switch activated {
			case "Continue":
				path := strings.TrimSpace(m.pathInput.Value())
				if path != "" {
					m.pathButtons.focus = 0
					m.focusZone = focusZoneList
					m.step = stepConfirm
				}
				return m, nil
			case "Back":
				m.pathButtons.focus = 0
				m.focusZone = focusZoneList
				m.pathInput.Focus()
				m.step = prevStep(stepPath)
				return m, nil
			}
		} else {
			// Enter in list zone: treat as Continue shortcut.
			path := strings.TrimSpace(m.pathInput.Value())
			if path != "" {
				m.focusZone = focusZoneList
				m.step = stepConfirm
			}
			return m, nil
		}

	case "left", "shift+tab":
		if m.focusZone == focusZoneButtons {
			m.pathButtons = m.pathButtons.move(-1)
		}
		return m, nil

	case "right":
		if m.focusZone == focusZoneButtons {
			m.pathButtons = m.pathButtons.move(1)
		}
		return m, nil
	}

	// Forward to textinput when in list zone.
	if m.focusZone == focusZoneList {
		var cmd tea.Cmd
		m.pathInput, cmd = m.pathInput.Update(msg)
		return m, cmd
	}

	return m, nil
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

	overwrite := make(map[string]bool, len(m.overwriteConsent))
	for k, v := range m.overwriteConsent {
		overwrite[k] = v
	}

	return InstallPlan{
		Platforms: selectedIDs,
		Mode:      m.mode,
		BasePath:  strings.TrimSpace(m.pathInput.Value()),
		PrevMode:  DocsMode(m.cfg.Mode),
		Yes:       true,
		Overwrite: overwrite,
	}
}
