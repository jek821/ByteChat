package tui

import (
	"fmt"

	"ByteChat/internal/client"

	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	cfg      Config
	screen   screen
	welcome  welcomeModel
	login    loginModel
	register registerModel
	chat     chatModel
	chatConn *client.ChatClient
}

func New(cfg Config) Model {
	return Model{
		cfg:      cfg,
		screen:   screenWelcome,
		welcome:  newWelcomeModel(),
		login:    newLoginModel(cfg.Auth),
		register: newRegisterModel(cfg.Auth),
	}
}

func (m Model) Init() tea.Cmd {
	return m.welcome.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" && m.screen == screenWelcome {
			return m, tea.Quit
		}

	case navigateMsg:
		m.screen = msg.to
		switch m.screen {
		case screenLogin:
			m.login = newLoginModel(m.cfg.Auth)
			return m, m.login.Init()
		case screenRegister:
			m.register = newRegisterModel(m.cfg.Auth)
			return m, m.register.Init()
		}
		return m, nil

	case loginSuccessMsg:
		return m.connectChat(msg.creds)

	case registerSuccessMsg:
		return m.connectChat(msg.creds)

	case chatDisconnectedMsg:
		if m.chatConn != nil {
			_ = m.chatConn.Close()
		}
		m.login = newLoginModel(m.cfg.Auth)
		m.login.err = fmt.Errorf("disconnected from chat server")
		m.screen = screenLogin
		return m, m.login.Init()

	case client.ChatEvent:
		switch msg.Kind {
		case client.EventMessage:
			if m.screen == screenChat {
				m.chat, _ = m.chat.Update(incomingMessageMsg{from: msg.Message.From, body: msg.Message.Body})
			}
			return m, waitForChatEvent(m.chatConn)
		case client.EventContacts:
			if m.screen == screenChat {
				m.chat, _ = m.chat.Update(contactsUpdatedMsg{contacts: msg.Contacts})
			}
			return m, waitForChatEvent(m.chatConn)
		}
	}

	var cmd tea.Cmd
	switch m.screen {
	case screenWelcome:
		m.welcome, cmd = m.welcome.Update(msg)
	case screenLogin:
		m.login, cmd = m.login.Update(msg)
	case screenRegister:
		m.register, cmd = m.register.Update(msg)
	case screenChat:
		m.chat, cmd = m.chat.Update(msg)
	}
	return m, cmd
}

func (m Model) connectChat(creds client.Credentials) (Model, tea.Cmd) {
	chatConn := client.NewChatClient(m.cfg.TCPAddr)
	if err := chatConn.Connect(creds.Token); err != nil {
		m.login = newLoginModel(m.cfg.Auth)
		m.login.err = err
		m.screen = screenLogin
		return m, m.login.Init()
	}

	m.chatConn = chatConn
	m.chat = newChatModel(creds.Username, chatConn)
	m.screen = screenChat
	return m, tea.Batch(m.chat.Init(), waitForChatEvent(chatConn))
}

func waitForChatEvent(conn *client.ChatClient) tea.Cmd {
	if conn == nil {
		return nil
	}
	return func() tea.Msg {
		event, ok := <-conn.Events()
		if !ok {
			return chatDisconnectedMsg{}
		}
		return event
	}
}

func (m Model) View() string {
	switch m.screen {
	case screenWelcome:
		return m.welcome.View()
	case screenLogin:
		return m.login.View()
	case screenRegister:
		return m.register.View()
	case screenChat:
		return m.chat.View()
	default:
		return ""
	}
}

func Run(cfg Config) error {
	p := tea.NewProgram(New(cfg), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
