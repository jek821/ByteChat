package tui

import (
	"ByteChat/internal/client"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type adminLoginModel struct {
	admin    *client.AdminClient
	inputs   []textinput.Model
	focusIdx int
	loading  bool
	err      error
}

func newAdminLoginModel(admin *client.AdminClient) adminLoginModel {
	inputs := make([]textinput.Model, 2)
	inputs[0] = textinput.New()
	inputs[0].Placeholder = "admin username"
	inputs[0].CharLimit = 32
	inputs[0].Width = 30
	inputs[0].Prompt = "> "
	inputs[1] = textinput.New()
	inputs[1].Placeholder = "password"
	inputs[1].EchoMode = textinput.EchoPassword
	inputs[1].EchoCharacter = '•'
	inputs[1].CharLimit = 64
	inputs[1].Width = 30
	inputs[1].Prompt = "> "
	inputs[0].Focus()
	return adminLoginModel{admin: admin, inputs: inputs}
}

func (m adminLoginModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m adminLoginModel) Update(msg tea.Msg) (adminLoginModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return navigateMsg{to: screenWelcome} }
		case "tab", "shift+tab", "up", "down":
			if msg.String() == "shift+tab" || msg.String() == "up" {
				m.focusIdx--
			} else {
				m.focusIdx++
			}
			if m.focusIdx > 1 {
				m.focusIdx = 0
			}
			if m.focusIdx < 0 {
				m.focusIdx = 1
			}
			cmds := make([]tea.Cmd, 2)
			for i := range m.inputs {
				if i == m.focusIdx {
					cmds[i] = m.inputs[i].Focus()
				} else {
					m.inputs[i].Blur()
				}
			}
			return m, tea.Batch(cmds...)
		case "enter":
			m.loading = true
			return m, m.submit()
		}
	case authErrorMsg:
		m.loading = false
		m.err = msg.err
		return m, nil
	case adminLoginSuccessMsg:
		m.loading = false
		return m, nil
	}
	if m.loading {
		return m, nil
	}
	cmds := make([]tea.Cmd, 2)
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return m, tea.Batch(cmds...)
}

func (m adminLoginModel) submit() tea.Cmd {
	user := m.inputs[0].Value()
	pass := m.inputs[1].Value()
	admin := m.admin
	return func() tea.Msg {
		creds, err := admin.Login(user, pass)
		if err != nil {
			return authErrorMsg{err: err}
		}
		return adminLoginSuccessMsg{creds: creds}
	}
}

func (m adminLoginModel) View() string {
	s := titleStyle.Render("Admin Login") + "\n\n"
	s += labelStyle.Render("Username") + "\n" + m.inputs[0].View() + "\n\n"
	s += labelStyle.Render("Password") + "\n" + m.inputs[1].View() + "\n"
	if m.err != nil {
		s += "\n" + errStyle.Render(m.err.Error())
	}
	if m.loading {
		s += "\n" + infoStyle.Render("Signing in...")
	}
	return s
}
