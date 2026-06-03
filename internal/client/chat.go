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
)

type ChatEvent struct {
	Kind     ChatEventKind
	Message  Message
	Contacts []string
}

type Message struct {
	From string
	Body string
}

type ChatClient struct {
	addr      string
	tlsConfig *tls.Config
	conn      net.Conn
	events    chan ChatEvent
	done      chan struct{}
	mu        sync.Mutex
}

func NewChatClient(addr string) *ChatClient {
	return &ChatClient{
		addr: addr,
		tlsConfig: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: true,
		},
		events: make(chan ChatEvent, 32),
		done:   make(chan struct{}),
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
	c.mu.Unlock()

	go c.readLoop()
	return nil
}

func (c *ChatClient) Send(toUsername, body string) error {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		return errors.New("not connected")
	}

	data, err := protocol.MarshalData(protocol.SendMessage{
		ToUsername: toUsername,
		Body:       body,
	})
	if err != nil {
		return err
	}
	return protocol.Write(conn, protocol.Packet{Type: protocol.SEND_MESSAGE, Data: data})
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
			select {
			case c.events <- ChatEvent{
				Kind: EventMessage,
				Message: Message{
					From: msg.FromUsername,
					Body: msg.Body,
				},
			}:
			case <-c.done:
				return
			}
		case protocol.CONTACTS_RESPONSE:
			var contacts protocol.ContactsResponse
			if err := protocol.UnmarshalData(pkt.Data, &contacts); err != nil {
				continue
			}
			select {
			case c.events <- ChatEvent{Kind: EventContacts, Contacts: contacts.Usernames}:
			case <-c.done:
				return
			}
		}
	}
}
