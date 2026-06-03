package tui

import "github.com/charmbracelet/lipgloss"

var (
	accentColor   = lipgloss.Color("86")
	mutedColor    = lipgloss.Color("240")
	textColor     = lipgloss.Color("252")
	subtleColor   = lipgloss.Color("244")
	errorColor    = lipgloss.Color("203")
	successColor  = lipgloss.Color("114")
	borderColor   = lipgloss.Color("238")
	activeColor   = lipgloss.Color("117")
	panelBg       = lipgloss.Color("235")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(subtleColor)

	labelStyle = lipgloss.NewStyle().
			Foreground(textColor).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginTop(1)

	errStyle = lipgloss.NewStyle().
			Foreground(errorColor)

	successStyle = lipgloss.NewStyle().
			Foreground(successColor)

	infoStyle = lipgloss.NewStyle().
			Foreground(subtleColor)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor).
			Padding(0, 1)

	sidebarStyle = lipgloss.NewStyle().
			Width(22).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Background(panelBg).
			Padding(0, 1)

	chatPaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	inputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(0, 1)

	statusStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	tabActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("229")).
			Background(activeColor).
			Padding(0, 1)

	tabInactiveStyle = lipgloss.NewStyle().
			Foreground(subtleColor).
			Background(lipgloss.Color("237")).
			Padding(0, 1)

	sidebarItemActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor)

	sidebarItemStyle = lipgloss.NewStyle().
			Foreground(textColor)

	selfMsgStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("159")).
			Italic(true)

	peerMsgStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	modalOverlayStyle = lipgloss.NewStyle().
			Width(44).
			Border(lipgloss.DoubleBorder()).
			BorderForeground(accentColor).
			Background(lipgloss.Color("234")).
			Padding(1, 2)

	modalTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor).
			MarginBottom(1)

	footerStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(borderColor).
			Padding(0, 1).
			MarginTop(1)

	hudKeyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor)

	hudLabelStyle = lipgloss.NewStyle().
			Foreground(subtleColor)

	hudSepStyle = lipgloss.NewStyle().
			Foreground(borderColor)
)
