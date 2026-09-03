package proxy

import (
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"

	"example.com/brave-revival/src/cert"
)

const connectTimeout = 15 * time.Second

type Config struct {
	Listen string `json:"listen"`
	Domain string `json:"domain"`
	TLSDir string `json:"tls_dir"`
}

type handler struct {
	config  *Config
	tlsCert tls.Certificate
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	slog.Info("received request", "method", r.Method, "url", r.URL.String(), "remote_addr", r.RemoteAddr)

	if r.Method == http.MethodConnect {
		if err := h.handleTLS(w, r); err != nil {
			slog.Error("failed to handle TLS connection", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	switch r.URL.Path {
	case "/proxy.pac":
		w.Header().Set("Content-Type", "application/x-ns-proxy-autoconfig")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `function FindProxyForURL(url,host){return host=="%s"?"PROXY %s":"DIRECT";}`, h.config.Domain, r.Host)

	case "/ca.crt":
		caBytes, err := cert.LoadCA(h.config.TLSDir, h.config.Domain)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Failed to load CA certificate: %v\n", err)
			return
		}
		w.Header().Set("Content-Type", "application/x-pem-file")
		w.Header().Set("Content-Disposition", `attachment; filename="ca.crt"`)
		w.WriteHeader(http.StatusOK)
		w.Write(caBytes)

	default:
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "404 Not Found\n")
	}
}

func (h *handler) handleTLS(w http.ResponseWriter, r *http.Request) error {
	hj, ok := w.(http.Hijacker)
	if !ok {
		return fmt.Errorf("response writer does not support hijacking")
	}

	clientConn, buf, err := hj.Hijack()
	if err != nil {
		return fmt.Errorf("failed to hijack connection: %w", err)
	}

	_, _ = buf.WriteString("HTTP/1.1 200 Connection Established\r\n\r\n")
	_ = buf.Flush()

	tlsConn := tls.Server(clientConn, &tls.Config{
		Certificates: []tls.Certificate{h.tlsCert},
	})
	if err := tlsConn.Handshake(); err != nil {
		_ = tlsConn.Close()
		return fmt.Errorf("TLS handshake failed: %w", err)
	}

	httpServer := &http.Server{
		Handler: http.HandlerFunc(func(ww http.ResponseWriter, rr *http.Request) {
			ww.Header().Set("Content-Type", "text/plain; charset=utf-8")
			ww.WriteHeader(http.StatusOK)
			fmt.Fprintf(ww, "Hello %q!\n", rr.URL.String())
		}),
	}
	if err := httpServer.Serve(&singleConnListener{conn: tlsConn}); err != nil && err != io.EOF {
		return fmt.Errorf("failed to serve HTTPS connection: %w", err)
	}

	return nil
}

type singleConnListener struct {
	conn net.Conn
}

func (l *singleConnListener) Accept() (net.Conn, error) {
	if l.conn == nil {
		return nil, io.EOF
	}
	conn := l.conn
	l.conn = nil
	return conn, nil
}

func (l *singleConnListener) Close() error {
	if l.conn != nil {
		return l.conn.Close()
	}
	return nil
}

func (l *singleConnListener) Addr() net.Addr {
	if l.conn != nil {
		return l.conn.LocalAddr()
	}
	return nil
}

func Run(config *Config) error {
	tlsCert, err := cert.LoadSelfSignedCert(config.TLSDir, config.Domain)
	if err != nil {
		return fmt.Errorf("failed to load self-signed certificate: %w", err)
	}

	server := http.Server{
		Addr: config.Listen,
		Handler: &handler{
			config:  config,
			tlsCert: tlsCert,
		},
	}
	return server.ListenAndServe()
}
