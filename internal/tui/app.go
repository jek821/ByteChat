package tui

import (
	"fmt"

	"ByteChat/internal/client"

	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	cfg        Config
	screen     screen
	width      int
	height     int
	welcome    welcomeModel
	login      loginModel
	register   registerModel
	chat       chatModel
	adminLogin adminLoginModel
	adminPanel adminPanelModel
	chatConn   *client.ChatClient
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
	return tea.Batch(m.welcome.Init(), tea.WindowSize())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if sz, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = sz.Width
		m.height = sz.Height
	}

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
		case screenAdminLogin:
			if m.cfg.Admin == nil {
				m.screen = screenWelcome
				return m, nil
			}
			m.adminLogin = newAdminLoginModel(m.cfg.Admin)
			return m, m.adminLogin.Init()
		}
		return m, nil

	case adminLoginSuccessMsg:
		m.adminPanel = newAdminPanelModel(m.cfg.Admin, msg.creds.Username)
		m.screen = screenAdminPanel
		if m.width > 0 && m.height > 0 {
			m.adminPanel, _ = m.adminPanel.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		}
		return m, m.adminPanel.Init()

	case loginSuccessMsg:
		return m.connectChat(msg.creds, screenLogin)

	case registerSuccessMsg:
		return m.connectChat(msg.creds, screenRegister)

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
				m.chat, _ = m.chat.Update(contactsUpdatedMsg{
					friends:  msg.Contacts.Friends,
					pending:  msg.Contacts.PendingRequests,
					outgoing: msg.Contacts.OutgoingRequests,
				})
			}
			return m, waitForChatEvent(m.chatConn)
		case client.EventFriendRequest:
			if m.screen == screenChat {
				m.chat, _ = m.chat.Update(friendRequestMsg{from: msg.From})
			}
			return m, waitForChatEvent(m.chatConn)
		case client.EventHistory:
			if m.screen == screenChat {
				msgs := make([]chatMessage, len(msg.History.Messages))
				for i, hm := range msg.History.Messages {
					msgs[i] = chatMessage{
						from: hm.From,
						body: hm.Body,
						self: hm.From == msg.History.SelfUser,
					}
				}
				m.chat, _ = m.chat.Update(historyMsg{peer: msg.History.Peer, messages: msgs})
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
	case screenAdminLogin:
		m.adminLogin, cmd = m.adminLogin.Update(msg)
	case screenAdminPanel:
		m.adminPanel, cmd = m.adminPanel.Update(msg)
	}
	return m, cmd
}

func (m Model) connectChat(creds client.Credentials, from screen) (Model, tea.Cmd) {
	chatConn := client.NewChatClient(m.cfg.TCPAddr)
	if err := chatConn.Connect(creds.Token); err != nil {
		switch from {
		case screenRegister:
			m.register = newRegisterModel(m.cfg.Auth)
			m.register.err = err
			m.screen = screenRegister
			return m, m.register.Init()
		default:
			m.login = newLoginModel(m.cfg.Auth)
			m.login.err = err
			m.screen = screenLogin
			return m, m.login.Init()
		}
	}

	m.chatConn = chatConn
	m.chat = newChatModel(creds.Username, chatConn)
	if m.width > 0 && m.height > 0 {
		m.chat, _ = m.chat.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
	}
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
	w := m.width
	if w == 0 {
		w = 80
	}
	switch m.screen {
	case screenWelcome:
		return m.welcome.View() + "\n" + welcomeHUD(w)
	case screenLogin:
		return m.login.View() + "\n" + authHUD(w)
	case screenRegister:
		return m.register.View() + "\n" + authHUD(w)
	case screenChat:
		return m.chat.View()
	case screenAdminLogin:
		return m.adminLogin.View() + "\n" + authHUD(w)
	case screenAdminPanel:
		return m.adminPanel.View()
	default:
		return ""
	}
}

func Run(cfg Config) error {
	p := tea.NewProgram(New(cfg), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
