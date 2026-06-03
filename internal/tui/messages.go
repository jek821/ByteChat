package tui

import "ByteChat/internal/client"

type screen int

const (
	screenWelcome screen = iota
	screenLogin
	screenRegister
	screenChat
)

type navigateMsg struct {
	to screen
}

type loginSuccessMsg struct {
	creds client.Credentials
}

type registerSuccessMsg struct {
	creds client.Credentials
}

type authErrorMsg struct {
	err error
}

type incomingMessageMsg struct {
	from string
	body string
}

type contactsUpdatedMsg struct {
	contacts []string
}

type chatDisconnectedMsg struct{}
