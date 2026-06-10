package main

// ---------------------------------------------------------------------------
// Slice 1 unit tests — TDD RED first
// ---------------------------------------------------------------------------
//
// Tests T1.7 (all sub-tests) for:
//   - buttonRow: Tab/Shift+Tab focus, Enter activation, render with Plain styles.
//   - renderBanner: Plain mode byte-stability, no ANSI escape sequences.
//   - bannerPalette: correct mapping for indices 0/1/2.
//   - Welcome screen: Salir quits, Continuar advances to stepPlatformSelect.
//
// These tests must compile and run GREEN after T1.3–T1.6 are complete.

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// ---------------------------------------------------------------------------
// TestButtonRow_Handle_TabMovesFocus
// ---------------------------------------------------------------------------

// TestButtonRow_Handle_TabMovesFocus verifies that Tab cycles focus forward and
// Shift+Tab cycles it backward, with wrapping.
func TestButtonRow_Handle_TabMovesFocus(t *testing.T) {
	b := buttonRow{labels: []string{"A", "B", "C"}, focus: 0}

	// Tab moves focus forward.
	_, _ = b.handle("tab")
	if b.focus != 1 {
		t.Fatalf("after tab, focus = %d, want 1", b.focus)
	}

	_, _ = b.handle("tab")
	if b.focus != 2 {
		t.Fatalf("after second tab, focus = %d, want 2", b.focus)
	}

	// Wraps back to 0.
	_, _ = b.handle("tab")
	if b.focus != 0 {
		t.Fatalf("tab should wrap focus to 0, got %d", b.focus)
	}

	// Shift+Tab goes backward.
	_, _ = b.handle("shift+tab")
	if b.focus != 2 {
		t.Fatalf("shift+tab from 0 should wrap to 2, got %d", b.focus)
	}

	_, _ = b.handle("shift+tab")
	if b.focus != 1 {
		t.Fatalf("shift+tab from 2 should go to 1, got %d", b.focus)
	}
}

// TestButtonRow_Handle_LeftRightMovesFocus verifies left/right keys move focus.
func TestButtonRow_Handle_LeftRightMovesFocus(t *testing.T) {
	b := buttonRow{labels: []string{"A", "B"}, focus: 0}

	_, _ = b.handle("right")
	if b.focus != 1 {
		t.Fatalf("right from 0: focus = %d, want 1", b.focus)
	}

	_, _ = b.handle("left")
	if b.focus != 0 {
		t.Fatalf("left from 1: focus = %d, want 0", b.focus)
	}
}

// ---------------------------------------------------------------------------
// TestButtonRow_Handle_EnterActivates
// ---------------------------------------------------------------------------

// TestButtonRow_Handle_EnterActivates verifies that Enter returns the focused label.
func TestButtonRow_Handle_EnterActivates(t *testing.T) {
	b := buttonRow{labels: []string{"Continuar", "Salir"}, focus: 0}

	label, handled := b.handle("enter")
	if !handled {
		t.Fatal("handle('enter') returned handled=false")
	}
	if label != "Continuar" {
		t.Fatalf("expected activated label 'Continuar', got %q", label)
	}

	// Move to second button.
	_, _ = b.handle("tab")
	label, handled = b.handle("enter")
	if !handled {
		t.Fatal("handle('enter') returned handled=false on second button")
	}
	if label != "Salir" {
		t.Fatalf("expected activated label 'Salir', got %q", label)
	}
}

// ---------------------------------------------------------------------------
// TestButtonRow_Render_FocusedChip
// ---------------------------------------------------------------------------

// TestButtonRow_Render_FocusedChip verifies focused/unfocused rendering in Plain mode.
func TestButtonRow_Render_FocusedChip(t *testing.T) {
	s := NoColor() // Plain = true
	b := buttonRow{labels: []string{"Yes", "No"}, focus: 0}

	rendered := b.render(s)
	// In plain mode both labels must appear.
	if !strings.Contains(rendered, "Yes") {
		t.Errorf("render missing 'Yes':\n%s", rendered)
	}
	if !strings.Contains(rendered, "No") {
		t.Errorf("render missing 'No':\n%s", rendered)
	}
	// No ANSI escape sequences in plain mode.
	if strings.Contains(rendered, "\x1b[") {
		t.Errorf("render contains ANSI escapes in plain mode:\n%s", rendered)
	}
}

// ---------------------------------------------------------------------------
// TestRenderBanner_Plain_ByteStable
// ---------------------------------------------------------------------------

// TestRenderBanner_Plain_ByteStable verifies that renderBanner with Plain=true
// is byte-identical across two calls and contains no ANSI escape sequences.
func TestRenderBanner_Plain_ByteStable(t *testing.T) {
	s := NoColor()
	first := renderBanner(welcomeBanner, s)
	second := renderBanner(welcomeBanner, s)

	if first != second {
		t.Fatal("renderBanner with Plain=true is not byte-identical across two calls")
	}
	if strings.Contains(first, "\x1b[") {
		t.Errorf("renderBanner with Plain=true contains ANSI escape sequences")
	}
}

// ---------------------------------------------------------------------------
// TestBannerPalette_Mapping
// ---------------------------------------------------------------------------

// TestBannerPalette_Mapping verifies the palette color order matches the brand spec.
func TestBannerPalette_Mapping(t *testing.T) {
	if len(bannerPalette) < 3 {
		t.Fatalf("bannerPalette len = %d, want >= 3", len(bannerPalette))
	}
	if bannerPalette[0] != "#00A5E7" {
		t.Errorf("bannerPalette[0] = %q, want '#00A5E7' (cyan)", bannerPalette[0])
	}
	if bannerPalette[1] != "#0D1012" {
		t.Errorf("bannerPalette[1] = %q, want '#0D1012' (dark)", bannerPalette[1])
	}
	if bannerPalette[2] != "#E5E8EA" {
		t.Errorf("bannerPalette[2] = %q, want '#E5E8EA' (grey)", bannerPalette[2])
	}
}

// ---------------------------------------------------------------------------
// TestWelcomeScreen_Salir_Quits
// ---------------------------------------------------------------------------

// TestWelcomeScreen_Salir_Quits verifies that activating Salir returns tea.Quit.
func TestWelcomeScreen_Salir_Quits(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	// Ensure we are on the welcome step.
	m.step = stepWelcome
	// Focus the Salir button (index 1).
	m.welcomeButtons.focus = 1

	_, cmd := m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))
	if cmd == nil {
		t.Fatal("expected a tea.Cmd (tea.Quit) when Salir is activated, got nil")
	}
	// Execute the command to check it produces a quit message.
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("Salir should produce tea.QuitMsg, got %T", msg)
	}
}

// ---------------------------------------------------------------------------
// TestWelcomeScreen_Continuar_AdvancesToPlatformSelect
// ---------------------------------------------------------------------------

// TestWelcomeScreen_Continuar_AdvancesToPlatformSelect verifies that activating
// Continuar from the welcome step moves to stepPlatformSelect.
func TestWelcomeScreen_Continuar_AdvancesToPlatformSelect(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepWelcome
	// Ensure Continuar is focused (index 0).
	m.welcomeButtons.focus = 0

	next, _ := m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))
	im, ok := next.(InstallModel)
	if !ok {
		t.Fatalf("Update returned %T, want InstallModel", next)
	}
	if im.step != stepPlatformSelect {
		t.Fatalf("after Continuar, step = %v, want stepPlatformSelect", im.step)
	}
}
