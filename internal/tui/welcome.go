package tui

import tea "github.com/charmbracelet/bubbletea"

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
	return titleStyle.Render("byteChat") + "\n" +
		subtitleStyle.Render("CLI chat over HTTPS + TCP/TLS") + "\n\n" +
		labelStyle.Render("l") + helpStyle.Render(" login") + "\n" +
		labelStyle.Render("r") + helpStyle.Render(" register") + "\n" +
		labelStyle.Render("q") + helpStyle.Render(" quit")
}
