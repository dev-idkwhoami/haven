package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"haven/server/models"
)

type activityEntry struct {
	Time    string
	Icon    string
	Message string
	Color   lipgloss.Color
}

// DashboardModel shows server info and recent activity.
type DashboardModel struct {
	deps           *Deps
	width          int
	height         int
	serverName     string
	listenAddr     string
	accessMode     string
	usersOnline    int
	pendingReqs    int
	uptime         string
	recentActivity []activityEntry
}

// NewDashboard creates the dashboard page model.
func NewDashboard(deps *Deps) *DashboardModel {
	return &DashboardModel{deps: deps}
}

func (d *DashboardModel) Init() tea.Cmd { return nil }

func (d *DashboardModel) refresh() {
	d.serverName = d.deps.ServerName
	d.listenAddr = d.deps.ListenAddr

	d.accessMode = d.deps.Hot.AccessMode()
	if d.accessMode == "" {
		var srv models.Server
		if err := d.deps.DB.First(&srv).Error; err == nil {
			d.accessMode = srv.AccessMode
		}
	}

	d.usersOnline = d.deps.Hub.ClientCount()
	d.pendingReqs = d.deps.WaitingRoom.Count()
	d.uptime = formatDuration(time.Since(d.deps.StartTime))

	var entries []models.AuditLogEntry
	d.deps.DB.Order("created_at DESC").Limit(10).Find(&entries)
	d.recentActivity = make([]activityEntry, len(entries))
	for i, e := range entries {
		targetShort := e.TargetID
		if len(targetShort) > 8 {
			targetShort = targetShort[:8]
		}
		d.recentActivity[i] = activityEntry{
			Time:    e.CreatedAt.Format("15:04:05"),
			Icon:    actionIcon(e.Action),
			Message: e.Action + " " + e.TargetType + " " + targetShort,
			Color:   actionColor(e.Action),
		}
	}
}

func (d *DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.width = msg.Width
		d.height = msg.Height
	case tickMsg:
		d.refresh()
	}
	return d, nil
}

func (d *DashboardModel) View() string {
	d.refresh()

	// Page title
	title := StylePageTitle.Render("Dashboard") + "    " +
		StyleOnline.Render("● ONLINE")

	// Server Info
	pendingStyle := StyleValue
	if d.pendingReqs > 0 {
		pendingStyle = StyleWarning
	}
	infoContent := strings.Join([]string{
		renderKV("Name:", d.serverName),
		renderKV("Listening:", d.listenAddr),
		renderKV("Access Mode:", d.accessMode),
		renderKV("Users Online:", fmt.Sprintf("%d", d.usersOnline)),
		StyleLabel.Render("Requests:") + " " + pendingStyle.Render(fmt.Sprintf("%d pending", d.pendingReqs)),
		renderKV("Uptime:", d.uptime),
	}, "\n")

	boxWidth := d.width - 4
	if boxWidth < 40 {
		boxWidth = 60
	}
	infoBox := renderBox("Server Info", infoContent, boxWidth)

	// Recent Activity
	activityLines := make([]string, 0, len(d.recentActivity))
	for _, e := range d.recentActivity {
		line := fmt.Sprintf("  %s  %s %s",
			lipgloss.NewStyle().Foreground(ColorTextMuted).Render(e.Time),
			lipgloss.NewStyle().Foreground(e.Color).Render(e.Icon),
			lipgloss.NewStyle().Foreground(e.Color).Render(e.Message),
		)
		activityLines = append(activityLines, line)
	}
	if len(activityLines) == 0 {
		activityLines = append(activityLines,
			lipgloss.NewStyle().Foreground(ColorTextMuted).Render("  No recent activity"))
	}
	activityBox := renderBox("Recent Activity", strings.Join(activityLines, "\n"), boxWidth)

	return lipgloss.JoinVertical(lipgloss.Left, title, "", infoBox, "", activityBox)
}
