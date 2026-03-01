package tui

import (
	"encoding/hex"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"haven/server/models"
)

// BansModel manages the bans page.
type BansModel struct {
	deps   *Deps
	bans   []models.Ban
	cursor int
	width  int
	height int
}

// NewBans creates the bans page model.
func NewBans(deps *Deps) *BansModel {
	return &BansModel{deps: deps}
}

func (b *BansModel) Init() tea.Cmd { return nil }

func (b *BansModel) loadBans() {
	b.deps.DB.Order("created_at DESC").Find(&b.bans)
	if b.cursor >= len(b.bans) {
		b.cursor = max(0, len(b.bans)-1)
	}
}

func (b *BansModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		b.width = msg.Width
		b.height = msg.Height
	case tickMsg:
		b.loadBans()
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if b.cursor > 0 {
				b.cursor--
			}
		case "down", "j":
			if b.cursor < len(b.bans)-1 {
				b.cursor++
			}
		case "u": // unban
			if len(b.bans) > 0 && b.cursor < len(b.bans) {
				b.deps.DB.Delete(&b.bans[b.cursor])
				b.loadBans()
			}
		}
	}
	return b, nil
}

func (b *BansModel) View() string {
	b.loadBans()

	title := StylePageTitle.Render(fmt.Sprintf("Bans (%d)", len(b.bans)))

	boxWidth := b.width - 4
	if boxWidth < 40 {
		boxWidth = 60
	}

	header := StyleTableHeader.Render(
		fmt.Sprintf("  %-3s %-20s %-20s %s", "#", "PubKey", "Reason", "Date"))

	rows := make([]string, len(b.bans))
	for i, ban := range b.bans {
		pubHex := hex.EncodeToString(ban.PublicKey)
		if len(pubHex) > 16 {
			pubHex = pubHex[:16] + "…"
		}
		reason := "—"
		if ban.Reason != nil {
			reason = *ban.Reason
		}

		row := fmt.Sprintf("  %-3d %-20s %-20s %s",
			i+1, pubHex, truncate(reason, 18), ban.CreatedAt.Format("2006-01-02"))

		if i == b.cursor {
			rows[i] = StyleTableRowSelected.Render(row)
		} else {
			rows[i] = StyleTableRow.Render(row)
		}
	}
	if len(rows) == 0 {
		rows = append(rows,
			lipgloss.NewStyle().Foreground(ColorTextMuted).Render("  No bans"))
	}

	tableContent := strings.Join(append([]string{header}, rows...), "\n")
	tableBox := renderBox("Ban List", tableContent, boxWidth)

	help := StyleHelpBar.Render("[u] Unban  [↑/↓] Select")

	return lipgloss.JoinVertical(lipgloss.Left, title, "", tableBox, "", help)
}
