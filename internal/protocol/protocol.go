package protocol

import "encoding/json"

type Code uint32

const (
	NEW_USER Code = iota
	SEND_MESSAGE
	FRIEND_REQUEST
	RECEIVE_MESSSAGE
	REQUEST_AUTH
	AUTH_RESPONSE
	ACCEPT_FRIEND_REQUEST
	CONTACTS_RESPONSE
	FRIEND_REQUEST_RECEIVED
)

type Packet struct {
	Type Code            `json:"code"`
	Data json.RawMessage `json:"data"`
}
