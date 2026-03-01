package tui

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/oklog/ulid/v2"

	"haven/server/models"
)

// InvitesModel manages the invites page.
type InvitesModel struct {
	deps     *Deps
	invites  []models.InviteCode
	cursor   int
	width    int
	height   int
	message  string
	msgTimer int
}

// NewInvites creates the invites page model.
func NewInvites(deps *Deps) *InvitesModel {
	return &InvitesModel{deps: deps}
}

func (inv *InvitesModel) Init() tea.Cmd { return nil }

func (inv *InvitesModel) loadInvites() {
	inv.deps.DB.Order("created_at DESC").Find(&inv.invites)
	if inv.cursor >= len(inv.invites) {
		inv.cursor = max(0, len(inv.invites)-1)
	}
}

func (inv *InvitesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		inv.width = msg.Width
		inv.height = msg.Height
	case tickMsg:
		inv.loadInvites()
		if inv.msgTimer > 0 {
			inv.msgTimer--
			if inv.msgTimer == 0 {
				inv.message = ""
			}
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if inv.cursor > 0 {
				inv.cursor--
			}
		case "down", "j":
			if inv.cursor < len(inv.invites)-1 {
				inv.cursor++
			}
		case "n": // new invite
			code := generateInviteCode()
			invite := models.InviteCode{
				ID:   ulid.Make().String(),
				Code: code,
			}
			inv.deps.DB.Create(&invite)
			inv.message = fmt.Sprintf("Created: %s", code)
			inv.msgTimer = 5
			inv.loadInvites()
		case "d": // delete
			if len(inv.invites) > 0 && inv.cursor < len(inv.invites) {
				inv.deps.DB.Delete(&inv.invites[inv.cursor])
				inv.loadInvites()
			}
		}
	}
	return inv, nil
}

func (inv *InvitesModel) View() string {
	inv.loadInvites()

	title := StylePageTitle.Render(fmt.Sprintf("Invites (%d)", len(inv.invites)))
	if inv.message != "" {
		title += "  " + StyleOnline.Render(inv.message)
	}

	boxWidth := inv.width - 4
	if boxWidth < 40 {
		boxWidth = 60
	}

	header := StyleTableHeader.Render(
		fmt.Sprintf("  %-3s %-20s %-10s %-12s %s", "#", "Code", "Uses Left", "Expires", "Created"))

	rows := make([]string, len(inv.invites))
	for i, invite := range inv.invites {
		usesLeft := "∞"
		if invite.UsesLeft != nil {
			usesLeft = fmt.Sprintf("%d", *invite.UsesLeft)
		}
		expires := "never"
		if invite.ExpiresAt != nil {
			expires = invite.ExpiresAt.Format("2006-01-02")
		}

		row := fmt.Sprintf("  %-3d %-20s %-10s %-12s %s",
			i+1, invite.Code, usesLeft, expires, invite.CreatedAt.Format("2006-01-02"))

		if i == inv.cursor {
			rows[i] = StyleTableRowSelected.Render(row)
		} else {
			rows[i] = StyleTableRow.Render(row)
		}
	}
	if len(rows) == 0 {
		rows = append(rows,
			lipgloss.NewStyle().Foreground(ColorTextMuted).Render("  No invites"))
	}

	tableContent := strings.Join(append([]string{header}, rows...), "\n")
	tableBox := renderBox("Invite Codes", tableContent, boxWidth)

	help := StyleHelpBar.Render("[n] New Invite  [d] Delete  [↑/↓] Select")

	return lipgloss.JoinVertical(lipgloss.Left, title, "", tableBox, "", help)
}

func generateInviteCode() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
