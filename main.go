package froach

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"time"

	"github.com/KarpelesLab/fleet"
)

func init() {
	go start()
}

func start() {
	fleet.Self().DbWatch("froach:ca:key", updateKey)
	fleet.Self().WaitReady()    // this will wait for fleet to start
	time.Sleep(5 * time.Second) // give a bit of time just in case

	_, err := fleet.Self().DbGet("froach:ca:key")
	if err == nil {
		// no key? generate one
		newKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err == nil {
			kData, err := x509.MarshalPKCS8PrivateKey(newKey)
			if err == nil {
				// let's try to use this key
				fleet.Self().DbSet("froach:ca:key", kData)
				setPrivateKey(newKey)
			}
		}
	}

	go monitor()
}
