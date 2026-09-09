package cert

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

const certValidity = 365 * 24 * time.Hour // 1 year (can't be significantly longer, see https://support.apple.com/en-us/102028)

type cert struct {
	caCert []byte
	caKey  []byte
	cert   []byte
	key    []byte
}

func generateKey() (*ecdsa.PrivateKey, *big.Int, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	return key, serial, nil
}

func encodeCert(certDer []byte, key *ecdsa.PrivateKey) ([]byte, []byte, error) {
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDer})
	keyDer, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal private key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDer})
	return certPEM, keyPEM, nil
}

func generateSelfSignedCert() (*cert, error) {
	now := time.Now()

	caKey, caSerial, err := generateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate CA private key: %w", err)
	}
	caTemplate := x509.Certificate{
		SerialNumber:          caSerial,
		Subject:               pkix.Name{CommonName: "Brave Revival CA"},
		NotBefore:             now,
		NotAfter:              now.Add(certValidity + 1),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}

	caDer, err := x509.CreateCertificate(rand.Reader, &caTemplate, &caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create CA certificate: %w", err)
	}

	key, serial, err := generateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate certificate private key: %w", err)
	}

	template := x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "*.enish-games.com"},
		NotBefore:             now,
		NotAfter:              now.Add(certValidity),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"*.enish-games.com"},
	}

	certDer, err := x509.CreateCertificate(rand.Reader, &template, &caTemplate, &key.PublicKey, caKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	caCertPEM, caKeyPEM, err := encodeCert(caDer, caKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encode CA certificate and key: %w", err)
	}
	certPEM, keyPEM, err := encodeCert(certDer, key)
	if err != nil {
		return nil, fmt.Errorf("failed to encode certificate and key: %w", err)
	}

	return &cert{
		caCert: caCertPEM,
		caKey:  caKeyPEM,
		cert:   certPEM,
		key:    keyPEM,
	}, nil
}

func LoadCA(cacheDir string) ([]byte, error) {
	caCertPath := filepath.Join(cacheDir, "ca.crt")
	if content, err := os.ReadFile(caCertPath); err == nil {
		return content, nil
	}

	_, err := LoadSelfSignedCert(cacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load self-signed certificate: %w", err)
	}
	return os.ReadFile(caCertPath)
}

func LoadSelfSignedCert(cacheDir string) (tls.Certificate, error) {
	certPath := filepath.Join(cacheDir, "cert.crt")
	keyPath := filepath.Join(cacheDir, "cert.key")

	tlsCert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err == nil {
		return tlsCert, nil
	}

	certBytes, err := generateSelfSignedCert()
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to generate self-signed certificate: %w", err)
	}

	certFile, err := os.Create(certPath)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to create certificate file: %w", err)
	}
	defer certFile.Close()
	if _, err = certFile.Write(certBytes.cert); err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to write certificate file: %w", err)
	}
	if _, err = certFile.Write(certBytes.caCert); err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to write certificate file (root): %w", err)
	}

	if err := os.WriteFile(keyPath, certBytes.key, 0600); err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to write key file: %w", err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "ca.crt"), certBytes.caCert, 0644); err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to write CA certificate file: %w", err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "ca.key"), certBytes.caKey, 0600); err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to write CA key file: %w", err)
	}

	return tls.LoadX509KeyPair(certPath, keyPath)
}
