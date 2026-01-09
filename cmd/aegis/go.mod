module github.com/thecodearcher/aegis/cmd/aegis

go 1.24.0

replace github.com/thecodearcher/aegis => ../../

require github.com/thecodearcher/aegis v0.0.0

require (
	github.com/go-sql-driver/mysql v1.8.1
	github.com/jackc/pgx/v5 v5.6.0
	github.com/urfave/cli/v3 v3.6.1
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	golang.org/x/crypto v0.43.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/text v0.30.0 // indirect
)
