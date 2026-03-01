package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"haven/server/models"
)

// SettingsModel shows a read-only view of server configuration.
type SettingsModel struct {
	deps   *Deps
	width  int
	height int
}

// NewSettings creates the settings page model.
func NewSettings(deps *Deps) *SettingsModel {
	return &SettingsModel{deps: deps}
}

func (s *SettingsModel) Init() tea.Cmd { return nil }

func (s *SettingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		s.width = msg.Width
		s.height = msg.Height
	}
	return s, nil
}

func (s *SettingsModel) View() string {
	title := StylePageTitle.Render("Settings")

	var srv models.Server
	s.deps.DB.First(&srv)

	accessMode := s.deps.Hot.AccessMode()
	if accessMode == "" {
		accessMode = srv.AccessMode
	}

	boxWidth := s.width - 4
	if boxWidth < 40 {
		boxWidth = 60
	}

	content := strings.Join([]string{
		renderKV("Server Name:", srv.Name),
		renderKV("Access Mode:", accessMode),
		renderKV("Max File Size:", formatBytes(srv.MaxFileSize)),
		renderKV("Storage Limit:", formatBytes(srv.TotalStorageLimit)),
		renderKV("Access Requests:", fmt.Sprintf("%v", s.deps.Hot.AllowAccessRequests())),
		renderKV("Request Timeout:", s.deps.Hot.RequestTimeout().String()),
	}, "\n")

	box := renderBox("Server Configuration", content, boxWidth)

	note := lipgloss.NewStyle().Foreground(ColorTextMuted).
		Render("  (read-only — edit haven-server.toml to change)")

	return lipgloss.JoinVertical(lipgloss.Left, title, "", box, "", note)
}
