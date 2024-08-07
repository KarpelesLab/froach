package froach

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"errors"
	"io/fs"
	"time"

	"github.com/KarpelesLab/fleet"
)

func init() {
	go start()
}

func start() {
	fleet.Self().DbWatch("froach:ca:key!", updateKey)
	fleet.Self().WaitReady()    // this will wait for fleet to start
	time.Sleep(5 * time.Second) // give a bit of time just in case

	k, err := fleet.Self().DbGet("froach:ca:key!")
	if errors.Is(err, fs.ErrNotExist) {
		// no key? generate one
		newKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err == nil {
			kData, err := x509.MarshalPKCS8PrivateKey(newKey)
			if err == nil {
				// let's try to use this key
				// DbSet will trigger the watcher, that will call updateKey accordingly
				fleet.Self().DbSet("froach:ca:key!", kData)
			}
		}
	} else {
		// initially set the key
		updateKey("froach:ca:key!", k)
	}

	go monitor()
}
