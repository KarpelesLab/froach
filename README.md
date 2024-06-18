# froach

Cockroach runner for fleet system

* Manages a CA and peers using the fleet system (CA private key is shared for now, since that's the same as using the cryptseed).
* will create node CA and root user certificate
* certificates stored in ~/.config/froach and data in ~/.cache/froach/db
* Able to download latest version of cockroachdb in ~/.cache/froach (will use azusa if found)

It will launch a daemon listening on port `36257` and connect the various daemons together.
