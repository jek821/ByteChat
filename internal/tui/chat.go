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
	SendFriendRequest(toUsername string) error
	AcceptFriendRequest(fromUsername string) error
}

type inputMode int

const (
	modeMessage inputMode = iota
	modeAddFriend
)

type sidebarEntry struct {
	username string
	pending  bool
}

type chatMessage struct {
	from string
	body string
	self bool
}

type chatModel struct {
	username  string
	entries   []sidebarEntry
	active    int
	threads   map[string][]chatMessage
	viewport  viewport.Model
	input     textinput.Model
	messenger messenger
	mode      inputMode
	err       string
	info      string
	ready     bool
	width     int
	height    int
}

func newChatModel(username string, messenger messenger) chatModel {
	input := textinput.New()
	input.Placeholder = "Type a message..."
	input.CharLimit = 512
	input.Prompt = "> "
	input.Focus()

	return chatModel{
		username:  username,
		threads:   make(map[string][]chatMessage),
		viewport:  viewport.New(0, 0),
		input:     input,
		messenger: messenger,
		mode:      modeMessage,
	}
}

func (m chatModel) Init() tea.Cmd {
	return tea.Batch(m.input.Focus(), textinput.Blink, tea.WindowSize())
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
		return m, m.input.Focus()

	case contactsUpdatedMsg:
		m.rebuildEntries(msg.friends, msg.pending)
		return m, nil

	case friendRequestMsg:
		m.addPending(msg.from)
		m.info = fmt.Sprintf("Friend request from %s", msg.from)
		return m, nil

	case chatDisconnectedMsg:
		m.err = "disconnected from chat server"
		return m, nil

	case incomingMessageMsg:
		m.appendMessage(msg.from, msg.body, false)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}

		if m.mode == modeAddFriend {
			switch msg.String() {
			case "esc":
				m.mode = modeMessage
				m.input.SetValue("")
				m.input.Placeholder = "Type a message..."
				m.err = ""
				return m, m.input.Focus()
			case "enter":
				username := strings.TrimSpace(m.input.Value())
				if username == "" {
					return m, nil
				}
				if m.messenger != nil {
					if err := m.messenger.SendFriendRequest(username); err != nil {
						m.err = err.Error()
						return m, nil
					}
				}
				m.mode = modeMessage
				m.input.SetValue("")
				m.input.Placeholder = "Type a message..."
				m.err = ""
				m.info = fmt.Sprintf("Friend request sent to %s", username)
				return m, m.input.Focus()
			}
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "tab":
			if len(m.entries) > 0 {
				m.active = (m.active + 1) % len(m.entries)
				m.syncViewport()
			}
			return m, nil
		case "shift+tab":
			if len(m.entries) > 0 {
				m.active--
				if m.active < 0 {
					m.active = len(m.entries) - 1
				}
				m.syncViewport()
			}
			return m, nil
		case "a":
			m.mode = modeAddFriend
			m.input.SetValue("")
			m.input.Placeholder = "Username to add as friend..."
			m.err = ""
			return m, m.input.Focus()
		case "enter":
			if len(m.entries) == 0 {
				return m, nil
			}
			entry := m.entries[m.active]
			if entry.pending {
				if m.messenger != nil {
					if err := m.messenger.AcceptFriendRequest(entry.username); err != nil {
						m.err = err.Error()
						return m, nil
					}
				}
				m.info = fmt.Sprintf("Accepted %s", entry.username)
				m.err = ""
				return m, nil
			}
			body := strings.TrimSpace(m.input.Value())
			if body == "" {
				return m, nil
			}
			if m.messenger != nil {
				if err := m.messenger.Send(entry.username, body); err != nil {
					m.err = err.Error()
					return m, nil
				}
			}
			m.appendMessage(entry.username, body, true)
			m.input.SetValue("")
			m.err = ""
			return m, tea.Batch(m.input.Focus(), textinput.Blink)
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *chatModel) rebuildEntries(friends, pending []string) {
	m.entries = nil
	for _, name := range pending {
		m.entries = append(m.entries, sidebarEntry{username: name, pending: true})
	}
	for _, name := range friends {
		m.entries = append(m.entries, sidebarEntry{username: name, pending: false})
	}
	if m.active >= len(m.entries) {
		m.active = 0
	}
	m.syncViewport()
}

func (m *chatModel) addPending(username string) {
	for _, e := range m.entries {
		if e.username == username && e.pending {
			return
		}
	}
	m.entries = append([]sidebarEntry{{username: username, pending: true}}, m.entries...)
	if m.active >= len(m.entries) {
		m.active = 0
	}
}

func (m *chatModel) appendMessage(peer, body string, self bool) {
	key := peer
	if self {
		key = m.activePeer()
	}
	if key == "" {
		key = peer
	}
	m.threads[key] = append(m.threads[key], chatMessage{from: peer, body: body, self: self})
	m.syncViewport()
}

func (m chatModel) activePeer() string {
	if len(m.entries) == 0 || m.active >= len(m.entries) {
		return ""
	}
	entry := m.entries[m.active]
	if entry.pending {
		return ""
	}
	return entry.username
}

func (m *chatModel) syncViewport() {
	peer := m.activePeer()
	messages := m.threads[peer]

	var b strings.Builder
	if peer == "" && len(m.entries) > 0 && m.entries[m.active].pending {
		b.WriteString("Select a friend request and press enter to accept.\n")
	} else if peer == "" {
		b.WriteString("No friend selected. Press a to add a friend.\n")
	}
	for _, msg := range messages {
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
	if len(m.entries) == 0 {
		sidebar.WriteString("  (no friends)\n\n  press a to add")
	} else {
		if len(m.entries) > 0 && m.entries[0].pending {
			sidebar.WriteString(labelStyle.Render("Requests") + "\n")
		}
		for i, entry := range m.entries {
			if i > 0 && entry.pending && !m.entries[i-1].pending {
				sidebar.WriteString("\n" + labelStyle.Render("Friends") + "\n")
			}
			prefix := "  "
			if entry.pending {
				prefix = "* "
			}
			line := prefix + entry.username
			if i == m.active {
				line = ">" + line[1:]
			}
			sidebar.WriteString(line)
			sidebar.WriteByte('\n')
		}
	}

	header := statusStyle.Render(fmt.Sprintf("Signed in as %s", m.username))
	if m.info != "" {
		header += "  " + statusStyle.Render(m.info)
	}
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
	help := helpStyle.Render("enter send/accept • tab switch • a add friend • q quit")
	if m.mode == modeAddFriend {
		help = helpStyle.Render("enter send request • esc cancel")
	}

	return header + "\n\n" + body + "\n" + inputBar + "\n" + help
}
