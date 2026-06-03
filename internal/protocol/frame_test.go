package protocol

import (
	"bytes"
	"testing"
)

func TestWriteReadRoundTrip(t *testing.T) {
	original := Packet{
		Type: SEND_MESSAGE,
		Data: mustMarshal(t, SendMessage{ToUsername: "alice", Body: "hello"}),
	}

	var buf bytes.Buffer
	if err := Write(&buf, original); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got, err := Read(&buf)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.Type != original.Type {
		t.Fatalf("type: got %d want %d", got.Type, original.Type)
	}

	var msg SendMessage
	if err := UnmarshalData(got.Data, &msg); err != nil {
		t.Fatalf("UnmarshalData: %v", err)
	}
	if msg.ToUsername != "alice" || msg.Body != "hello" {
		t.Fatalf("unexpected message: %+v", msg)
	}
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	data, err := MarshalData(v)
	if err != nil {
		t.Fatalf("MarshalData: %v", err)
	}
	return data
}
