package froach

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

func readPrivateKeyFile(fn string) (crypto.PrivateKey, error) {
	b, err := readPemFile(fn)
	if err != nil {
		return nil, err
	}
	if b.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("invalid PEM data type: %s", b.Type)
	}

	return x509.ParsePKCS8PrivateKey(b.Bytes)
}

func readCertificateFile(fn string) (*x509.Certificate, error) {
	b, err := readPemFile(fn)
	if err != nil {
		return nil, err
	}
	if b.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("invalid PEM data type: %s", b.Type)
	}

	return x509.ParseCertificate(b.Bytes)
}

func readPemFile(fn string) (*pem.Block, error) {
	dat, err := os.ReadFile(fn)
	if err != nil {
		return nil, err
	}

	b, _ := pem.Decode(dat)
	if b == nil {
		return nil, errors.New("malformed PEM file")
	}

	return b, nil
}
