package protocol

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
)

const MaxFrameSize = 1 << 20 // 1 MiB

var (
	ErrFrameTooLarge = errors.New("frame exceeds maximum size")
	ErrShortFrame    = errors.New("frame shorter than header")
)

func Write(w io.Writer, pkt Packet) error {
	payload, err := json.Marshal(pkt)
	if err != nil {
		return err
	}
	if len(payload) > MaxFrameSize {
		return ErrFrameTooLarge
	}

	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, uint32(len(payload)))
	if _, err := w.Write(header); err != nil {
		return err
	}
	_, err = w.Write(payload)
	return err
}

func Read(r io.Reader) (Packet, error) {
	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return Packet{}, err
	}

	size := binary.BigEndian.Uint32(header[:])
	if size == 0 {
		return Packet{}, ErrShortFrame
	}
	if size > MaxFrameSize {
		return Packet{}, ErrFrameTooLarge
	}

	payload := make([]byte, size)
	if _, err := io.ReadFull(r, payload); err != nil {
		return Packet{}, err
	}

	var pkt Packet
	if err := json.Unmarshal(payload, &pkt); err != nil {
		return Packet{}, err
	}
	return pkt, nil
}

func MarshalData(v any) (json.RawMessage, error) {
	return json.Marshal(v)
}

func UnmarshalData(data json.RawMessage, v any) error {
	return json.Unmarshal(data, v)
}
