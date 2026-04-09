# Limen

A modern, plugin-first authentication library for Go, inspired by [better-auth](https://www.better-auth.com/).

> **Status:** Work in progress — APIs may change before v1.0.

## Documentation

Full guides, configuration reference, and plugin documentation are available at **[limenauth.dev](https://limenauth.dev)**.

## Features

- **Plugin-first architecture** — only import the authentication methods you need
- **Struct-based configuration** with sensible defaults
- **Database adapters** for GORM and database/sql
- **Framework-agnostic** — returns a standard `http.Handler`
- **Security-first defaults** — Argon2id password hashing, 32-byte signing secret, secure session management

## Requirements

- Go 1.25+

## Installation

```bash
go get github.com/thecodearcher/limen
```

Then add the adapter and plugins your application needs:

```bash
go get github.com/thecodearcher/limen/adapters/gorm
go get github.com/thecodearcher/limen/plugins/credential-password
```

## Quick Start

```go
package main

import (
	"log"
	"net/http"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/thecodearcher/limen"
	gormadapter "github.com/thecodearcher/limen/adapters/gorm"
	credentialpassword "github.com/thecodearcher/limen/plugins/credential-password"
)

func main() {
	db, err := gorm.Open(postgres.Open("your-dsn"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	auth, err := limen.New(&limen.Config{
		BaseURL:  "http://localhost:8080",
		Database: gormadapter.New(db),
		Secret:   []byte("your-32-byte-secret-key-here!!!!"), // exactly 32 bytes
		Plugins: []limen.Plugin{
			credentialpassword.New(),
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/api/auth/", auth.Handler())

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
```

The `Secret` field accepts exactly 32 bytes. Alternatively, set the `LIMEN_SECRET` environment variable and omit it from the struct.

For a more complete example with OAuth providers, two-factor auth, and Gin integration, see [`examples/basic`](examples/basic).

For full configuration options, middleware usage, and plugin APIs, visit **[limenauth.dev](https://limenauth.dev)**.

## Plugins and Adapters

Limen ships with plugins for email/password, OAuth (Google, GitHub, Apple, and more), two-factor auth, and JWT sessions — plus database adapters for GORM and database/sql. Each is a separate Go module so you only import what you need.

See the full list and setup guides at **[limenauth.dev](https://limenauth.dev)**.

## Development

```bash
# Run all tests (uses go.work for the multi-module workspace)
go test ./...

# Lint
golangci-lint run ./...
```

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

## License

MIT License — see [LICENSE](LICENSE) for details.
