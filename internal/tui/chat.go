package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type sidebarTab int

const (
	tabFriends sidebarTab = iota
	tabIncoming
	tabOutgoing
)

type messenger interface {
	Send(toUsername, body string) error
	SendFriendRequest(toUsername string) error
	AcceptFriendRequest(fromUsername string) error
	RequestHistory(peerUsername string) error
}

type chatMessage struct {
	from string
	body string
	self bool
}

type chatModel struct {
	username      string
	sidebarTab    sidebarTab
	friends       []string
	incoming      []string
	outgoing      []string
	active        int
	threads       map[string][]chatMessage
	historyLoaded map[string]bool
	viewport      viewport.Model
	input         textinput.Model
	modal         addFriendModal
	messenger     messenger
	err           string
	info          string
	ready         bool
	width         int
	height        int
}

func newChatModel(username string, messenger messenger) chatModel {
	input := textinput.New()
	input.Placeholder = "Message..."
	input.CharLimit = 512
	input.Prompt = "❯ "
	input.Focus()

	return chatModel{
		username:      username,
		threads:       make(map[string][]chatMessage),
		historyLoaded: make(map[string]bool),
		viewport:      viewport.New(0, 0),
		input:         input,
		modal:         newAddFriendModal(),
		messenger:     messenger,
		sidebarTab:    tabFriends,
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
		m.viewport.Width = max(20, msg.Width-26)
		m.viewport.Height = max(4, msg.Height-8)
		m.input.Width = max(20, msg.Width-8)
		m.syncViewport()
		return m, m.input.Focus()

	case contactsUpdatedMsg:
		m.friends = msg.friends
		m.incoming = msg.pending
		m.outgoing = msg.outgoing
		m.clampActive()
		return m, m.loadHistoryForActive()

	case friendRequestMsg:
		m.incoming = appendUnique(m.incoming, msg.from)
		m.sidebarTab = tabIncoming
		m.info = fmt.Sprintf("Friend request from %s", msg.from)
		return m, nil

	case historyErrorMsg:
		if m.sidebarTab == tabFriends {
			m.err = msg.err.Error()
		}
		return m, nil

	case historyMsg:
		m.threads[msg.peer] = msg.messages
		m.historyLoaded[msg.peer] = true
		m.syncViewport()
		return m, nil

	case modalResultMsg:
		m.modal.loading = false
		if msg.err != nil {
			m.modal.err = msg.err.Error()
			var cmd tea.Cmd
			m.modal, cmd = m.modal.focus()
			return m, cmd
		}
		m.outgoing = appendUnique(m.outgoing, msg.username)
		m.sidebarTab = tabOutgoing
		m.info = fmt.Sprintf("Request sent to %s", msg.username)
		m.modal = m.modal.close()
		return m, m.input.Focus()

	case chatDisconnectedMsg:
		m.err = "disconnected from chat server"
		return m, nil

	case incomingMessageMsg:
		m.appendMessage(msg.from, msg.body, false)
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		if m.modal.visible {
			if msg.String() == "enter" && !m.modal.loading {
				username := strings.TrimSpace(m.modal.input.Value())
				if username != "" {
					m.modal.loading = true
					m.modal.err = ""
					return m, m.submitFriendRequest(username)
				}
			}
			var cmd tea.Cmd
			m.modal, cmd = m.modal.Update(msg)
			if msg.String() == "esc" {
				return m, m.input.Focus()
			}
			return m, cmd
		}

		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "1":
			m.sidebarTab = tabFriends
			m.active = 0
			m.syncViewport()
			return m, m.loadHistoryForActive()
		case "2":
			m.sidebarTab = tabIncoming
			m.active = 0
			m.syncViewport()
			return m, nil
		case "3":
			m.sidebarTab = tabOutgoing
			m.active = 0
			m.syncViewport()
			return m, nil
		case "up":
			if m.active > 0 {
				m.active--
				m.syncViewport()
				return m, m.loadHistoryForActive()
			}
			return m, nil
		case "down":
			if m.active < len(m.currentList())-1 {
				m.active++
				m.syncViewport()
				return m, m.loadHistoryForActive()
			}
			return m, nil
		case "a":
			m.modal = m.modal.open()
			var cmd tea.Cmd
			m.modal, cmd = m.modal.focus()
			return m, cmd
		case "enter":
			return m.handleEnter()
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m chatModel) handleEnter() (chatModel, tea.Cmd) {
	list := m.currentList()
	if len(list) == 0 {
		return m, nil
	}
	name := list[m.active]

	switch m.sidebarTab {
	case tabIncoming:
		if m.messenger != nil {
			if err := m.messenger.AcceptFriendRequest(name); err != nil {
				m.err = err.Error()
				return m, nil
			}
		}
		m.info = fmt.Sprintf("Accepted %s", name)
		m.err = ""
		return m, nil
	case tabOutgoing:
		return m, nil
	default:
		body := strings.TrimSpace(m.input.Value())
		if body == "" {
			return m, nil
		}
		if m.messenger != nil {
			if err := m.messenger.Send(name, body); err != nil {
				m.err = err.Error()
				return m, nil
			}
		}
		m.appendMessage(name, body, true)
		m.input.SetValue("")
		m.err = ""
		return m, tea.Batch(m.input.Focus(), textinput.Blink)
	}
}

func (m chatModel) submitFriendRequest(username string) tea.Cmd {
	msgr := m.messenger
	return func() tea.Msg {
		err := msgr.SendFriendRequest(username)
		return modalResultMsg{username: username, err: err}
	}
}

func (m chatModel) loadHistoryForActive() tea.Cmd {
	if m.sidebarTab != tabFriends {
		return nil
	}
	peer := m.activeFriend()
	if peer == "" || m.historyLoaded[peer] {
		return nil
	}
	msgr := m.messenger
	return func() tea.Msg {
		if err := msgr.RequestHistory(peer); err != nil {
			return historyErrorMsg{peer: peer, err: err}
		}
		return nil
	}
}

func (m *chatModel) currentList() []string {
	switch m.sidebarTab {
	case tabIncoming:
		return m.incoming
	case tabOutgoing:
		return m.outgoing
	default:
		return m.friends
	}
}

func (m *chatModel) activeFriend() string {
	if m.sidebarTab != tabFriends {
		return ""
	}
	list := m.friends
	if m.active >= len(list) {
		return ""
	}
	return list[m.active]
}

func (m *chatModel) clampActive() {
	if m.active >= len(m.currentList()) {
		m.active = 0
	}
}

func (m *chatModel) appendMessage(peer, body string, self bool) {
	key := peer
	if self {
		key = m.activeFriend()
		if key == "" {
			key = peer
		}
	}
	m.threads[key] = append(m.threads[key], chatMessage{from: peer, body: body, self: self})
	m.syncViewport()
}

func (m *chatModel) syncViewport() {
	peer := m.activeFriend()
	messages := m.threads[peer]

	var b strings.Builder
	switch m.sidebarTab {
	case tabIncoming:
		if len(m.incoming) == 0 {
			b.WriteString(infoStyle.Render("No incoming requests.") + "\n")
		} else {
			b.WriteString(infoStyle.Render("Select a request and press Enter to accept.") + "\n\n")
		}
	case tabOutgoing:
		if len(m.outgoing) == 0 {
			b.WriteString(infoStyle.Render("No outgoing requests.") + "\n")
		} else {
			b.WriteString(infoStyle.Render("Waiting for them to accept.") + "\n\n")
		}
	default:
		if peer == "" {
			b.WriteString(infoStyle.Render("Select a friend or press a to add one.") + "\n")
		} else if len(messages) == 0 {
			b.WriteString(infoStyle.Render(fmt.Sprintf("Conversation with %s", peer)) + "\n\n")
		}
	}

	for _, msg := range messages {
		line := msg.body
		if msg.self {
			b.WriteString(selfMsgStyle.Render("you: " + line))
		} else {
			b.WriteString(peerMsgStyle.Render(msg.from + ": " + line))
		}
		b.WriteByte('\n')
	}
	m.viewport.SetContent(b.String())
	m.viewport.GotoBottom()
}

func (m chatModel) renderTabs() string {
	tabs := []struct {
		id    sidebarTab
		label string
	}{
		{tabFriends, "Friends"},
		{tabIncoming, "In"},
		{tabOutgoing, "Out"},
	}
	var b strings.Builder
	for _, t := range tabs {
		label := t.label
		if t.id == tabIncoming && len(m.incoming) > 0 {
			label = fmt.Sprintf("In(%d)", len(m.incoming))
		}
		if t.id == tabOutgoing && len(m.outgoing) > 0 {
			label = fmt.Sprintf("Out(%d)", len(m.outgoing))
		}
		if m.sidebarTab == t.id {
			b.WriteString(tabActiveStyle.Render(label))
		} else {
			b.WriteString(tabInactiveStyle.Render(label))
		}
		b.WriteByte(' ')
	}
	return strings.TrimSpace(b.String())
}

func (m chatModel) renderSidebar() string {
	var b strings.Builder
	b.WriteString(m.renderTabs())
	b.WriteByte('\n')
	b.WriteString(strings.Repeat("─", 20))
	b.WriteByte('\n')

	list := m.currentList()
	if len(list) == 0 {
		switch m.sidebarTab {
		case tabIncoming:
			b.WriteString(infoStyle.Render("  none"))
		case tabOutgoing:
			b.WriteString(infoStyle.Render("  none"))
		default:
			b.WriteString(infoStyle.Render("  no friends"))
		}
		return b.String()
	}

	for i, name := range list {
		prefix := "  "
		if i == m.active {
			prefix = "▸ "
		}
		line := prefix + name
		if i == m.active {
			b.WriteString(sidebarItemActiveStyle.Render(line))
		} else {
			b.WriteString(sidebarItemStyle.Render(line))
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func (m chatModel) View() string {
	if !m.ready {
		return infoStyle.Render("Connecting...")
	}

	header := headerStyle.Render("byteChat") + " " + statusStyle.Render(fmt.Sprintf("@%s", m.username))
	if m.info != "" {
		header += "  " + successStyle.Render(m.info)
	}
	if m.err != "" {
		header += "  " + errStyle.Render(m.err)
	}

	chatTitle := "Chat"
	if peer := m.activeFriend(); peer != "" {
		chatTitle = peer
	}

	chatPane := chatPaneStyle.
		Width(m.viewport.Width).
		Height(m.viewport.Height).
		Render(labelStyle.Render(chatTitle)+"\n"+strings.Repeat("─", max(8, m.viewport.Width-2))+"\n"+m.viewport.View())

	sidePane := sidebarStyle.Render(m.renderSidebar())
	inputBar := inputStyle.Width(m.input.Width).Render(m.input.View())
	body := lipgloss.JoinHorizontal(lipgloss.Top, sidePane, chatPane)
	hud := chatHUD(m)

	base := header + "\n\n" + body + "\n" + inputBar + "\n" + hud

	if m.modal.visible {
		overlayHeight := max(4, m.height-lipgloss.Height(hud)-1)
		overlay := lipgloss.Place(
			m.width, overlayHeight,
			lipgloss.Center, lipgloss.Center,
			m.modal.View(),
			lipgloss.WithWhitespaceBackground(lipgloss.Color("235")),
			lipgloss.WithWhitespaceForeground(lipgloss.Color("238")),
		)
		return overlay + "\n" + hud
	}
	return base
}

func appendUnique(list []string, item string) []string {
	for _, v := range list {
		if v == item {
			return list
		}
	}
	return append(list, item)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
