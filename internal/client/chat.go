package client

import (
	"crypto/tls"
	"errors"
	"net"
	"sync"
	"time"

	"ByteChat/internal/protocol"
)

type ChatEventKind int

const (
	EventMessage ChatEventKind = iota
	EventContacts
	EventFriendRequest
	EventHistory
)

type ChatEvent struct {
	Kind     ChatEventKind
	Message  Message
	Contacts Contacts
	From     string
	History  History
}

type Contacts struct {
	Friends          []string
	PendingRequests  []string
	OutgoingRequests []string
}

type Message struct {
	From string
	Body string
}

type History struct {
	Peer     string
	Messages []Message
	SelfUser string
}

type ChatClient struct {
	addr      string
	tlsConfig *tls.Config
	conn      net.Conn
	events    chan ChatEvent
	done      chan struct{}
	mu        sync.Mutex
	username  string
}

func NewChatClient(addr string, insecureTLS bool) *ChatClient {
	return &ChatClient{
		addr:      addr,
		tlsConfig: TLSConfig(insecureTLS),
		events:    make(chan ChatEvent, 32),
		done:      make(chan struct{}),
	}
}

func (c *ChatClient) Connect(token string) error {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", c.addr, c.tlsConfig)
	if err != nil {
		return err
	}

	reqData, err := protocol.MarshalData(protocol.AuthRequest{Token: token})
	if err != nil {
		conn.Close()
		return err
	}
	if err := protocol.Write(conn, protocol.Packet{Type: protocol.REQUEST_AUTH, Data: reqData}); err != nil {
		conn.Close()
		return err
	}

	pkt, err := protocol.Read(conn)
	if err != nil {
		conn.Close()
		return err
	}
	if pkt.Type != protocol.AUTH_RESPONSE {
		conn.Close()
		return errors.New("expected auth response")
	}

	var resp protocol.AuthResponse
	if err := protocol.UnmarshalData(pkt.Data, &resp); err != nil {
		conn.Close()
		return err
	}
	if !resp.OK {
		conn.Close()
		if resp.Error != "" {
			return errors.New(resp.Error)
		}
		return errors.New("authentication failed")
	}

	c.mu.Lock()
	c.conn = conn
	c.username = resp.Username
	c.mu.Unlock()

	go c.readLoop()
	return nil
}

func (c *ChatClient) Username() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.username
}

func (c *ChatClient) Send(toUsername, body string) error {
	return c.writePacket(protocol.SEND_MESSAGE, protocol.SendMessage{
		ToUsername: toUsername,
		Body:       body,
	})
}

func (c *ChatClient) SendFriendRequest(toUsername string) error {
	return c.writePacket(protocol.FRIEND_REQUEST, protocol.FriendRequest{ToUsername: toUsername})
}

func (c *ChatClient) AcceptFriendRequest(fromUsername string) error {
	return c.writePacket(protocol.ACCEPT_FRIEND_REQUEST, protocol.AcceptFriendRequest{FromUsername: fromUsername})
}

func (c *ChatClient) RequestHistory(peerUsername string) error {
	return c.writePacket(protocol.REQUEST_HISTORY, protocol.HistoryRequest{PeerUsername: peerUsername})
}

func (c *ChatClient) Events() <-chan ChatEvent {
	return c.events
}

func (c *ChatClient) Close() error {
	select {
	case <-c.done:
	default:
		close(c.done)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	return err
}

func (c *ChatClient) writePacket(code protocol.Code, payload any) error {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		return errors.New("not connected")
	}

	data, err := protocol.MarshalData(payload)
	if err != nil {
		return err
	}
	return protocol.Write(conn, protocol.Packet{Type: code, Data: data})
}

func (c *ChatClient) readLoop() {
	defer close(c.events)
	for {
		select {
		case <-c.done:
			return
		default:
		}

		c.mu.Lock()
		conn := c.conn
		username := c.username
		c.mu.Unlock()
		if conn == nil {
			return
		}

		pkt, err := protocol.Read(conn)
		if err != nil {
			return
		}

		switch pkt.Type {
		case protocol.RECEIVE_MESSSAGE:
			var msg protocol.ReceiveMessage
			if err := protocol.UnmarshalData(pkt.Data, &msg); err != nil {
				continue
			}
			c.pushEvent(ChatEvent{
				Kind: EventMessage,
				Message: Message{
					From: msg.FromUsername,
					Body: msg.Body,
				},
			})
		case protocol.CONTACTS_RESPONSE:
			var contacts protocol.ContactsResponse
			if err := protocol.UnmarshalData(pkt.Data, &contacts); err != nil {
				continue
			}
			c.pushEvent(ChatEvent{
				Kind: EventContacts,
				Contacts: Contacts{
					Friends:          contacts.Friends,
					PendingRequests:  contacts.PendingRequests,
					OutgoingRequests: contacts.OutgoingRequests,
				},
			})
		case protocol.FRIEND_REQUEST_RECEIVED:
			var req protocol.FriendRequestReceived
			if err := protocol.UnmarshalData(pkt.Data, &req); err != nil {
				continue
			}
			c.pushEvent(ChatEvent{Kind: EventFriendRequest, From: req.FromUsername})
		case protocol.HISTORY_RESPONSE:
			var hist protocol.HistoryResponse
			if err := protocol.UnmarshalData(pkt.Data, &hist); err != nil {
				continue
			}
			msgs := make([]Message, len(hist.Messages))
			for i, m := range hist.Messages {
				msgs[i] = Message{From: m.FromUsername, Body: m.Body}
			}
			c.pushEvent(ChatEvent{
				Kind: EventHistory,
				History: History{
					Peer:     hist.PeerUsername,
					Messages: msgs,
					SelfUser: username,
				},
			})
		}
	}
}

func (c *ChatClient) pushEvent(event ChatEvent) {
	select {
	case c.events <- event:
	case <-c.done:
	}
}
