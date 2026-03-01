package tui

import "github.com/charmbracelet/lipgloss"

// Colors matching the haven.pen TUI mockup.
var (
	ColorBg            = lipgloss.Color("#0d0d0d")
	ColorSidebarBg     = lipgloss.Color("#111111")
	ColorTitleBarBg    = lipgloss.Color("#000000")
	ColorDivider       = lipgloss.Color("#222222")
	ColorBorder        = lipgloss.Color("#333333")
	ColorBorderLight   = lipgloss.Color("#444444")
	ColorPrimary       = lipgloss.Color("#00D084")
	ColorText          = lipgloss.Color("#e0e0e0")
	ColorTextSecondary = lipgloss.Color("#888888")
	ColorTextMuted     = lipgloss.Color("#555555")
	ColorWarning       = lipgloss.Color("#FFB800")
	ColorError         = lipgloss.Color("#FF4444")
	ColorOnline        = lipgloss.Color("#00D084")
	ColorOffline       = lipgloss.Color("#555555")
)

// Reusable lipgloss styles.
var (
	StyleSidebar = lipgloss.NewStyle().
			Background(ColorSidebarBg).
			Padding(1, 0)

	StyleSidebarItem = lipgloss.NewStyle().
				Foreground(ColorTextSecondary).
				Padding(0, 2)

	StyleSidebarActive = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true).
				Padding(0, 1).
				PaddingRight(2).
				BorderLeft(true).
				BorderStyle(lipgloss.ThickBorder()).
				BorderLeftForeground(ColorPrimary)

	StyleSidebarHeader = lipgloss.NewStyle().
				Foreground(ColorTextMuted).
				Padding(0, 2).
				MarginBottom(1)

	StyleContentArea = lipgloss.NewStyle().
				Padding(1, 2)

	StylePageTitle = lipgloss.NewStyle().
			Foreground(ColorText).
			Bold(true)

	StyleBoxTitle = lipgloss.NewStyle().
			Foreground(ColorTextMuted).
			Italic(true)

	StyleLabel = lipgloss.NewStyle().
			Foreground(ColorTextMuted).
			Width(18)

	StyleValue = lipgloss.NewStyle().
			Foreground(ColorText)

	StyleOnline = lipgloss.NewStyle().
			Foreground(ColorOnline).
			Bold(true)

	StyleWarning = lipgloss.NewStyle().
			Foreground(ColorWarning)

	StyleError = lipgloss.NewStyle().
			Foreground(ColorError)

	StyleHelpBar = lipgloss.NewStyle().
			Foreground(ColorTextMuted).
			Padding(0, 1)

	StyleTableHeader = lipgloss.NewStyle().
				Foreground(ColorTextMuted).
				Bold(true)

	StyleTableRow = lipgloss.NewStyle().
			Foreground(ColorText)

	StyleTableRowSelected = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true)

	StyleBadge = lipgloss.NewStyle().
			Foreground(ColorWarning).
			Bold(true)
)
