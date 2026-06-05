module example/email-otp

go 1.25.0

require (
	github.com/lib/pq v1.10.9
	github.com/thecodearcher/limen v0.1.1
	github.com/thecodearcher/limen/adapters/sql v0.0.1
	github.com/thecodearcher/limen/plugins/email-otp v0.0.0
)

require (
	github.com/jmoiron/sqlx v1.4.0 // indirect
	golang.org/x/crypto v0.48.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
)

replace (
	github.com/thecodearcher/limen => ../..
	github.com/thecodearcher/limen/adapters/sql => ../../adapters/sql
	github.com/thecodearcher/limen/plugins/email-otp => ../../plugins/email-otp
)
