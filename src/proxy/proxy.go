package proxy

import (
	"crypto/tls"
	_ "embed"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"time"

	"example.com/brave-revival/src/cert"
	"example.com/brave-revival/src/config"
	"example.com/brave-revival/src/www"
)

const connectTimeout = 15 * time.Second

//go:embed templates/proxy.pac
var proxyPACTemplate string

//go:embed templates/singbox.json
var singBoxJSONTemplate string

type handler struct {
	config  *config.Config
	tlsCert tls.Certificate
	www     *www.Handler
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		if err := h.handleTLS(w, r); err != nil {
			slog.Error("failed to handle TLS connection", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	slog.Info("handling direct request", "method", r.Method, "url", &r.URL, "remote_addr", r.RemoteAddr)

	switch r.URL.Path {
	case "/proxy.pac":
		w.Header().Set("Content-Type", "application/x-ns-proxy-autoconfig")
		w.WriteHeader(http.StatusOK)
		host := net.JoinHostPort(h.config.AdvertiseHost, fmt.Sprint(h.config.ProxyPort))
		fmt.Fprintf(w, proxyPACTemplate, host)

	case "/ca.crt":
		caBytes, err := cert.LoadCA(h.config.TLSDir)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Failed to load CA certificate: %v\n", err)
			return
		}
		w.Header().Set("Content-Type", "application/x-pem-file")
		w.Header().Set("Content-Disposition", `attachment; filename="ca.crt"`)
		w.WriteHeader(http.StatusOK)
		w.Write(caBytes)

	case "/singbox.json":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, singBoxJSONTemplate, h.config.AdvertiseHost, h.config.ProxyPort)

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
		Handler: h.www,
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

func Run(cfg *config.Config) error {
	tlsCert, err := cert.LoadSelfSignedCert(cfg.TLSDir)
	if err != nil {
		return fmt.Errorf("failed to load self-signed certificate: %w", err)
	}

	wwwHandler, err := www.NewHandler(cfg)
	if err != nil {
		return fmt.Errorf("failed to create www handler: %w", err)
	}

	server := http.Server{
		Addr: netip.AddrPortFrom(cfg.Host, cfg.ProxyPort).String(),
		Handler: &handler{
			config:  cfg,
			tlsCert: tlsCert,
			www:     wwwHandler,
		},
	}
	slog.Info("starting proxy server", "addr", server.Addr, "advertise_host", cfg.AdvertiseHost, "proxy_port", cfg.ProxyPort)
	return server.ListenAndServe()
}
