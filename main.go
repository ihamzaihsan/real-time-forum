package main

import (
	"RTF/database"
	"RTF/handlers"
	"RTF/routes"
	"context"
	"crypto/tls"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func newHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/favicon.svg", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "frontend/favicon.svg") })
	mux.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("frontend/css"))))
	mux.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("frontend/js"))))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := database.DBInstance.DB.PingContext(ctx); err != nil {
			http.Error(w, "Database unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{\"status\":\"ok\"}\n"))
	})
	routes.Register(mux)
	return handlers.Middleware(mux)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		log.Fatal("PORT must be between 1 and 65535")
	}
	if os.Getenv("TLS_CERT_FILE") != "" {
		os.Setenv("COOKIE_SECURE", "true")
	}
	origin := os.Getenv("PUBLIC_ORIGIN")
	if origin != "" {
		parsed, err := url.Parse(origin)
		if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path != "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
			log.Fatal("PUBLIC_ORIGIN must be an http(s) origin without a path")
		}
		if parsed.Scheme == "https" && os.Getenv("COOKIE_SECURE") != "true" {
			log.Fatal("Set COOKIE_SECURE=true with an HTTPS PUBLIC_ORIGIN")
		}
	}
	if value := os.Getenv("COOKIE_SECURE"); value != "" && value != "true" && value != "false" {
		log.Fatal("COOKIE_SECURE must be true or false")
	}
	if value := os.Getenv("FORUM_PREMODERATE"); value != "" && value != "true" && value != "false" {
		log.Fatal("FORUM_PREMODERATE must be true or false")
	}
	for _, value := range strings.Split(os.Getenv("ADMIN_USER_IDS"), ",") {
		if value != "" {
			if id, err := strconv.Atoi(strings.TrimSpace(value)); err != nil || id < 1 {
				log.Fatal("ADMIN_USER_IDS must contain comma-separated positive IDs")
			}
		}
	}
	cert, key := os.Getenv("TLS_CERT_FILE"), os.Getenv("TLS_KEY_FILE")
	if (cert == "") != (key == "") {
		log.Fatal("TLS_CERT_FILE and TLS_KEY_FILE must be configured together")
	}
	if err := database.InitDB(); err != nil {
		log.Fatal(err)
	}
	defer database.DBInstance.DB.Close()
	for _, value := range strings.Split(os.Getenv("ADMIN_USER_IDS"), ",") {
		if strings.TrimSpace(value) == "" {
			continue
		}
		var exists bool
		if err := database.DBInstance.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id=?)", strings.TrimSpace(value)).Scan(&exists); err != nil || !exists {
			log.Fatal("ADMIN_USER_IDS must refer to existing trusted accounts; register first or use cmd/admin")
		}
	}
	handlers.ReconcileUploads()
	server := newServer(":"+port, newHandler())
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			log.Printf("shutdown: %v", err)
		}
		handlers.CloseConnections()
	}()
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				handlers.ReconcileUploads()
				if _, err := database.DBInstance.DB.Exec("DELETE FROM oauth_attempts WHERE julianday(expires_at)<=julianday('now'); DELETE FROM oauth_pending WHERE julianday(expires_at)<=julianday('now')"); err != nil {
					log.Printf("OAuth cleanup: %v", err)
				}
				if _, err := database.DBInstance.DB.Exec("DELETE FROM sessions WHERE julianday(expires_at)<=julianday('now')"); err != nil {
					log.Printf("session cleanup: %v", err)
				}
			}
		}
	}()
	scheme := "http"
	if cert != "" {
		scheme = "https"
	}
	log.Printf("Yaplane Live listening on %s://localhost:%s", scheme, port)
	if cert != "" {
		err = server.ListenAndServeTLS(cert, key)
	} else {
		err = server.ListenAndServe()
	}
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
	<-shutdownDone
}

func newServer(addr string, handler http.Handler) *http.Server {
	server := &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 60 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 * 1024}
	server.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12, CipherSuites: []uint16{
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256, tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384, tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256, tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256}}
	return server
}
