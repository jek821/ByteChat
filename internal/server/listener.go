package server

import (
	"crypto/tls"

	"ByteChat/internal/logx"
)

func ListenTCP(addr string, tlsConfig *tls.Config, hub *Hub) error {
	ln, err := tls.Listen("tcp", addr, tlsConfig)
	if err != nil {
		return err
	}
	logx.Info(logx.CatServer, "TCP+TLS listening on %s", addr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go hub.handleConn(conn)
	}
}
