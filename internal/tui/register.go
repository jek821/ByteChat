package tui

import (
	"ByteChat/internal/client"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type registerModel struct {
	auth     client.AuthClient
	inputs   []textinput.Model
	focusIdx int
	loading  bool
	err      error
}

func newRegisterModel(auth client.AuthClient) registerModel {
	inputs := make([]textinput.Model, 2)
	inputs[0] = textinput.New()
	inputs[0].Placeholder = "username"
	inputs[0].CharLimit = 32
	inputs[0].Width = 30
	inputs[0].Prompt = "> "

	inputs[1] = textinput.New()
	inputs[1].Placeholder = "password (min 8 chars)"
	inputs[1].EchoMode = textinput.EchoPassword
	inputs[1].EchoCharacter = '•'
	inputs[1].CharLimit = 64
	inputs[1].Width = 30
	inputs[1].Prompt = "> "

	inputs[0].Focus()

	return registerModel{auth: auth, inputs: inputs}
}

func (m registerModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m registerModel) Update(msg tea.Msg) (registerModel, tea.Cmd) {
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
			if m.focusIdx > len(m.inputs)-1 {
				m.focusIdx = 0
			}
			if m.focusIdx < 0 {
				m.focusIdx = len(m.inputs) - 1
			}
			cmds := make([]tea.Cmd, len(m.inputs))
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
			m.err = nil
			return m, m.submit()
		}
	case authErrorMsg:
		m.loading = false
		m.err = msg.err
		return m, nil
	case registerSuccessMsg:
		m.loading = false
		return m, nil
	}

	if m.loading {
		return m, nil
	}

	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return m, tea.Batch(cmds...)
}

func (m registerModel) submit() tea.Cmd {
	username := m.inputs[0].Value()
	password := m.inputs[1].Value()
	auth := m.auth
	return func() tea.Msg {
		creds, err := auth.Register(username, password)
		if err != nil {
			return authErrorMsg{err: err}
		}
		return registerSuccessMsg{creds: creds}
	}
}

func (m registerModel) View() string {
	s := titleStyle.Render("Register") + "\n\n"
	s += labelStyle.Render("Username") + "\n" + m.inputs[0].View() + "\n\n"
	s += labelStyle.Render("Password") + "\n" + m.inputs[1].View() + "\n"

	if m.err != nil {
		s += "\n" + errStyle.Render(m.err.Error())
	}
	if m.loading {
		s += "\n" + statusStyle.Render("Creating account...")
	}

	return s
}
