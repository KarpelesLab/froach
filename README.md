# froach

Cockroach runner for fleet system

* Manages a CA and peers using the fleet system (CA private key is shared for now, since that's the same as using the cryptseed).
* will create node CA and root user certificate
* certificates stored in ~/.config/froach and data in ~/.cache/froach

Path: `/pkg/main/dev-db.cockroach-bin.core/bin/cockroach`
