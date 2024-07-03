package froach

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/KarpelesLab/fleet"
)

var (
	keyLk sync.Mutex
	caKey crypto.PrivateKey
	caCrt *x509.Certificate
)

func updateKey(k string, enc []byte) {
	dec, err := x509.ParsePKCS8PrivateKey(enc)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to parse encoded private key: %s", err), "event", "froach:update_key:parse_error")
		return
	}

	err = setPrivateKey(dec)
	if err != nil {
		slog.Error(fmt.Sprintf("cockroachdb private key setting failed: %s", err), "event", "froach:update_key:fail")
	}
}

// setPrivateKey will update the private key, and generate a new matching CA. The CA will
// be different on each host (different expiration date), but will share the same CN and private
// key, so these will work everywhere.
func setPrivateKey(k crypto.PrivateKey) error {
	keyLk.Lock()
	defer keyLk.Unlock()

	key, ok := k.(crypto.Signer)
	if !ok {
		return fmt.Errorf("unsupported private key type %T (must match crypto.Signer for x509 certificate generation)", k)
	}

	keyBin, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}

	// generate pubkey hash & put it into the common name to guarantee we're not using the wrong key
	// SubjectKeyId will also be included in the CA, but that's sha1 hash
	pubKey := key.Public()
	pubKeyBin, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return fmt.Errorf("failed to marshal PKIX: %w", err)
	}
	pubHash := sha256.Sum256(pubKeyBin)

	now := time.Now()

	caSubject := pkix.Name{CommonName: "CockroachDB CA #" + base64.RawURLEncoding.EncodeToString(pubHash[:])}

	caCrtTpl := &x509.Certificate{
		BasicConstraintsValid: true,
		IsCA:                  true,
		SerialNumber:          big.NewInt(1),
		Issuer:                caSubject,
		Subject:               caSubject,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		MaxPathLen:            1,
		NotBefore:             now,
		NotAfter:              now.Add(10 * 365 * 24 * time.Hour), // + ~10 years
	}

	caCrtBin, err := x509.CreateCertificate(rand.Reader, caCrtTpl, caCrtTpl, pubKey, key)
	if err != nil {
		return fmt.Errorf("failed to create CA crt: %w", err)
	}

	// func ParseCertificate(der []byte) (*Certificate, error)
	caCrtParsed, err := x509.ParseCertificate(caCrtBin)
	if err != nil {
		return fmt.Errorf("failed to parse freshly generated CA: %w", err)
	}

	caKeyPem := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBin})
	caCrtPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCrtBin})

	// write files
	// certificates stored in ~/.config/froach and data in ~/.cache/froach
	p := basePath()
	os.MkdirAll(p, 0755)

	slog.Debug(fmt.Sprintf("[froach] writing cockroachdb keys to %s", p), "event", "froach:key:write_dir")

	// TODO we write ca.key for now, we should not in the future
	err = os.WriteFile(filepath.Join(p, "ca.key"), caKeyPem, 0600)
	if err != nil {
		return fmt.Errorf("failed to write ca.key: %w", err)
	}
	err = os.WriteFile(filepath.Join(p, "ca.crt"), caCrtPem, 0644)
	if err != nil {
		return fmt.Errorf("failed to write ca.crt: %w", err)
	}

	caKey = key
	caCrt = caCrtParsed

	return checkNodeKeys()
}

// checkNodeKeys checks if node.pem and user.root.pem exist, are not expiring and are signed by the correct CA. If not, these are re-generated.
func checkNodeKeys() error {
	if err := checkOrCreateKey("node.crt", "node.key", "node", fleet.Self().AltNames()...); err != nil {
		return err
	}
	if err := checkOrCreateKey("client.root.crt", "client.root.key", "root"); err != nil {
		return err
	}
	return nil
}

func checkOrCreateKey(crtFile, keyFile, cn string, altNames ...string) error {
	p := basePath()

	_, err := os.Stat(filepath.Join(p, crtFile))
	_, err2 := os.Stat(filepath.Join(p, keyFile))
	// either file is missing → create
	if err != nil || err2 != nil {
		return createKey(crtFile, keyFile, cn, altNames...)
	}

	crt, err := readCertificateFile(filepath.Join(p, crtFile))
	if err != nil {
		return createKey(crtFile, keyFile, cn, altNames...)
	}

	// check if crt.Issuer == caCrt.Subject. Only check commonname for now
	if crt.Issuer.CommonName != caCrt.Subject.CommonName {
		return createKey(crtFile, keyFile, cn, altNames...)
	}

	// assume ok
	return nil
}

func createKey(crtFile, keyFile, cn string, altNames ...string) error {
	newKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	keyBin, err := x509.MarshalPKCS8PrivateKey(newKey)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}

	// create key & CA-signed CA
	now := time.Now()
	crtTpl := &x509.Certificate{
		BasicConstraintsValid: true,
		IsCA:                  false,
		SerialNumber:          new(big.Int).SetBytes(keyBin),
		Issuer:                caCrt.Subject,
		Subject:               pkix.Name{CommonName: cn},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageDataEncipherment | x509.KeyUsageKeyEncipherment,
		NotBefore:             now,
		NotAfter:              now.Add(365 * 24 * time.Hour),
	}

	// add altnames
	for _, a := range altNames {
		if ip := net.ParseIP(a); ip != nil {
			crtTpl.IPAddresses = append(crtTpl.IPAddresses, ip)
		} else {
			crtTpl.DNSNames = append(crtTpl.DNSNames, a)
		}
	}

	crtBin, err := x509.CreateCertificate(rand.Reader, crtTpl, caCrt, newKey.Public(), caKey)
	if err != nil {
		return err
	}

	keyPem := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBin})
	crtPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: crtBin})

	// write files
	// certificates stored in ~/.config/froach and data in ~/.cache/froach
	p := basePath()
	os.MkdirAll(p, 0755)

	// TODO we write ca.key for now, we should not in the future
	err = os.WriteFile(filepath.Join(p, keyFile), keyPem, 0600)
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", keyFile, err)
	}
	err = os.WriteFile(filepath.Join(p, crtFile), crtPem, 0644)
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", crtFile, err)
	}
	return nil
}
