package server

import (
	"context"
	"errors"
	"net"
	"sync"

	"ByteChat/internal/logx"
	"ByteChat/internal/protocol"
	"ByteChat/internal/service"
)

type Hub struct {
	mu       sync.RWMutex
	clients  map[int64]*clientConn
	messages *service.MessageService
}

func NewHub(messages *service.MessageService) *Hub {
	return &Hub{
		clients:  make(map[int64]*clientConn),
		messages: messages,
	}
}

func (h *Hub) register(c *clientConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if existing, ok := h.clients[c.userID]; ok {
		existing.close()
	}
	h.clients[c.userID] = c
}

func (h *Hub) unregister(userID int64, c *clientConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if current, ok := h.clients[userID]; ok && current == c {
		delete(h.clients, userID)
	}
}

func (h *Hub) clientForUser(userID int64) *clientConn {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.clients[userID]
}

func (h *Hub) OnlineUsernames() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	names := make([]string, 0, len(h.clients))
	for _, c := range h.clients {
		names = append(names, c.username)
	}
	return names
}

func (h *Hub) handleConn(raw net.Conn) {
	conn := newClientConn(raw, h)
	defer conn.close()

	ctx := context.Background()
	if err := conn.authenticate(ctx); err != nil {
		logx.Warn(logx.CatTCP, "connection authentication failed")
		return
	}

	h.register(conn)
	logx.TCPConnected(conn.username)
	defer func() {
		logx.TCPDisconnected(conn.username)
		h.unregister(conn.userID, conn)
	}()

	if err := conn.sendBootstrap(ctx); err != nil {
		logx.Warn(logx.CatTCP, "bootstrap failed username=%s err=%v", conn.username, err)
		return
	}

	for {
		pkt, err := protocol.Read(conn)
		if err != nil {
			return
		}
		if err := h.handlePacket(ctx, conn, pkt); err != nil {
			logx.Warn(logx.CatTCP, "handler error username=%s err=%v", conn.username, err)
		}
	}
}

func (h *Hub) handlePacket(ctx context.Context, conn *clientConn, pkt protocol.Packet) error {
	switch pkt.Type {
	case protocol.SEND_MESSAGE:
		var msg protocol.SendMessage
		if err := protocol.UnmarshalData(pkt.Data, &msg); err != nil {
			return err
		}
		return h.deliver(ctx, conn, msg)
	case protocol.FRIEND_REQUEST:
		var req protocol.FriendRequest
		if err := protocol.UnmarshalData(pkt.Data, &req); err != nil {
			return err
		}
		return h.handleFriendRequest(ctx, conn, req)
	case protocol.ACCEPT_FRIEND_REQUEST:
		var req protocol.AcceptFriendRequest
		if err := protocol.UnmarshalData(pkt.Data, &req); err != nil {
			return err
		}
		return h.handleAcceptFriendRequest(ctx, conn, req)
	case protocol.REQUEST_HISTORY:
		var req protocol.HistoryRequest
		if err := protocol.UnmarshalData(pkt.Data, &req); err != nil {
			return err
		}
		return h.handleHistoryRequest(ctx, conn, req)
	default:
		return nil
	}
}

func (h *Hub) deliver(ctx context.Context, sender *clientConn, msg protocol.SendMessage) error {
	msgID, toUserID, err := h.messages.Send(ctx, sender.userID, msg.ToUsername, msg.Body)
	if err != nil {
		return err
	}

	recipient := h.clientForUser(toUserID)
	if recipient == nil {
		logx.MessageSent(sender.username, msg.ToUsername, msgID)
		return nil
	}

	out := protocol.ReceiveMessage{
		FromUsername: sender.username,
		Body:         msg.Body,
		MessageID:    msgID,
	}
	if err := recipient.writePacket(protocol.RECEIVE_MESSSAGE, out); err != nil {
		return err
	}
	logx.MessageSent(sender.username, msg.ToUsername, msgID)
	if err := h.messages.MarkDelivered(ctx, msgID); err != nil {
		return err
	}
	logx.MessageDelivered(msgID, msg.ToUsername)
	return nil
}

func (h *Hub) handleFriendRequest(ctx context.Context, sender *clientConn, req protocol.FriendRequest) error {
	toUserID, err := h.messages.SendFriendRequest(ctx, sender.userID, req.ToUsername)
	if err != nil {
		return err
	}

	h.refreshContacts(ctx, sender.userID)
	logx.FriendRequest(sender.username, req.ToUsername)

	recipient := h.clientForUser(toUserID)
	if recipient == nil {
		return nil
	}

	return recipient.writePacket(protocol.FRIEND_REQUEST_RECEIVED, protocol.FriendRequestReceived{
		FromUsername: sender.username,
	})
}

func (h *Hub) handleHistoryRequest(ctx context.Context, conn *clientConn, req protocol.HistoryRequest) error {
	entries, err := h.messages.GetConversationHistory(ctx, conn.userID, conn.username, req.PeerUsername)
	if err != nil {
		return err
	}

	msgs := make([]protocol.HistoryMessage, len(entries))
	for i, e := range entries {
		msgs[i] = protocol.HistoryMessage{
			FromUsername: e.FromUsername,
			Body:         e.Body,
			CreatedAt:    e.CreatedAt,
		}
	}
	return conn.writePacket(protocol.HISTORY_RESPONSE, protocol.HistoryResponse{
		PeerUsername: req.PeerUsername,
		Messages:     msgs,
	})
}

func (h *Hub) handleAcceptFriendRequest(ctx context.Context, accepter *clientConn, req protocol.AcceptFriendRequest) error {
	fromUserID, err := h.messages.AcceptFriendRequest(ctx, accepter.userID, req.FromUsername)
	if err != nil {
		return err
	}
	logx.FriendAccepted(accepter.username, req.FromUsername)

	h.refreshContacts(ctx, accepter.userID)
	h.refreshContacts(ctx, fromUserID)
	return nil
}

func (h *Hub) refreshContacts(ctx context.Context, userID int64) {
	client := h.clientForUser(userID)
	if client == nil {
		return
	}
	if err := client.sendContacts(ctx); err != nil {
		logx.Warn(logx.CatTCP, "refresh contacts failed user_id=%d err=%v", userID, err)
	}
}

type clientConn struct {
	conn     net.Conn
	hub      *Hub
	userID   int64
	username string
	mu       sync.Mutex
	closed   bool
}

func newClientConn(conn net.Conn, hub *Hub) *clientConn {
	return &clientConn{conn: conn, hub: hub}
}

func (c *clientConn) authenticate(ctx context.Context) error {
	pkt, err := protocol.Read(c.conn)
	if err != nil {
		return err
	}
	if pkt.Type != protocol.REQUEST_AUTH {
		return errors.New("expected auth request")
	}

	var req protocol.AuthRequest
	if err := protocol.UnmarshalData(pkt.Data, &req); err != nil {
		return err
	}

	userID, username, err := c.hub.messages.AuthenticateToken(ctx, req.Token)
	if err != nil {
		resp, _ := protocol.MarshalData(protocol.AuthResponse{OK: false, Error: "invalid token"})
		_ = protocol.Write(c.conn, protocol.Packet{Type: protocol.AUTH_RESPONSE, Data: resp})
		return err
	}

	c.userID = userID
	c.username = username

	resp, err := protocol.MarshalData(protocol.AuthResponse{
		OK:       true,
		UserID:   userID,
		Username: username,
	})
	if err != nil {
		return err
	}
	return protocol.Write(c.conn, protocol.Packet{Type: protocol.AUTH_RESPONSE, Data: resp})
}

func (c *clientConn) sendBootstrap(ctx context.Context) error {
	if err := c.sendContacts(ctx); err != nil {
		return err
	}

	pending, err := c.hub.messages.PendingMessages(ctx, c.userID)
	if err != nil {
		return err
	}
	for _, msg := range pending {
		out := protocol.ReceiveMessage{
			FromUsername: msg.FromUsername,
			Body:         msg.Body,
			MessageID:    msg.ID,
		}
		if err := c.writePacket(protocol.RECEIVE_MESSSAGE, out); err != nil {
			return err
		}
		if err := c.hub.messages.MarkDelivered(ctx, msg.ID); err != nil {
			return err
		}
	}
	return nil
}

func (c *clientConn) sendContacts(ctx context.Context) error {
	contacts, err := c.hub.messages.ListContacts(ctx, c.userID)
	if err != nil {
		return err
	}
	return c.writePacket(protocol.CONTACTS_RESPONSE, protocol.ContactsResponse{
		Friends:          contacts.Friends,
		PendingRequests:  contacts.PendingRequests,
		OutgoingRequests: contacts.OutgoingRequests,
	})
}

func (c *clientConn) writePacket(code protocol.Code, payload any) error {
	data, err := protocol.MarshalData(payload)
	if err != nil {
		return err
	}
	return c.writeRaw(protocol.Packet{Type: code, Data: data})
}

func (c *clientConn) writeRaw(pkt protocol.Packet) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return net.ErrClosed
	}
	return protocol.Write(c.conn, pkt)
}

func (c *clientConn) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	c.closed = true
	_ = c.conn.Close()
}

func (c *clientConn) Read(p []byte) (int, error) {
	return c.conn.Read(p)
}
