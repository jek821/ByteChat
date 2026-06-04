package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"strings"

	"ByteChat/internal/logx"
	"ByteChat/internal/paths"
	"ByteChat/internal/router"
	"ByteChat/internal/server"
	"ByteChat/internal/service"
	"ByteChat/internal/store/sqlite"
)

func main() {
	httpsAddr := flag.String("https-addr", ":8443", "HTTPS listen address")
	tcpAddr := flag.String("tcp-addr", ":8444", "TCP+TLS listen address")
	createAdmin := flag.String("create-admin", "", "create or promote admin user (format: username:password)")
	tlsCert := flag.String("tls-cert", "", "path to TLS certificate PEM (e.g. Let's Encrypt fullchain.pem)")
	tlsKey := flag.String("tls-key", "", "path to TLS private key PEM")
	tlsHostname := flag.String("tls-hostname", "", "hostname or IP for auto-generated cert (when not using -tls-cert)")
	flag.Parse()

	if err := logx.Init(); err != nil {
		log.Fatalf("log config: %v", err)
	}

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
	admin := service.NewAdminService(store, auth, hub)

	if *createAdmin != "" {
		username, password, ok := strings.Cut(*createAdmin, ":")
		if !ok || username == "" || password == "" {
			log.Fatal("create-admin format must be username:password")
		}
		if err := admin.CreateAdmin(context.Background(), username, password); err != nil {
			log.Fatalf("create admin: %v", err)
		}
		logx.AdminAction("bootstrap", "admin user ready username="+username)
		log.Printf("admin user %q is ready", username)
	}

	tlsOpts := server.TLSOptions{
		CertPath: *tlsCert,
		KeyPath:  *tlsKey,
		Hostname: *tlsHostname,
	}
	tlsConfig, err := server.LoadTLSConfig(tlsOpts)
	if err != nil {
		log.Fatalf("tls config: %v", err)
	}
	certPath := *tlsCert
	keyPath := *tlsKey
	if certPath == "" || keyPath == "" {
		certPath, keyPath, err = server.CertFiles()
		if err != nil {
			log.Fatalf("cert files: %v", err)
		}
	}

	go func() {
		if err := server.ListenTCP(*tcpAddr, tlsConfig, hub); err != nil {
			log.Fatalf("tcp server: %v", err)
		}
	}()

	srv := &http.Server{
		Addr:      *httpsAddr,
		Handler:   router.New(auth, admin),
		TLSConfig: tlsConfig,
	}

	logx.Info(logx.CatServer, "HTTPS listening on %s", *httpsAddr)
	log.Fatal(srv.ListenAndServeTLS(certPath, keyPath))
}
