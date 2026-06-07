module example/passkey

go 1.25.0

require (
	github.com/lib/pq v1.10.9
	github.com/thecodearcher/limen v0.1.1
	github.com/thecodearcher/limen/adapters/sql v0.0.1
	github.com/thecodearcher/limen/plugins/credential-password v0.0.1
	github.com/thecodearcher/limen/plugins/passkey v0.0.0
)

require (
	github.com/fxamacker/cbor/v2 v2.9.0 // indirect
	github.com/go-webauthn/webauthn v0.13.4 // indirect
	github.com/go-webauthn/x v0.1.23 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.3 // indirect
	github.com/google/go-tpm v0.9.5 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jmoiron/sqlx v1.4.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	golang.org/x/crypto v0.48.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
)

replace (
	github.com/thecodearcher/limen => ../..
	github.com/thecodearcher/limen/adapters/sql => ../../adapters/sql
	github.com/thecodearcher/limen/plugins/credential-password => ../../plugins/credential-password
	github.com/thecodearcher/limen/plugins/passkey => ../../plugins/passkey
)
