package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type screen int

const (
	screenHome screen = iota
	screenInstall
	screenUninstall
)

type RootModel struct {
	screen     screen
	bundle     Bundle
	styles     Styles
	width      int
	height     int
	menuCursor int
	notice     string
	install    *InstallModel
	uninstall  *UninstallModel
}

func (m RootModel) Init() tea.Cmd { return nil }

func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "ctrl+c" {
		return m, tea.Quit
	}

	if key, ok := msg.(tea.KeyMsg); ok {
		switch m.screen {
		case screenInstall:
			if m.install != nil && m.install.step == stepDone {
				m.screen = screenHome
				m.install = nil
				return m, nil
			}
		case screenUninstall:
			if m.uninstall != nil && m.uninstall.step == uninstallStepDone {
				m.screen = screenHome
				m.uninstall = nil
				return m, nil
			}
		case screenHome:
			switch key.String() {
			case "up", "k":
				m.menuCursor = (m.menuCursor + 2) % 3
				return m, nil
			case "down", "j":
				m.menuCursor = (m.menuCursor + 1) % 3
				return m, nil
			case "q":
				return m, tea.Quit
			case "enter":
				switch m.menuCursor {
				case 0:
					return m.startInstall()
				case 1:
					return m.startUninstall()
				default:
					return m, tea.Quit
				}
			}
		}
	}

	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = typed.Width
		m.height = typed.Height
	}

	switch m.screen {
	case screenInstall:
		if m.install == nil {
			return m, nil
		}
		next, cmd := m.install.Update(msg)
		updated := next.(InstallModel)
		m.install = &updated
		return m, cmd
	case screenUninstall:
		if m.uninstall == nil {
			return m, nil
		}
		next, cmd := m.uninstall.Update(msg)
		updated := next.(UninstallModel)
		m.uninstall = &updated
		return m, cmd
	default:
		return m, nil
	}
}

func (m RootModel) startInstall() (RootModel, tea.Cmd) {
	if missing := ValidateBundle(m.bundle); len(missing) > 0 {
		m.notice = fmt.Sprintf("bundle is incomplete — %d artifacts missing.", len(missing))
		return m, nil
	}

	cfg, cfgExisted, err := loadConfig()
	if err != nil {
		cfg = AppConfig{}
		cfgExisted = false
	}

	allPlatforms := detectAllPlatforms(m.bundle.Manifest)
	if len(allPlatforms) == 0 {
		m.notice = "No supported platform detected — install opencode, claude, copilot, qwen, or pi first."
		return m, nil
	}

	install := newInstallModel(cfg, cfgExisted, m.bundle, allPlatforms, m.styles)
	if m.width > 0 && m.height > 0 {
		install.width = m.width
		install.height = m.height
	}
	m.install = &install
	m.notice = ""
	m.screen = screenInstall
	return m, nil
}

func (m RootModel) startUninstall() (RootModel, tea.Cmd) {
	allPlatforms := detectAllPlatforms(m.bundle.Manifest)
	installed := checkWhatIsInstalled(m.bundle.Manifest, allPlatforms)
	if len(installed) == 0 {
		m.notice = "doc-agent-ai is not installed on any detected platform — nothing to uninstall."
		return m, nil
	}
	uninstall := newUninstallModel(installed, m.bundle.Manifest, m.styles)
	if m.width > 0 && m.height > 0 {
		uninstall.width = m.width
		uninstall.height = m.height
	}
	m.uninstall = &uninstall
	m.notice = ""
	m.screen = screenUninstall
	return m, nil
}

func (m RootModel) View() string {
	switch m.screen {
	case screenInstall:
		if m.install != nil {
			return m.install.View()
		}
	case screenUninstall:
		if m.uninstall != nil {
			return m.uninstall.View()
		}
	}
	return m.viewHome()
}

func (m RootModel) viewHome() string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(renderZeenLockup(m.styles))
	sb.WriteString("\n")
	if m.notice != "" {
		sb.WriteString(m.styles.Warning.Render("  ⚠  "+m.notice) + "\n\n")
	} else {
		sb.WriteString(m.styles.Subtitle.Render("  Multi-platform documentation agents — setup") + "\n\n")
	}

	rows := []struct{ title, desc string }{
		{"Install", "Set up doc-agent-ai on your detected AI platforms"},
		{"Uninstall", "Remove doc-agent-ai from your platforms"},
		{"Quit", "Exit"},
	}
	for i, row := range rows {
		label := "    " + row.title
		if i == m.menuCursor {
			label = m.styles.SelectedItem.Render("  ▸  " + row.title)
		} else {
			label = "     " + m.styles.Subtitle.Render(row.title)
		}
		sb.WriteString(label)
		if row.desc != "" {
			sb.WriteString("        " + m.styles.Dim.Render(row.desc))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	sb.WriteString(m.styles.Dim.Render("  ↑/↓ move · enter select · q quit") + "\n")
	return sb.String()
}
