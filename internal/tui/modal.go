package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type addFriendModal struct {
	input   textinput.Model
	visible bool
	loading bool
	err     string
	success string
}

func newAddFriendModal() addFriendModal {
	input := textinput.New()
	input.Placeholder = "username"
	input.CharLimit = 32
	input.Width = 30
	input.Prompt = "@ "
	return addFriendModal{input: input}
}

func (m addFriendModal) Init() tea.Cmd {
	return textinput.Blink
}

func (m addFriendModal) open() addFriendModal {
	m.visible = true
	m.loading = false
	m.err = ""
	m.success = ""
	m.input.SetValue("")
	return m
}

func (m addFriendModal) close() addFriendModal {
	m.visible = false
	m.loading = false
	m.err = ""
	m.success = ""
	m.input.Blur()
	return m
}

func (m addFriendModal) Update(msg tea.Msg) (addFriendModal, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m.close(), nil
		case "enter":
			if m.loading {
				return m, nil
			}
			if m.input.Value() == "" {
				return m, nil
			}
			m.loading = true
			m.err = ""
			return m, nil
		}
	}

	if m.loading {
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m addFriendModal) View() string {
	if !m.visible {
		return ""
	}

	s := modalTitleStyle.Render("Add Friend") + "\n\n"
	s += labelStyle.Render("Username") + "\n"
	s += m.input.View() + "\n"

	if m.loading {
		s += "\n" + infoStyle.Render("Sending request...")
	} else if m.err != "" {
		s += "\n" + errStyle.Render(m.err)
	} else 	if m.success != "" {
		s += "\n" + successStyle.Render(m.success)
	}

	return modalOverlayStyle.Render(s)
}

func (m addFriendModal) focus() (addFriendModal, tea.Cmd) {
	m.input.Focus()
	return m, textinput.Blink
}
