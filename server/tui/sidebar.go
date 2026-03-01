package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// NavItem represents a single entry in the sidebar navigation.
type NavItem struct {
	Icon  string
	Label string
	Page  Page
	Badge int
}

// SidebarModel manages sidebar navigation state and rendering.
type SidebarModel struct {
	items       []NavItem
	activeIndex int
	width       int
	height      int
}

// NewSidebar creates a sidebar with the standard navigation items.
func NewSidebar() SidebarModel {
	return SidebarModel{
		items: []NavItem{
			{Icon: "■", Label: "Dashboard", Page: PageDashboard},
			{Icon: "●", Label: "Requests", Page: PageRequests},
			{Icon: "◆", Label: "Users", Page: PageUsers},
			{Icon: "✕", Label: "Bans", Page: PageBans},
			{Icon: "→", Label: "Invites", Page: PageInvites},
			{Icon: "⚙", Label: "Settings", Page: PageSettings},
			{Icon: "≡", Label: "Logs", Page: PageLogs},
		},
		width: 22,
	}
}

// ActivePage returns the Page value of the currently highlighted nav item.
func (s SidebarModel) ActivePage() Page {
	return s.items[s.activeIndex].Page
}

// Update handles keyboard input for sidebar navigation.
func (s SidebarModel) Update(msg tea.Msg) (SidebarModel, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "up", "k":
			if s.activeIndex > 0 {
				s.activeIndex--
			}
		case "down", "j":
			if s.activeIndex < len(s.items)-1 {
				s.activeIndex++
			}
		}
	}
	return s, nil
}

// SetBadge sets the badge count for a given page (e.g. pending request count).
func (s SidebarModel) SetBadge(page Page, count int) SidebarModel {
	for i := range s.items {
		if s.items[i].Page == page {
			s.items[i].Badge = count
			break
		}
	}
	return s
}

// View renders the sidebar.
func (s SidebarModel) View() string {
	header := StyleSidebarHeader.Render("NAVIGATION")

	items := header + "\n"
	for i, item := range s.items {
		label := fmt.Sprintf("%s %s", item.Icon, item.Label)
		if item.Badge > 0 {
			label += StyleBadge.Render(fmt.Sprintf(" [%d]", item.Badge))
		}

		if i == s.activeIndex {
			items += StyleSidebarActive.Render(label) + "\n"
		} else {
			items += StyleSidebarItem.Render(label) + "\n"
		}
	}

	return StyleSidebar.
		Width(s.width).
		Height(s.height).
		Render(items)
}
