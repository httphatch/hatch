// Package certs handles root CA generation and macOS Keychain trust for
// local HTTPS development certificates.
package certs

import "path/filepath"

const (
	RootCACertFile           = "rootCA.pem"
	RootCAKeyFile            = "rootCA-key.pem"
	IntermediateCACertFile   = "intermediateCA.pem"
	IntermediateCAKeyFile    = "intermediateCA-key.pem"
	CACommonName             = "Hatch Local CA"
	IntermediateCACommonName = "Hatch Local CA - Intermediate"
	CAOrg                    = "Hatch"
	CAValidYears             = 10
)

// CAPaths holds the file paths for the root and intermediate CA certificates and keys.
type CAPaths struct {
	Cert             string // e.g. "~/.hatch/certs/rootCA.pem"
	Key              string // e.g. "~/.hatch/certs/rootCA-key.pem"
	IntermediateCert string // e.g. "~/.hatch/certs/intermediateCA.pem"
	IntermediateKey  string // e.g. "~/.hatch/certs/intermediateCA-key.pem"
}

// NewCAPaths returns CAPaths rooted in certsDir.
func NewCAPaths(certsDir string) CAPaths {
	return CAPaths{
		Cert:             filepath.Join(certsDir, RootCACertFile),
		Key:              filepath.Join(certsDir, RootCAKeyFile),
		IntermediateCert: filepath.Join(certsDir, IntermediateCACertFile),
		IntermediateKey:  filepath.Join(certsDir, IntermediateCAKeyFile),
	}
}
