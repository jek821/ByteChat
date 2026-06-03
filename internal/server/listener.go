package server

import (
	"crypto/tls"
	"log"
)

func ListenTCP(addr string, tlsConfig *tls.Config, hub *Hub) error {
	ln, err := tls.Listen("tcp", addr, tlsConfig)
	if err != nil {
		return err
	}
	log.Printf("byteChat TCP+TLS server listening on %s", addr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go hub.handleConn(conn)
	}
}
