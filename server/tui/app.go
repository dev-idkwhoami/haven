package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"gorm.io/gorm"

	"haven/server/auth"
	"haven/server/config"
	"haven/server/ws"
	"haven/shared"
)

// Page identifies the active TUI page.
type Page int

const (
	PageDashboard Page = iota
	PageRequests
	PageUsers
	PageBans
	PageInvites
	PageSettings
	PageLogs
)

// Deps holds shared dependencies injected from main.
type Deps struct {
	DB          *gorm.DB
	Hub         *ws.Hub
	Hot         *config.HotConfig
	WaitingRoom *auth.WaitingRoom
	LogBuffer   *LogBuffer
	ServerName  string
	ListenAddr  string
	StartTime   time.Time
}

// tickMsg triggers periodic data refresh.
type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// App is the top-level Bubble Tea model.
type App struct {
	deps           *Deps
	activePage     Page
	sidebar        SidebarModel
	pages          map[Page]tea.Model
	width          int
	height         int
	sidebarFocused bool
}

// NewApp creates the TUI application with all pages wired up.
func NewApp(deps *Deps) *App {
	sidebar := NewSidebar()

	pages := map[Page]tea.Model{
		PageDashboard: NewDashboard(deps),
		PageRequests:  NewRequests(deps),
		PageUsers:     NewUsers(deps),
		PageBans:      NewBans(deps),
		PageInvites:   NewInvites(deps),
		PageSettings:  NewSettings(deps),
		PageLogs:      NewLogs(deps),
	}

	return &App{
		deps:           deps,
		activePage:     PageDashboard,
		sidebar:        sidebar,
		pages:          pages,
		sidebarFocused: true,
	}
}

func (a *App) Init() tea.Cmd {
	cmds := []tea.Cmd{tickCmd()}
	for _, p := range a.pages {
		if cmd := p.Init(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.sidebar.height = msg.Height - 2

		contentWidth := msg.Width - a.sidebar.width - 1
		contentHeight := msg.Height - 2
		var cmds []tea.Cmd
		for page, model := range a.pages {
			sizeMsg := tea.WindowSizeMsg{Width: contentWidth, Height: contentHeight}
			updated, cmd := model.Update(sizeMsg)
			a.pages[page] = updated
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return a, tea.Batch(cmds...)

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return a, tea.Quit
		case "tab":
			a.sidebarFocused = !a.sidebarFocused
			return a, nil
		}

		if a.sidebarFocused {
			oldPage := a.sidebar.ActivePage()
			var cmd tea.Cmd
			a.sidebar, cmd = a.sidebar.Update(msg)
			newPage := a.sidebar.ActivePage()
			if oldPage != newPage {
				a.activePage = newPage
			}
			return a, cmd
		}

		// Delegate to active page
		if model, ok := a.pages[a.activePage]; ok {
			updated, cmd := model.Update(msg)
			a.pages[a.activePage] = updated
			return a, cmd
		}

	case tickMsg:
		// Update request badge count
		pendingCount := a.deps.WaitingRoom.Count()
		a.sidebar = a.sidebar.SetBadge(PageRequests, pendingCount)

		// Forward tick to active page
		if model, ok := a.pages[a.activePage]; ok {
			updated, cmd := model.Update(msg)
			a.pages[a.activePage] = updated
			return a, tea.Batch(cmd, tickCmd())
		}
		return a, tickCmd()
	}

	return a, nil
}

func (a *App) View() string {
	if a.width == 0 || a.height == 0 {
		return "Initializing..."
	}

	bodyHeight := a.height - 2

	// Title bar
	titleLeft := lipgloss.NewStyle().Bold(true).Foreground(ColorText).
		Render(fmt.Sprintf("  ▶ %s", a.deps.ServerName))
	titleRight := lipgloss.NewStyle().Foreground(ColorTextMuted).
		Render(fmt.Sprintf("v%s  |  q: quit  |  ?: help  ", shared.Version))
	gap := a.width - lipgloss.Width(titleLeft) - lipgloss.Width(titleRight)
	if gap < 0 {
		gap = 0
	}
	titleBar := lipgloss.NewStyle().
		Background(ColorTitleBarBg).
		Width(a.width).
		Render(titleLeft + strings.Repeat(" ", gap) + titleRight)

	// Sidebar
	a.sidebar.height = bodyHeight
	sidebarView := a.sidebar.View()

	// Divider
	dividerLines := make([]string, bodyHeight)
	for i := range dividerLines {
		dividerLines[i] = "│"
	}
	divider := lipgloss.NewStyle().
		Foreground(ColorDivider).
		Render(strings.Join(dividerLines, "\n"))

	// Content area
	contentWidth := a.width - a.sidebar.width - 1
	contentView := ""
	if model, ok := a.pages[a.activePage]; ok {
		contentView = model.View()
	}
	content := StyleContentArea.
		Width(contentWidth).
		Height(bodyHeight).
		Render(contentView)

	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, divider, content)

	// Help bar
	var helpText string
	if a.sidebarFocused {
		helpText = "↑/↓: navigate  |  tab: switch panel  |  q: quit"
	} else {
		helpText = "tab: sidebar  |  q: quit"
	}
	helpBar := lipgloss.NewStyle().
		Foreground(ColorTextMuted).
		Width(a.width).
		Render("  " + helpText)

	return lipgloss.JoinVertical(lipgloss.Left, titleBar, body, helpBar)
}
