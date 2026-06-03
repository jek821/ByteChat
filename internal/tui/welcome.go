package tui

import (
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

type welcomeModel struct{}

func newWelcomeModel() welcomeModel {
	return welcomeModel{}
}

func (m welcomeModel) Init() tea.Cmd {
	return nil
}

func (m welcomeModel) Update(msg tea.Msg) (welcomeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "l":
			return m, func() tea.Msg { return navigateMsg{to: screenLogin} }
		case "r":
			return m, func() tea.Msg { return navigateMsg{to: screenRegister} }
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m welcomeModel) View() string {
	logo := titleStyle.Render("byteChat")
	tagline := subtitleStyle.Render("end-to-end CLI chat")
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(1, 3).
		Render(logo + "\n" + tagline)

	return box
}
