package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"haven/server/models"
	"haven/server/ws"
)

type userEntry struct {
	PubKeyHex string
	UserID    string
	Name      string
	Role      string
}

// UsersModel manages the connected users page.
type UsersModel struct {
	deps     *Deps
	users    []userEntry
	cursor   int
	width    int
	height   int
	message  string
	msgTimer int
}

// NewUsers creates the users page model.
func NewUsers(deps *Deps) *UsersModel {
	return &UsersModel{deps: deps}
}

func (u *UsersModel) Init() tea.Cmd { return nil }

func (u *UsersModel) loadUsers() {
	u.users = nil
	u.deps.Hub.ForEachClient(func(c *ws.Client) {
		name := c.PubKeyHex[:12] + "…"
		var user models.User
		if err := u.deps.DB.Where("id = ?", c.UserID).First(&user).Error; err == nil {
			name = user.DisplayName
		}

		role := "Member"
		if u.deps.Hot.IsOwner(c.PubKey) {
			role = "Owner"
		}

		u.users = append(u.users, userEntry{
			PubKeyHex: c.PubKeyHex,
			UserID:    c.UserID,
			Name:      name,
			Role:      role,
		})
	})
	if u.cursor >= len(u.users) {
		u.cursor = max(0, len(u.users)-1)
	}
}

func (u *UsersModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		u.width = msg.Width
		u.height = msg.Height
	case tickMsg:
		u.loadUsers()
		if u.msgTimer > 0 {
			u.msgTimer--
			if u.msgTimer == 0 {
				u.message = ""
			}
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if u.cursor > 0 {
				u.cursor--
			}
		case "down", "j":
			if u.cursor < len(u.users)-1 {
				u.cursor++
			}
		case "K": // shift+k to kick
			if len(u.users) > 0 && u.cursor < len(u.users) {
				user := u.users[u.cursor]
				if client := u.deps.Hub.GetClient(user.PubKeyHex); client != nil {
					client.Close()
					u.message = fmt.Sprintf("Kicked %s", user.Name)
					u.msgTimer = 3
					u.loadUsers()
				}
			}
		}
	}
	return u, nil
}

func (u *UsersModel) View() string {
	u.loadUsers()

	title := StylePageTitle.Render(fmt.Sprintf("Users (%d online)", len(u.users)))
	if u.message != "" {
		title += "  " + StyleOnline.Render(u.message)
	}

	boxWidth := u.width - 4
	if boxWidth < 40 {
		boxWidth = 60
	}

	header := fmt.Sprintf("  %-3s %-16s %-16s %-10s",
		"#", "Name", "PubKey", "Role")
	tableHeader := StyleTableHeader.Render(header)

	rows := make([]string, len(u.users))
	for i, user := range u.users {
		shortKey := user.PubKeyHex
		if len(shortKey) > 12 {
			shortKey = shortKey[:12] + "…"
		}

		row := fmt.Sprintf("  %-3d %-16s %-16s %-10s",
			i+1, truncate(user.Name, 14), shortKey, user.Role)

		if i == u.cursor {
			rows[i] = StyleTableRowSelected.Render(row)
		} else {
			rows[i] = StyleTableRow.Render(row)
		}
	}
	if len(rows) == 0 {
		rows = append(rows,
			lipgloss.NewStyle().Foreground(ColorTextMuted).Render("  No users connected"))
	}

	tableContent := strings.Join(append([]string{tableHeader}, rows...), "\n")
	tableBox := renderBox("Connected Users", tableContent, boxWidth)

	help := StyleHelpBar.Render("[K] Kick  [↑/↓] Select")

	return lipgloss.JoinVertical(lipgloss.Left, title, "", tableBox, "", help)
}
