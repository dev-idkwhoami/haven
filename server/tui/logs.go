package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// LogsModel shows a live tail of server log output.
type LogsModel struct {
	deps   *Deps
	offset int // scroll offset from bottom
	width  int
	height int
}

// NewLogs creates the logs page model.
func NewLogs(deps *Deps) *LogsModel {
	return &LogsModel{deps: deps}
}

func (l *LogsModel) Init() tea.Cmd { return nil }

func (l *LogsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		l.width = msg.Width
		l.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			l.offset++
		case "down", "j":
			if l.offset > 0 {
				l.offset--
			}
		case "G": // go to bottom
			l.offset = 0
		case "g": // go to top
			if l.deps.LogBuffer != nil {
				l.offset = l.deps.LogBuffer.Count()
			}
		}
	}
	return l, nil
}

func (l *LogsModel) View() string {
	title := StylePageTitle.Render("Logs")

	if l.deps.LogBuffer == nil {
		return title + "\n\n" +
			lipgloss.NewStyle().Foreground(ColorTextMuted).
				Render("  Log buffer not available (headless mode?)")
	}

	lines := l.deps.LogBuffer.Lines()
	totalLines := len(lines)

	visibleLines := l.height - 6
	if visibleLines < 1 {
		visibleLines = 10
	}

	// Calculate visible window
	end := totalLines - l.offset
	if end < 0 {
		end = 0
	}
	if end > totalLines {
		end = totalLines
	}
	start := end - visibleLines
	if start < 0 {
		start = 0
	}

	displayLines := lines[start:end]

	logContent := strings.Join(displayLines, "\n")
	if logContent == "" {
		logContent = lipgloss.NewStyle().Foreground(ColorTextMuted).
			Render("  No log entries")
	}

	boxWidth := l.width - 4
	if boxWidth < 40 {
		boxWidth = 60
	}
	logBox := renderBox(
		fmt.Sprintf("Server Logs (%d lines)", totalLines),
		logContent, boxWidth)

	scrollInfo := ""
	if l.offset > 0 {
		scrollInfo = StyleWarning.Render(fmt.Sprintf("  ↑ %d lines above", l.offset))
	}

	help := StyleHelpBar.Render("[↑/↓] Scroll  [g] Top  [G] Bottom")

	parts := []string{title, "", logBox}
	if scrollInfo != "" {
		parts = append(parts, scrollInfo)
	}
	parts = append(parts, "", help)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}
