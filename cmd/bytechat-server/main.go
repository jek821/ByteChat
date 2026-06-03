package main

import (
	"flag"
	"log"
	"net/http"

	"ByteChat/internal/paths"
	"ByteChat/internal/router"
	"ByteChat/internal/server"
	"ByteChat/internal/service"
	"ByteChat/internal/store/sqlite"
)

func main() {
	httpsAddr := flag.String("https-addr", ":8443", "HTTPS listen address")
	tcpAddr := flag.String("tcp-addr", ":8444", "TCP+TLS listen address")
	flag.Parse()

	dbPath, err := paths.DBPath()
	if err != nil {
		log.Fatalf("db path: %v", err)
	}

	store, err := sqlite.New(dbPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer store.Close()

	auth := service.NewAuthService(store)
	messages := service.NewMessageService(store)
	hub := server.NewHub(messages)

	tlsConfig, err := server.LoadTLSConfig()
	if err != nil {
		log.Fatalf("tls config: %v", err)
	}
	certPath, keyPath, err := server.CertFiles()
	if err != nil {
		log.Fatalf("cert files: %v", err)
	}

	go func() {
		if err := server.ListenTCP(*tcpAddr, tlsConfig, hub); err != nil {
			log.Fatalf("tcp server: %v", err)
		}
	}()

	srv := &http.Server{
		Addr:      *httpsAddr,
		Handler:   router.New(auth),
		TLSConfig: tlsConfig,
	}

	log.Printf("byteChat HTTPS server listening on https://localhost%s", *httpsAddr)
	log.Fatal(srv.ListenAndServeTLS(certPath, keyPath))
}
