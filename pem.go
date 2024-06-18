package froach

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

// readPrivateKeyFile returns a private key from a given PEM file
func readPrivateKeyFile(fn string) (crypto.PrivateKey, error) {
	b, err := readPemFile(fn, "PRIVATE KEY")
	if err != nil {
		return nil, err
	}

	return x509.ParsePKCS8PrivateKey(b.Bytes)
}

// readCertificateFile returns a certificate from a given PEM file
func readCertificateFile(fn string) (*x509.Certificate, error) {
	b, err := readPemFile(fn, "CERTIFICATE")
	if err != nil {
		return nil, err
	}

	return x509.ParseCertificate(b.Bytes)
}

// readPemFile reads a PEM file and returns the first block found of the given type
func readPemFile(fn, typ string) (*pem.Block, error) {
	dat, err := os.ReadFile(fn)
	if err != nil {
		return nil, err
	}

	for {
		var b *pem.Block
		b, dat = pem.Decode(dat)
		if b == nil {
			break
		}
		if b.Type == typ {
			return b, nil
		}
	}

	return nil, fmt.Errorf("failed to parse PEM file %s: %s not found", typ, fn)
}
