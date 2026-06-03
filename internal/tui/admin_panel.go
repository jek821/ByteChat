package tui

import (
	"fmt"
	"strings"

	"ByteChat/internal/client"
	"ByteChat/internal/logx"
	"ByteChat/internal/service"
	"ByteChat/internal/store"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

type adminTab int

const (
	adminTabDashboard adminTab = iota
	adminTabUsers
	adminTabLogs
	adminTabWipe
)

type adminPanelModel struct {
	admin    *client.AdminClient
	username string
	width    int
	height   int

	tab     adminTab
	loading bool
	err     string
	status  string

	dash    service.AdminDashboard
	users   []store.UserSummary
	userIdx int
	pendingDelete string

	logIdx int

	wipeInput textinput.Model
	wiping    bool
}

func newAdminPanelModel(admin *client.AdminClient, username string) adminPanelModel {
	wipe := textinput.New()
	wipe.Placeholder = "type WIPE DATABASE"
	wipe.CharLimit = 32
	wipe.Width = 30
	wipe.Prompt = "> "
	return adminPanelModel{
		admin:    admin,
		username: username,
		tab:      adminTabDashboard,
		wipeInput: wipe,
	}
}

func (m adminPanelModel) Init() tea.Cmd {
	return m.refresh()
}

func (m adminPanelModel) refresh() tea.Cmd {
	admin := m.admin
	return func() tea.Msg {
		dash, err := admin.Dashboard()
		if err != nil {
			return adminDataMsg{err: err}
		}
		users, err := admin.ListUsers()
		if err != nil {
			return adminDataMsg{err: err}
		}
		return adminDataMsg{dash: dash, users: users}
	}
}

func (m adminPanelModel) Update(msg tea.Msg) (adminPanelModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case adminDataMsg:
		m.loading = false
		m.wiping = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.dash = msg.dash
		m.users = msg.users
		if m.userIdx >= len(m.users) {
			m.userIdx = max(0, len(m.users)-1)
		}
		m.err = ""
		return m, nil

	case adminActionMsg:
		m.loading = false
		m.wiping = false
		if msg.err != nil {
			m.err = msg.err.Error()
			m.status = ""
		} else {
			m.err = ""
			m.status = msg.status
		}
		if msg.refresh {
			m.loading = true
			return m, m.refresh()
		}
		return m, nil

	case tea.KeyMsg:
		if m.loading || m.wiping {
			if msg.String() == "esc" {
				return m, func() tea.Msg { return navigateMsg{to: screenWelcome} }
			}
			return m, nil
		}
		switch msg.String() {
		case "esc":
			if m.tab == adminTabUsers && m.pendingDelete != "" {
				m.pendingDelete = ""
				m.status = ""
				return m, nil
			}
			return m, func() tea.Msg { return navigateMsg{to: screenWelcome} }
		case "q":
			if m.tab == adminTabUsers && m.pendingDelete != "" {
				m.pendingDelete = ""
				return m, nil
			}
			return m, func() tea.Msg { return navigateMsg{to: screenWelcome} }
		case "1":
			m.tab = adminTabDashboard
			m.status = ""
			return m, nil
		case "2":
			m.tab = adminTabUsers
			m.status = ""
			m.pendingDelete = ""
			return m, nil
		case "3":
			m.tab = adminTabLogs
			m.status = ""
			return m, nil
		case "4":
			m.tab = adminTabWipe
			m.status = ""
			m.wipeInput.Focus()
			return m, textinput.Blink
		case "r":
			m.loading = true
			m.status = ""
			return m, m.refresh()
		}

		switch m.tab {
		case adminTabUsers:
			return m.updateUsers(msg)
		case adminTabLogs:
			return m.updateLogs(msg)
		case adminTabWipe:
			return m.updateWipe(msg)
		}
	}
	return m, nil
}

func (m adminPanelModel) updateUsers(msg tea.KeyMsg) (adminPanelModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.pendingDelete = ""
		if m.userIdx > 0 {
			m.userIdx--
		}
	case "down", "j":
		m.pendingDelete = ""
		if m.userIdx < len(m.users)-1 {
			m.userIdx++
		}
	case "d":
		if len(m.users) == 0 {
			return m, nil
		}
		target := m.users[m.userIdx].Username
		if target == m.username {
			m.err = "cannot delete your own account"
			m.pendingDelete = ""
			return m, nil
		}
		if m.pendingDelete != target {
			m.pendingDelete = target
			m.err = ""
			m.status = ""
			return m, nil
		}
		m.pendingDelete = ""
		m.loading = true
		admin := m.admin
		return m, func() tea.Msg {
			err := admin.DeleteUser(target)
			if err != nil {
				return adminActionMsg{err: err}
			}
			return adminActionMsg{status: "deleted " + target, refresh: true}
		}
	}
	return m, nil
}

func (m adminPanelModel) updateLogs(msg tea.KeyMsg) (adminPanelModel, tea.Cmd) {
	cats := logx.AllCategories
	switch msg.String() {
	case "up", "k":
		if m.logIdx > 0 {
			m.logIdx--
		}
	case "down", "j":
		if m.logIdx < len(cats)-1 {
			m.logIdx++
		}
	case " ":
		cat := cats[m.logIdx]
		enabled := !m.dash.LoggingConfig.Enabled[cat]
		m.loading = true
		admin := m.admin
		return m, func() tea.Msg {
			if err := admin.SetLogCategory(cat, enabled); err != nil {
				return adminActionMsg{err: err}
			}
			state := "disabled"
			if enabled {
				state = "enabled"
			}
			return adminActionMsg{status: string(cat) + " " + state, refresh: true}
		}
	}
	return m, nil
}

func (m adminPanelModel) updateWipe(msg tea.KeyMsg) (adminPanelModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		confirm := strings.TrimSpace(m.wipeInput.Value())
		if confirm == "" {
			return m, nil
		}
		m.wiping = true
		m.loading = true
		admin := m.admin
		return m, func() tea.Msg {
			err := admin.WipeDatabase(confirm)
			if err != nil {
				return adminActionMsg{err: err}
			}
			return adminActionMsg{status: "database wiped", refresh: true}
		}
	}
	var cmd tea.Cmd
	m.wipeInput, cmd = m.wipeInput.Update(msg)
	return m, cmd
}

func (m adminPanelModel) View() string {
	w := m.width
	if w == 0 {
		w = 80
	}

	header := headerStyle.Render("Admin · " + m.username)
	tabs := m.renderTabs()
	body := m.renderBody()

	content := lipgloss.JoinVertical(lipgloss.Left, header, tabs, body)
	if m.err != "" {
		content += "\n" + errStyle.Render(m.err)
	}
	if m.status != "" {
		content += "\n" + successStyle.Render(m.status)
	}
	if m.loading {
		content += "\n" + infoStyle.Render("Loading...")
	}

	return content + "\n" + adminHUD(w, m)
}

func (m adminPanelModel) renderTabs() string {
	labels := []string{"1 Dashboard", "2 Users", "3 Logs", "4 Wipe"}
	var parts []string
	for i, label := range labels {
		if adminTab(i) == m.tab {
			parts = append(parts, tabActiveStyle.Render(label))
		} else {
			parts = append(parts, tabInactiveStyle.Render(label))
		}
	}
	return strings.Join(parts, " ")
}

func (m adminPanelModel) renderBody() string {
	switch m.tab {
	case adminTabDashboard:
		return m.viewDashboard()
	case adminTabUsers:
		return m.viewUsers()
	case adminTabLogs:
		return m.viewLogs()
	case adminTabWipe:
		return m.viewWipe()
	default:
		return ""
	}
}

func (m adminPanelModel) viewDashboard() string {
	s := m.dash.Stats
	lines := []string{
		labelStyle.Render("Server stats"),
		fmt.Sprintf("  Users:           %d", s.UserCount),
		fmt.Sprintf("  Messages:        %d", s.MessageCount),
		fmt.Sprintf("  Sessions:        %d", s.SessionCount),
		fmt.Sprintf("  Friends:         %d", s.FriendCount),
		fmt.Sprintf("  Friend requests: %d", s.RequestCount),
		"",
		labelStyle.Render(fmt.Sprintf("Online now (%d)", m.dash.OnlineCount)),
	}
	if len(m.dash.OnlineUsers) == 0 {
		lines = append(lines, infoStyle.Render("  (none)"))
	} else {
		for _, u := range m.dash.OnlineUsers {
			lines = append(lines, "  • "+u)
		}
	}
	return chatPaneStyle.Width(60).Render(strings.Join(lines, "\n"))
}

func (m adminPanelModel) viewUsers() string {
	if len(m.users) == 0 {
		return chatPaneStyle.Width(50).Render(infoStyle.Render("No users"))
	}
	var lines []string
	for i, u := range m.users {
		line := u.Username
		if u.IsAdmin {
			line += " [admin]"
		}
		if i == m.userIdx {
			lines = append(lines, sidebarItemActiveStyle.Render("> "+line))
		} else {
			lines = append(lines, sidebarItemStyle.Render("  "+line))
		}
	}
	body := sidebarStyle.Height(min(12, len(m.users)+2)).Render(strings.Join(lines, "\n"))
	if m.pendingDelete != "" {
		body += "\n" + errStyle.Render("Delete "+m.pendingDelete+"? Press d again to confirm, esc to cancel.")
	}
	return body
}

func (m adminPanelModel) viewLogs() string {
	cats := logx.AllCategories
	var lines []string
	for i, cat := range cats {
		on := m.dash.LoggingConfig.Enabled[cat]
		state := "off"
		style := infoStyle
		if on {
			state = "on"
			style = successStyle
		}
		prefix := "  "
		if i == m.logIdx {
			prefix = "> "
		}
		lines = append(lines, prefix+string(cat)+": "+style.Render(state))
	}
	return chatPaneStyle.Width(40).Render(strings.Join(lines, "\n"))
}

func (m adminPanelModel) viewWipe() string {
	warning := errStyle.Render("⚠ This permanently deletes ALL users, messages, and sessions.")
	body := warning + "\n\n" +
		labelStyle.Render("Type WIPE DATABASE to confirm:") + "\n" +
		m.wipeInput.View()
	return chatPaneStyle.Width(50).Render(body)
}
