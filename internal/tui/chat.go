package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type messenger interface {
	Send(toUsername, body string) error
}

type chatMessage struct {
	from string
	body string
	self bool
}

type chatModel struct {
	username  string
	contacts  []string
	active    int
	messages  []chatMessage
	viewport  viewport.Model
	input     textinput.Model
	messenger messenger
	err       string
	ready     bool
	width     int
	height    int
}

func newChatModel(username string, messenger messenger) chatModel {
	input := textinput.New()
	input.Placeholder = "Type a message..."
	input.CharLimit = 512
	input.Prompt = "> "

	vp := viewport.New(0, 0)

	return chatModel{
		username:  username,
		messenger: messenger,
		viewport:  vp,
		input:     input,
	}
}

func (m chatModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, tea.WindowSize())
}

func (m chatModel) Update(msg tea.Msg) (chatModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.viewport.Width = msg.Width - 24
		m.viewport.Height = msg.Height - 6
		m.input.Width = msg.Width - 6
		m.syncViewport()
		return m, nil

	case contactsUpdatedMsg:
		m.contacts = msg.contacts
		if len(m.contacts) == 0 {
			m.active = 0
		} else if m.active >= len(m.contacts) {
			m.active = 0
		}
		return m, nil

	case chatDisconnectedMsg:
		m.err = "disconnected from chat server"
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.active > 0 {
				m.active--
			}
			return m, nil
		case "down", "j":
			if m.active < len(m.contacts)-1 {
				m.active++
			}
			return m, nil
		case "enter":
			body := strings.TrimSpace(m.input.Value())
			if body == "" {
				return m, nil
			}
			if len(m.contacts) == 0 {
				m.err = "no contacts available"
				return m, nil
			}
			to := m.contacts[m.active]
			if m.messenger != nil {
				if err := m.messenger.Send(to, body); err != nil {
					m.err = err.Error()
					return m, nil
				}
			}
			m.messages = append(m.messages, chatMessage{
				from: m.username,
				body: body,
				self: true,
			})
			m.input.SetValue("")
			m.err = ""
			m.syncViewport()
			return m, textinput.Blink
		}

	case incomingMessageMsg:
		m.messages = append(m.messages, chatMessage{
			from: msg.from,
			body: msg.body,
			self: false,
		})
		m.syncViewport()
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *chatModel) syncViewport() {
	var b strings.Builder
	for _, msg := range m.messages {
		prefix := msg.from + ": "
		if msg.self {
			prefix = "you: "
		}
		b.WriteString(prefix)
		b.WriteString(msg.body)
		b.WriteByte('\n')
	}
	m.viewport.SetContent(b.String())
	m.viewport.GotoBottom()
}

func (m chatModel) View() string {
	if !m.ready {
		return "Loading..."
	}

	var sidebar strings.Builder
	if len(m.contacts) == 0 {
		sidebar.WriteString("  (no contacts)")
	} else {
		for i, contact := range m.contacts {
			line := contact
			if i == m.active {
				line = "> " + contact
			} else {
				line = "  " + contact
			}
			sidebar.WriteString(line)
			sidebar.WriteByte('\n')
		}
	}

	header := statusStyle.Render(fmt.Sprintf("Signed in as %s", m.username))
	if m.err != "" {
		header += "  " + errStyle.Render(m.err)
	}
	chatPane := chatPaneStyle.
		Width(m.viewport.Width).
		Height(m.viewport.Height).
		Render(m.viewport.View())
	sidePane := sidebarStyle.Render(sidebar.String())
	inputBar := inputStyle.Width(m.input.Width).Render(m.input.View())

	body := lipgloss.JoinHorizontal(lipgloss.Top, sidePane, chatPane)
	help := helpStyle.Render("enter send • j/k switch contact • q quit")

	return header + "\n\n" + body + "\n" + inputBar + "\n" + help
}
