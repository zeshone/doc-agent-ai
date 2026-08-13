package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	configpkg "github.com/zeshone/doc-agent-ai/internal/config"
	installpkg "github.com/zeshone/doc-agent-ai/internal/install"
)

// ---------------------------------------------------------------------------
// Install wizard Model
// ---------------------------------------------------------------------------

// InstallModel is the Bubbletea model for the install wizard.
// Final chain: Welcome → PlatformSelect → DocsMode → Path(vault) → Overwrite(conditional) → Progress → Done.
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

	// overwriteButtons is the footer button row on the Consolidated Overwrite screen.
	// Initialized with ["Install", "Back"].
	overwriteButtons buttonRow

	// platforms is the checkbox list for platform selection.
	platforms []platformItem

	// cursor is the focused index in the platforms list.
	cursor int

	// alreadyInstalled is a map of platform ID → true for platforms that have
	// existing agent files detected by checkAlreadyInstalled at startup.
	// Read-only after construction.
	alreadyInstalled map[string]bool

	// mode is the chosen documentation mode.
	mode configpkg.DocsMode

	// modeCursor is the focused index in the mode selection list (0=vault, 1=in-project).
	modeCursor int

	// pathInput is the textinput for vault base path entry.
	pathInput textinput.Model

	// overwriteChoice is the selected option on the Consolidated Overwrite screen.
	// 0 = "Overwrite all" (set Overwrite[id]=true for all selected platforms).
	// 1 = "Install only missing" (exclude already-installed platforms from plan.Platforms).
	overwriteChoice int

	// cfg is the AppConfig loaded at startup; used for pre-fill defaults.
	cfg configpkg.AppConfig

	// cfgExisted is true when a config file was found (enables PrevMode tracking).
	cfgExisted bool

	// bundle is the fully rendered install content.
	bundle installpkg.Bundle

	// manifest is cached from bundle.Manifest for read-only detection logic.
	manifest installpkg.DistManifest

	// allPlatforms is the full detected platform list (passed to executeInstall).
	allPlatforms []installpkg.Platform

	// installing is set to true when the install engine starts running.
	// While true, BACK navigation is disabled on stepProgress.
	installing bool

	// checklist is the per-platform animated progress list for stepProgress.
	// Seeded from selected platforms when entering stepProgress.
	checklist []checklistItem

	// checklistCursor is the index of the next item to advance in the checklist.
	// A tickMsg increments this cursor, marking each platform Installing → Done.
	checklistCursor int

	// progressBar is the bubbles/progress bar model used for visual pacing.
	// Rendered with ViewAs(pct) for deterministic golden output (no spring frames).
	progressBar progress.Model

	// progressLines collects Reporter output during stepProgress, or the
	// "nothing to install" summary when all selected platforms were already present.
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

	// quitting is set to true immediately before the model returns tea.Quit,
	// so RootModel can detect the exit request and return to Home instead of
	// propagating the quit to the top-level Bubbletea program. This preserves
	// standalone behavior: when InstallModel is run directly via RunInstallTUI,
	// the tea.Quit is never intercepted so the program exits normally.
	quitting bool
}

// newInstallModel constructs an InstallModel with defaults pre-filled from cfg.
// allPlatforms is the detected platform list; it is passed through to executeInstall.
func newInstallModel(cfg configpkg.AppConfig, cfgExisted bool, bundle installpkg.Bundle, allPlatforms []installpkg.Platform, styles Styles) InstallModel {
	manifest := bundle.Manifest
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
			label:            installpkg.PlatformDisplayName(p.ID()),
			selected:         sel,
			alreadyInstalled: alreadyInstalled[p.ID()],
		})
	}

	// Pre-fill mode from config (default vault if no prior config).
	mode := configpkg.ModeVault
	if cfgExisted && cfg.Mode == string(configpkg.ModeInProject) {
		mode = configpkg.ModeInProject
	}

	modeCursor := 0
	if mode == configpkg.ModeInProject {
		modeCursor = 1
	}

	// Build path textinput pre-filled from config.
	pi := textinput.New()
	pi.Placeholder = "/home/you/docs/"
	pi.CharLimit = 256
	if cfg.Path != "" {
		pi.SetValue(cfg.Path)
	}
	if mode == configpkg.ModeVault {
		pi.Focus()
	}

	// Initialize the progress bar with brand-cyan solid fill; no gradient.
	// Width is set to 40 here and can be adjusted by WindowSizeMsg.
	// WithSolidFill ensures NoColor mode doesn't break rendering.
	pb := progress.New(
		progress.WithSolidFill(brandCyan),
		progress.WithWidth(40),
		progress.WithoutPercentage(),
	)

	return InstallModel{
		step:                  stepWelcome, // wizard starts at the Welcome screen
		welcomeButtons:        buttonRow{labels: []string{"Continue", "Quit"}, focus: 0},
		platformSelectButtons: buttonRow{labels: []string{"Continue", "Back"}, focus: 0},
		docsModeButtons:       buttonRow{labels: []string{"Continue", "Back"}, focus: 0},
		pathButtons:           buttonRow{labels: []string{"Continue", "Back"}, focus: 0},
		overwriteButtons:      buttonRow{labels: []string{"Install", "Back"}, focus: 0},
		platforms:             items,
		cursor:                0,
		alreadyInstalled:      alreadyInstalled,
		mode:                  mode,
		modeCursor:            modeCursor,
		pathInput:             pi,
		overwriteChoice:       0,
		cfg:                   cfg,
		cfgExisted:            cfgExisted,
		bundle:                bundle,
		manifest:              manifest,
		allPlatforms:          allPlatforms,
		progressBar:           pb,
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
func buildAlreadyInstalledMap(manifest installpkg.DistManifest, allPlatforms []installpkg.Platform) map[string]bool {
	result := make(map[string]bool, len(allPlatforms))
	for _, plat := range allPlatforms {
		if len(installpkg.CheckAlreadyInstalled(manifest, plat)) > 0 {
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
		m.progressBar.Width = msg.Width - 8 // leave margin; at least 10 wide
		if m.progressBar.Width < 10 {
			m.progressBar.Width = 10
		}
		return m, nil

	case tickMsg:
		if m.step == stepProgress && m.installing {
			return m.processTickMsg()
		}
		return m, nil

	case installResultMsg:
		m.err = msg.err
		m.progressLines = msg.progressLines
		// Mark all non-skipped checklist items as Done on engine completion.
		for i, item := range m.checklist {
			if item.state != stateChecklist_Skipped {
				m.checklist[i].state = stateChecklist_Done
			}
		}
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

	case stepDocsMode:
		return m.updateDocsMode(msg)

	case stepPath:
		return m.updatePath(msg)

	case stepOverwrite:
		return m.updateOverwrite(msg)

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

// anySelected reports whether at least one platform checkbox is selected.
func (m InstallModel) anySelected() bool {
	for _, p := range m.platforms {
		if p.selected {
			return true
		}
	}
	return false
}

// anySelectedInstalled reports whether any of the currently selected platforms
// are in the alreadyInstalled map.
func (m InstallModel) anySelectedInstalled() bool {
	for _, p := range m.platforms {
		if p.selected && m.alreadyInstalled[p.id] {
			return true
		}
	}
	return false
}

// nextStepAfterDocsOrPath returns the next step after docs-mode or path entry,
// skipping stepOverwrite when no selected platform is already installed.
func (m InstallModel) nextStepAfterDocsOrPath() Step {
	if m.anySelectedInstalled() {
		return stepOverwrite
	}
	return stepProgress
}

// runInstall builds an InstallPlan from the current model state and
// executes the install engine as a Bubbletea Cmd (runs off the UI goroutine).
// Progress lines are collected inside the Cmd closure and returned via
// installResultMsg so the Bubbletea event loop can apply them to the model.
//
// Overwrite semantics (ADR-5):
//   - overwrite-all (overwriteChoice=0): Overwrite[id]=true for every selected platform.
//   - install-only-missing (overwriteChoice=1): already-installed platforms are excluded
//     from plan.Platforms. CRITICAL: plan.Platforms is always []string{} (non-nil) when
//     the exclusion leaves nothing, so the engine never receives nil (which means "all").
//     The all-already-installed case bypasses the engine entirely and sends a synthetic
//     installResultMsg directly (handled by the caller before runInstall is called).
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
	prevMode := configpkg.DocsMode(m.cfg.Mode)

	// Build the overwrite map and effective platform list based on overwriteChoice.
	overwrite := make(map[string]bool)
	effectiveIDs := make([]string, 0, len(selectedIDs))

	if m.overwriteChoice == 0 {
		// Overwrite all: set Overwrite[id]=true for all selected.
		for _, id := range selectedIDs {
			overwrite[id] = true
			effectiveIDs = append(effectiveIDs, id)
		}
	} else {
		// Install only missing: exclude already-installed platforms.
		for _, id := range selectedIDs {
			if m.alreadyInstalled[id] {
				// Skip — already present, user chose not to overwrite.
				continue
			}
			effectiveIDs = append(effectiveIDs, id)
		}
		// effectiveIDs is already []string{} (non-nil) if nothing remains
		// because we used make([]string, 0, ...) above. The all-already-present
		// case is handled in updateOverwrite BEFORE calling runInstall, so we
		// should never reach here with a zero effectiveIDs list. Guard anyway.
		if len(effectiveIDs) == 0 {
			return func() tea.Msg {
				return installResultMsg{err: fmt.Errorf("no platforms selected — nothing to install")}
			}
		}
	}

	plan := configpkg.InstallPlan{
		Platforms: effectiveIDs,
		Mode:      m.mode,
		BasePath:  configpkg.ExpandUserPath(m.pathInput.Value()),
		PrevMode:  prevMode,
		Yes:       true, // user confirmed via [Install] button
		Overwrite: overwrite,
	}

	bundle := m.bundle
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
		err := installpkg.ExecuteInstall(bundle, plan, allPlatforms, collector)
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
	case stepDocsMode:
		m.viewDocsMode(&sb)
	case stepPath:
		m.viewPath(&sb)
	case stepOverwrite:
		m.viewOverwrite(&sb)
	case stepProgress:
		m.viewProgress(&sb)
	case stepDone:
		m.viewDone(&sb)
	}

	return sb.String()
}

func (m InstallModel) viewPlatformSelect(sb *strings.Builder) {
	sb.WriteString(m.styles.Title.Render("  Select platforms") + "\n")
	sb.WriteString(m.styles.Subtitle.Render("  ↑/↓ choose · Space toggle · ←/→ buttons · Enter select") + "\n\n")

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

// viewOverwrite renders the consolidated overwrite screen (ADR-5).
// Shown when at least one selected platform is already installed.
// The user picks one of two radio options and activates [Install] or [Back].
func (m InstallModel) viewOverwrite(sb *strings.Builder) {
	// Count and list already-installed platforms among the selected set.
	var installedNames []string
	for _, p := range m.platforms {
		if p.selected && m.alreadyInstalled[p.id] {
			installedNames = append(installedNames, installpkg.PlatformDisplayName(p.id))
		}
	}

	n := len(installedNames)
	if n == 1 {
		sb.WriteString(m.styles.Title.Render(fmt.Sprintf("  1 of your selected platforms already has Zeen installed:")) + "\n")
	} else {
		sb.WriteString(m.styles.Title.Render(fmt.Sprintf("  %d of your selected platforms already have Zeen installed:", n)) + "\n")
	}

	for _, name := range installedNames {
		sb.WriteString(m.styles.Dim.Render("    - "+name) + "\n")
	}
	sb.WriteString("\n")

	// Hint line.
	sb.WriteString(m.styles.Subtitle.Render("  ↑/↓ choose · ←/→ buttons · Enter select") + "\n\n")

	choices := []string{"Overwrite all", "Install only missing"}
	for i, label := range choices {
		cursor := "  "
		if i == m.overwriteChoice {
			cursor = m.styles.SelectedItem.Render("▶ ")
		}
		radio := "( )"
		if i == m.overwriteChoice {
			radio = m.styles.CheckedItem.Render("(●)")
		}
		lbl := label
		if i == m.overwriteChoice {
			lbl = m.styles.SelectedItem.Render(label)
		}
		sb.WriteString(cursor + radio + " " + lbl + "\n")
	}

	sb.WriteString("\n")
	sb.WriteString("  " + m.overwriteButtons.render(m.styles) + "\n")
}

func (m InstallModel) viewDocsMode(sb *strings.Builder) {
	sb.WriteString(m.styles.Title.Render("  Documentation mode") + "\n")

	sb.WriteString(m.styles.Subtitle.Render("  ↑/↓ choose · ←/→ buttons · Enter select") + "\n\n")

	// Mode-switch notice: shown when there is a prior config and the currently
	// selected mode differs from the saved mode. Rendered here (docs-mode screen)
	// as an early signal. stepConfirm has been removed in slice 4.
	selectedMode := configpkg.ModeVault
	if m.modeCursor == 1 {
		selectedMode = configpkg.ModeInProject
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
	sb.WriteString(m.styles.Subtitle.Render("  Enter the root folder for all your doc-agent-ai documentation") + "\n")
	sb.WriteString(m.styles.Dim.Render("  Enter to continue · Esc to go back") + "\n\n")
	sb.WriteString("  " + m.pathInput.View() + "\n\n")
	sb.WriteString("  " + m.pathButtons.render(m.styles) + "\n")
}

func (m InstallModel) viewProgress(sb *strings.Builder) {
	sb.WriteString(m.styles.Title.Render("  Installing Zeen...") + "\n\n")

	// Progress bar: compute percentage from non-skipped done items.
	total := 0
	done := 0
	for _, item := range m.checklist {
		if item.state == stateChecklist_Skipped {
			continue
		}
		total++
		if item.state == stateChecklist_Done {
			done++
		}
	}
	pct := 0.0
	if total > 0 {
		pct = float64(done) / float64(total)
	}
	// ViewAs renders a static bar at the given percentage — deterministic,
	// no spring-animation FrameMsg noise in golden tests.
	sb.WriteString("  " + m.progressBar.ViewAs(pct) + "\n\n")

	// Per-platform checklist.
	for _, item := range m.checklist {
		var marker string
		var label string
		displayName := installpkg.PlatformDisplayName(item.platformID)
		switch item.state {
		case stateChecklist_Pending:
			marker = "  ○ "
			label = m.styles.Dim.Render(displayName)
		case stateChecklist_Installing:
			marker = "  → "
			label = m.styles.Subtitle.Render(displayName)
		case stateChecklist_Done:
			marker = "  ✔ "
			label = m.styles.Ok.Render(displayName)
		case stateChecklist_Skipped:
			marker = "  — "
			label = m.styles.Dim.Render(displayName + " (already installed, skipped)")
		}
		sb.WriteString(marker + label + "\n")
	}

	sb.WriteString("\n")
	sb.WriteString(m.styles.Dim.Render("  Please wait... (press Ctrl+C to abort)") + "\n")
}

func (m InstallModel) viewDone(sb *strings.Builder) {
	if m.err != nil {
		sb.WriteString(m.styles.ErrStyle.Render("  ✖ Install failed") + "\n\n")
		sb.WriteString(m.styles.ErrStyle.Render("  "+m.err.Error()) + "\n\n")
		sb.WriteString(m.styles.Dim.Render("  Run with --help for headless flag usage.") + "\n\n")
		sb.WriteString(m.styles.Dim.Render("  Press any key to return to the menu.") + "\n")
		return
	}

	sb.WriteString(m.styles.Ok.Render("  ✔ Install complete!") + "\n\n")

	// Show progress log.
	for _, line := range m.progressLines {
		sb.WriteString(line + "\n")
	}

	sb.WriteString("\n")
	sb.WriteString(m.styles.Dim.Render("  Restart your AI tool if it is currently running.") + "\n\n")
	sb.WriteString(m.styles.Dim.Render("  Press any key to return to the menu.") + "\n")
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
// ←/→ move button focus; Enter activates:
//   - "Continue" → advance to stepPlatformSelect
//   - "Quit"     → tea.Quit (exit 0)
//
// ↑/↓ are no-ops on this screen (no content list).
// Tab is NOT bound (no focusZone switching).
func (m InstallModel) updateWelcome(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "left", "right":
		m.welcomeButtons.handle(msg.String())
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
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

// updatePlatformSelect handles key input on the Platform Selection screen.
//
// Both navigation axes are always live (no Tab / focusZone switching):
//   - ↑/↓ (k/j) move the platform checkbox cursor.
//   - ←/→ move button-row focus.
//   - Space toggles the focused platform checkbox; always active.
//   - Enter activates the focused button:
//     "Continue" → nextStep (stepDocsMode); blocked if zero selected (shows notice).
//     "Back"     → prevStep (stepWelcome); state is preserved.
//   - Tab is a no-op (ignored).
func (m InstallModel) updatePlatformSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

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
		// Space always toggles the focused checkbox.
		if len(m.platforms) > 0 {
			m.platforms[m.cursor].selected = !m.platforms[m.cursor].selected
			m.notice = ""
		}
		return m, nil

	case "left":
		m.platformSelectButtons = m.platformSelectButtons.move(-1)
		return m, nil

	case "right":
		m.platformSelectButtons = m.platformSelectButtons.move(1)
		return m, nil

	case "enter":
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
			m.platformSelectButtons.focus = 0
			return m, nil
		case "Back":
			m.step = prevStep(stepPlatformSelect)
			m.platformSelectButtons.focus = 0
			return m, nil
		}
		return m, nil
	}

	return m, nil
}

// updateDocsMode handles key input on the Documentation Mode screen.
//
// Both navigation axes are always live (no Tab / focusZone switching):
//   - ↑/↓ (k/j) move modeCursor between vault (0) and in-project (1).
//   - ←/→ move button-row focus.
//   - Enter activates the focused button:
//     "Continue" → commits mode; vault → stepPath; in-project →
//     stepOverwrite (if any selected platform is already installed) or stepProgress.
//     "Back" → prevStep (stepPlatformSelect).
//   - Tab is a no-op (ignored).
//
// The mode-switch notice (non-migration warning) is rendered in viewDocsMode
// when cfgExisted=true and the selected mode differs from cfg.Mode.
func (m InstallModel) updateDocsMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

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

	case "left":
		m.docsModeButtons = m.docsModeButtons.move(-1)
		return m, nil

	case "right":
		m.docsModeButtons = m.docsModeButtons.move(1)
		return m, nil

	case "enter":
		activated := m.docsModeButtons.focused()
		switch activated {
		case "Continue":
			// Commit the mode from the current cursor position.
			if m.modeCursor == 0 {
				m.mode = configpkg.ModeVault
			} else {
				m.mode = configpkg.ModeInProject
			}
			// Reset button focus for the next screen.
			m.docsModeButtons.focus = 0
			m.pathButtons.focus = 0
			m.overwriteButtons.focus = 0
			if m.mode == configpkg.ModeVault {
				m.pathInput.Focus()
				m.step = stepPath
			} else {
				// In-project: skip path step; go to overwrite if needed, else progress.
				m.step = m.nextStepAfterDocsOrPath()
				if m.step == stepProgress {
					return m, m.runInstall()
				}
			}
			return m, nil
		case "Back":
			m.docsModeButtons.focus = 0
			m.step = prevStep(stepDocsMode)
			return m, nil
		}
	}

	return m, nil
}

// updatePath handles key input on the Vault Path Entry screen.
//
// The textinput owns ALL keys on this screen (path-screen exception):
//   - ←/→ edit the path cursor (they do NOT move button focus).
//   - Printable characters type into the path.
//   - Enter = Continue (if path is non-empty, advances to stepOverwrite when any
//     selected platform is already installed, else stepProgress).
//   - Esc = Back (navigates to stepDocsMode; non-alphabetic so it never collides
//     with path typing, e.g. C:\Users\bob\docs).
//
// The [Continue]/[Back] footer is a visual hint only; button focus does not
// change on this screen. Tab is a no-op (no focusZone switching).
func (m InstallModel) updatePath(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "esc":
		// Back shortcut (non-alphabetic so it never collides with path typing).
		m.pathButtons.focus = 0
		m.overwriteButtons.focus = 0
		m.pathInput.Focus()
		m.step = prevStep(stepPath)
		return m, nil

	case "enter":
		// Enter = Continue when path is non-empty.
		path := strings.TrimSpace(m.pathInput.Value())
		if path != "" {
			m.pathButtons.focus = 0
			m.overwriteButtons.focus = 0
			m.step = m.nextStepAfterDocsOrPath()
			if m.step == stepProgress {
				return m, m.runInstall()
			}
		}
		return m, nil
	}

	// All other keys (including ←/→) are forwarded to the textinput.
	var cmd tea.Cmd
	m.pathInput, cmd = m.pathInput.Update(msg)
	return m, cmd
}

// updateOverwrite handles key input on the Consolidated Overwrite screen.
//
// Both navigation axes are always live (no Tab / focusZone switching):
//   - ↑/↓ (k/j) move overwriteChoice (0=Overwrite all, 1=Install only missing).
//   - ←/→ move button-row focus.
//   - Enter activates the focused button:
//     [Install] → commits choice, routes to stepProgress (or stepDone if all-already-present).
//     [Back]    → prevStep(stepOverwrite). For vault mode prevStep=stepPath;
//     for in-project mode returns to stepDocsMode directly.
//   - Tab is a no-op (ignored).
//
// All-already-present guard (install-only-missing with all selected platforms installed):
// sets m.step=stepDone with a "nothing to do" summary — never calls runInstall.
func (m InstallModel) updateOverwrite(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "up", "k":
		if m.overwriteChoice > 0 {
			m.overwriteChoice--
		}
		return m, nil

	case "down", "j":
		if m.overwriteChoice < 1 {
			m.overwriteChoice++
		}
		return m, nil

	case "left":
		m.overwriteButtons = m.overwriteButtons.move(-1)
		return m, nil

	case "right":
		m.overwriteButtons = m.overwriteButtons.move(1)
		return m, nil

	case "enter":
		activated := m.overwriteButtons.focused()
		switch activated {
		case "Install":
			m.overwriteButtons.focus = 0
			return m.commitOverwriteAndInstall()
		case "Back":
			m.overwriteButtons.focus = 0
			// Mode-aware BACK: in-project skipped path, so go back to DocsMode.
			if m.mode == configpkg.ModeInProject {
				m.step = stepDocsMode
			} else {
				m.step = prevStep(stepOverwrite)
			}
			return m, nil
		}
	}

	return m, nil
}

// commitOverwriteAndInstall commits the overwrite choice and either starts the
// engine (stepProgress) or routes to a "nothing to do" done screen.
func (m InstallModel) commitOverwriteAndInstall() (tea.Model, tea.Cmd) {
	if m.overwriteChoice == 1 {
		// Install only missing: check if everything is already installed.
		noneLeft := true
		for _, p := range m.platforms {
			if p.selected && !m.alreadyInstalled[p.id] {
				noneLeft = false
				break
			}
		}
		if noneLeft {
			// All selected platforms already present — nothing to install.
			// Route to stepDone with a "nothing to do" summary. NEVER call
			// runInstall: plan.Platforms would be []string{} (non-nil empty)
			// and we bypass the engine entirely.
			m.step = stepDone
			m.progressLines = []string{"  — Nothing to install — all selected platforms already have Zeen present."}
			return m, nil
		}
	}

	m.step = stepProgress
	m.installing = true
	m = m.seedChecklist()
	return m, tea.Batch(m.runInstall(), tickCmd())
}

// seedChecklist builds the initial checklist from the current platform selection.
// For install-only-missing (overwriteChoice=1), already-installed platforms that
// are in alreadyInstalled are seeded as Skipped; all others are Pending.
// For overwrite-all (overwriteChoice=0), all selected platforms are Pending.
// Only selected platforms appear in the checklist.
func (m InstallModel) seedChecklist() InstallModel {
	var items []checklistItem
	for _, p := range m.platforms {
		if !p.selected {
			continue
		}
		state := stateChecklist_Pending
		if m.overwriteChoice == 1 && m.alreadyInstalled[p.id] {
			state = stateChecklist_Skipped
		}
		items = append(items, checklistItem{platformID: p.id, state: state})
	}
	m.checklist = items
	m.checklistCursor = 0
	return m
}

// tickCmd returns a tea.Cmd that fires a tickMsg after stepDelay.
// Using tea.Tick (not time.Sleep) keeps the tick cancelable and teatest-friendly:
// tests can inject tickMsg directly into Update() without waiting for wall-clock time.
func tickCmd() tea.Cmd {
	return tea.Tick(stepDelay, func(_ time.Time) tea.Msg {
		return tickMsg{}
	})
}

// processTickMsg advances the checklist cursor by one non-skipped step.
// Each call marks the current pending item as Installing and advances the cursor.
// If all items are processed, no further tick is emitted.
func (m InstallModel) processTickMsg() (tea.Model, tea.Cmd) {
	total := len(m.checklist)
	if total == 0 {
		return m, nil
	}

	// Find the next Pending item at or after checklistCursor.
	idx := m.checklistCursor
	for idx < total && m.checklist[idx].state != stateChecklist_Pending {
		idx++
	}

	if idx >= total {
		// All items processed — no more ticks needed.
		return m, nil
	}

	// Mark current item as Installing and advance cursor.
	m.checklist[idx].state = stateChecklist_Installing
	m.checklistCursor = idx + 1

	// Count non-skipped total for the tick guard.
	nonSkipped := 0
	for _, item := range m.checklist {
		if item.state != stateChecklist_Skipped {
			nonSkipped++
		}
	}
	_ = nonSkipped // guard: tick only runs while non-skipped items remain

	// Re-issue tickCmd for the next step.
	return m, tickCmd()
}

// BuildPlan returns the InstallPlan reflecting the current model state.
// This mirrors the plan that runInstall would send to the engine.
// For install-only-missing, plan.Platforms excludes already-installed platforms;
// it is always non-nil ([]string{} when empty, never nil).
func (m InstallModel) BuildPlan() configpkg.InstallPlan {
	var selectedIDs []string
	for _, p := range m.platforms {
		if p.selected {
			selectedIDs = append(selectedIDs, p.id)
		}
	}

	overwrite := make(map[string]bool)
	effectiveIDs := make([]string, 0, len(selectedIDs)) // non-nil empty by default

	if m.overwriteChoice == 0 {
		// Overwrite all.
		for _, id := range selectedIDs {
			overwrite[id] = true
			effectiveIDs = append(effectiveIDs, id)
		}
	} else {
		// Install only missing: exclude already-installed.
		for _, id := range selectedIDs {
			if !m.alreadyInstalled[id] {
				effectiveIDs = append(effectiveIDs, id)
			}
		}
	}

	return configpkg.InstallPlan{
		Platforms: effectiveIDs,
		Mode:      m.mode,
		BasePath:  configpkg.ExpandUserPath(m.pathInput.Value()),
		PrevMode:  configpkg.DocsMode(m.cfg.Mode),
		Yes:       true,
		Overwrite: overwrite,
	}
}
