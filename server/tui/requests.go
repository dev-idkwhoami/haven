package tui

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"haven/server/models"
)

// RequestEntry holds display data for one access request.
type RequestEntry struct {
	ID          string
	DisplayName string
	PubKeyHex   string
	Message     string
	IsOnline    bool
	CreatedAt   time.Time
}

// RequestsModel manages the access requests page.
type RequestsModel struct {
	deps     *Deps
	requests []RequestEntry
	cursor   int
	width    int
	height   int
	message  string // status message (e.g. "Approved alice")
	msgTimer int    // ticks until message clears
}

// NewRequests creates the requests page model.
func NewRequests(deps *Deps) *RequestsModel {
	return &RequestsModel{deps: deps}
}

func (r *RequestsModel) Init() tea.Cmd { return nil }

func (r *RequestsModel) loadRequests() {
	var reqs []models.AccessRequest
	r.deps.DB.Where("status = ?", "pending").Order("created_at ASC").Find(&reqs)

	r.requests = make([]RequestEntry, len(reqs))
	for i, req := range reqs {
		pubHex := hex.EncodeToString(req.PublicKey)
		msg := ""
		if req.Message != nil {
			msg = *req.Message
		}
		r.requests[i] = RequestEntry{
			ID:          req.ID,
			DisplayName: req.DisplayName,
			PubKeyHex:   pubHex,
			Message:     msg,
			IsOnline:    r.deps.WaitingRoom.IsOnline(pubHex),
			CreatedAt:   req.CreatedAt,
		}
	}
	if r.cursor >= len(r.requests) {
		r.cursor = max(0, len(r.requests)-1)
	}
}

func (r *RequestsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		r.width = msg.Width
		r.height = msg.Height
	case tickMsg:
		r.loadRequests()
		if r.msgTimer > 0 {
			r.msgTimer--
			if r.msgTimer == 0 {
				r.message = ""
			}
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if r.cursor > 0 {
				r.cursor--
			}
		case "down", "j":
			if r.cursor < len(r.requests)-1 {
				r.cursor++
			}
		case "a":
			if len(r.requests) > 0 && r.cursor < len(r.requests) {
				req := r.requests[r.cursor]
				r.deps.WaitingRoom.Approve(req.PubKeyHex)
				r.deps.DB.Model(&models.AccessRequest{}).
					Where("id = ?", req.ID).
					Updates(map[string]any{"status": "approved", "reviewed_by": "tui-admin"})
				r.message = fmt.Sprintf("Approved %s", req.DisplayName)
				r.msgTimer = 3
				r.loadRequests()
			}
		case "r":
			if len(r.requests) > 0 && r.cursor < len(r.requests) {
				req := r.requests[r.cursor]
				r.deps.WaitingRoom.Reject(req.PubKeyHex)
				r.deps.DB.Model(&models.AccessRequest{}).
					Where("id = ?", req.ID).
					Updates(map[string]any{"status": "rejected", "reviewed_by": "tui-admin"})
				r.message = fmt.Sprintf("Rejected %s", req.DisplayName)
				r.msgTimer = 3
				r.loadRequests()
			}
		}
	}
	return r, nil
}

func (r *RequestsModel) View() string {
	r.loadRequests()

	// Page title
	title := StylePageTitle.Render(fmt.Sprintf("Requests (%d pending)", len(r.requests)))
	if r.message != "" {
		title += "  " + StyleOnline.Render(r.message)
	}

	boxWidth := r.width - 4
	if boxWidth < 40 {
		boxWidth = 60
	}

	// Table header
	header := fmt.Sprintf("  %-3s %-14s %-20s %-8s %-10s",
		"#", "Name", "Message", "Online", "Submitted")
	tableHeader := StyleTableHeader.Render(header)

	// Table rows
	rows := make([]string, len(r.requests))
	for i, req := range r.requests {
		onlineStr := lipgloss.NewStyle().Foreground(ColorOffline).Render("○")
		if req.IsOnline {
			onlineStr = lipgloss.NewStyle().Foreground(ColorOnline).Render("●")
		}

		msg := req.Message
		if msg == "" {
			msg = "—"
		}

		timeAgo := formatTimeAgo(req.CreatedAt)

		row := fmt.Sprintf("  %-3d %-14s %-20s  %s      %-10s",
			i+1, truncate(req.DisplayName, 12), truncate(msg, 18), onlineStr, timeAgo)

		if i == r.cursor {
			rows[i] = StyleTableRowSelected.Render(row)
		} else {
			rows[i] = StyleTableRow.Render(row)
		}
	}
	if len(rows) == 0 {
		rows = append(rows,
			lipgloss.NewStyle().Foreground(ColorTextMuted).Render("  No pending requests"))
	}

	tableContent := strings.Join(append([]string{tableHeader}, rows...), "\n")
	tableBox := renderBox("Pending Access Requests", tableContent, boxWidth)

	// Detail panel
	detail := ""
	if len(r.requests) > 0 && r.cursor < len(r.requests) {
		req := r.requests[r.cursor]
		shortKey := req.PubKeyHex
		if len(shortKey) > 16 {
			shortKey = shortKey[:8] + "…" + shortKey[len(shortKey)-4:]
		}

		statusStr := StyleWarning.Render("● online – waiting")
		if !req.IsOnline {
			statusStr = lipgloss.NewStyle().Foreground(ColorTextMuted).Render("○ offline")
		}

		msg := req.Message
		if msg == "" {
			msg = "—"
		}

		detailContent := strings.Join([]string{
			renderKV("Name:", req.DisplayName),
			renderKV("PubKey:", shortKey),
			renderKV("Message:", msg),
			StyleLabel.Render("Status:") + " " + statusStr,
		}, "\n")
		detail = renderBox("Request Detail", detailContent, boxWidth)
	}

	help := StyleHelpBar.Render("[a] Approve  [r] Reject  [↑/↓] Select  [esc] Back")

	parts := []string{title, "", tableBox}
	if detail != "" {
		parts = append(parts, "", detail)
	}
	parts = append(parts, "", help)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}
