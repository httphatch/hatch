package certs

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

// GenerateCA creates a new ECDSA P-256 root CA and writes the certificate
// and private key as PEM files to the paths in CAPaths. Parent directories
// are created as needed.
func GenerateCA(paths CAPaths) error {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generating CA key: %w", err)
	}

	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return fmt.Errorf("generating serial number: %w", err)
	}

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   CACommonName,
			Organization: []string{CAOrg},
		},
		NotBefore:             now,
		NotAfter:              now.AddDate(CAValidYears, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return fmt.Errorf("creating CA certificate: %w", err)
	}

	// Ensure parent directories exist for both files.
	for _, p := range []string{paths.Cert, paths.Key} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return fmt.Errorf("creating directory for %s: %w", p, err)
		}
	}

	// Write certificate PEM.
	certFile, err := os.OpenFile(paths.Cert, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("creating cert file: %w", err)
	}
	defer func() { _ = certFile.Close() }()
	if err := pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return fmt.Errorf("writing cert PEM: %w", err)
	}

	// Write private key PEM.
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return fmt.Errorf("marshaling EC key: %w", err)
	}
	keyFile, err := os.OpenFile(paths.Key, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("creating key file: %w", err)
	}
	defer func() { _ = keyFile.Close() }()
	if err := pem.Encode(keyFile, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}); err != nil {
		return fmt.Errorf("writing key PEM: %w", err)
	}

	return nil
}

// LoadCA reads the PEM-encoded certificate and private key from disk and
// returns the parsed certificate and ECDSA private key.
func LoadCA(paths CAPaths) (*x509.Certificate, *ecdsa.PrivateKey, error) {
	certPEM, err := os.ReadFile(paths.Cert)
	if err != nil {
		return nil, nil, fmt.Errorf("reading CA cert: %w", err)
	}
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, nil, fmt.Errorf("no PEM block found in %s", paths.Cert)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing CA cert: %w", err)
	}

	keyPEM, err := os.ReadFile(paths.Key)
	if err != nil {
		return nil, nil, fmt.Errorf("reading CA key: %w", err)
	}
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, nil, fmt.Errorf("no PEM block found in %s", paths.Key)
	}
	key, err := x509.ParseECPrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing CA key: %w", err)
	}

	return cert, key, nil
}

// GenerateIntermediateCA creates a new ECDSA P-256 intermediate CA signed by
// the root CA. The intermediate cert and key are written as PEM files to the
// paths in CAPaths. The root CA must already exist.
func GenerateIntermediateCA(paths CAPaths) error {
	rootCert, rootKey, err := LoadCA(paths)
	if err != nil {
		return fmt.Errorf("loading root CA: %w", err)
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generating intermediate key: %w", err)
	}

	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return fmt.Errorf("generating serial number: %w", err)
	}

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   IntermediateCACommonName,
			Organization: []string{CAOrg},
		},
		NotBefore:             now,
		NotAfter:              now.AddDate(CAValidYears, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
		MaxPathLenZero:        true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, rootCert, &key.PublicKey, rootKey)
	if err != nil {
		return fmt.Errorf("creating intermediate certificate: %w", err)
	}

	// Ensure parent directories exist for both files.
	for _, p := range []string{paths.IntermediateCert, paths.IntermediateKey} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return fmt.Errorf("creating directory for %s: %w", p, err)
		}
	}

	// Write certificate PEM.
	certFile, err := os.OpenFile(paths.IntermediateCert, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("creating intermediate cert file: %w", err)
	}
	defer func() { _ = certFile.Close() }()
	if err := pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return fmt.Errorf("writing intermediate cert PEM: %w", err)
	}

	// Write private key PEM.
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return fmt.Errorf("marshaling intermediate EC key: %w", err)
	}
	keyFile, err := os.OpenFile(paths.IntermediateKey, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("creating intermediate key file: %w", err)
	}
	defer func() { _ = keyFile.Close() }()
	if err := pem.Encode(keyFile, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}); err != nil {
		return fmt.Errorf("writing intermediate key PEM: %w", err)
	}

	return nil
}

// CAExists reports whether both the CA certificate and key files exist.
func CAExists(paths CAPaths) bool {
	for _, p := range []string{paths.Cert, paths.Key} {
		if _, err := os.Stat(p); err != nil {
			return false
		}
	}
	return true
}

// IntermediateCAExists reports whether both the intermediate CA certificate
// and key files exist.
func IntermediateCAExists(paths CAPaths) bool {
	for _, p := range []string{paths.IntermediateCert, paths.IntermediateKey} {
		if _, err := os.Stat(p); err != nil {
			return false
		}
	}
	return true
}
