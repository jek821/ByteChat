package protocol

type AuthRequest struct {
	Token string `json:"token"`
}

type AuthResponse struct {
	OK       bool   `json:"ok"`
	UserID   int64  `json:"user_id,omitempty"`
	Username string `json:"username,omitempty"`
	Error    string `json:"error,omitempty"`
}

type SendMessage struct {
	ToUsername string `json:"to_username"`
	Body       string `json:"body"`
}

type ReceiveMessage struct {
	FromUsername string `json:"from_username"`
	Body         string `json:"body"`
	MessageID    int64  `json:"message_id"`
}

type ContactsResponse struct {
	Usernames []string `json:"usernames"`
}
