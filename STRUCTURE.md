# [WIP] Aegis Project Structure

This document describes the multi-module structure of the Aegis authentication library.

## Overview

Aegis follows a multi-module architecture where the core library and each adapter/plugin is a separate Go module. This allows users to import only what they need, keeping bundle sizes minimal.

## Directory Structure

```
aegis/
├── go.work                          # Go workspace file
├── go.mod                           # Core library module
├── *.go                             # Core library source files
├── *_test.go                        # Core library tests
├── README.md                        # Main documentation
├── STRUCTURE.md                     # This file
├── Makefile                         # Build and development tasks
├── .golangci.yml                    # Linting configuration
├── .github/workflows/               # CI/CD configuration
│
├── adapters/                        # Database adapters (separate modules)
│   ├── gorm/                        # GORM adapter
│   │   ├── go.mod
│   │   ├── adapter.go
│   │   └── adapter_test.go
│   ├── ent/                         # Ent adapter
│   │   ├── go.mod
│   │   ├── adapter.go
│   │   └── adapter_test.go
│   └── sql/                         # database/sql adapter
│       ├── go.mod
│       ├── adapter.go
│       └── adapter_test.go
│
├── plugins/                         # Authentication plugins (separate modules)
│   ├── email-password/              # Email/password authentication
│   │   ├── go.mod
│   │   ├── plugin.go
│   │   └── plugin_test.go
│   ├── oauth-google/                # Google OAuth
│   │   ├── go.mod
│   │   ├── plugin.go
│   │   └── plugin_test.go
│   ├── oauth-github/                # GitHub OAuth
│   │   ├── go.mod
│   │   ├── plugin.go
│   │   └── plugin_test.go
│   └── two-factor/                  # Two-factor authentication
│       ├── go.mod
│       ├── plugin.go
│       └── plugin_test.go
│
├── examples/                        # Example applications
│   ├── basic/                       # Basic usage example
│   │   └── main.go
│   ├── gin-app/                     # Gin framework example
│   │   └── main.go
│   └── echo-app/                    # Echo framework example
│       └── main.go
│
├── cmd/                             # CLI tools
│   └── aegis/                       # Aegis CLI tool
│       └── main.go
│
└── internal/                        # Internal packages (not exported)
    ├── auth/                        # Internal auth utilities
    ├── database/                    # Internal database utilities
    └── session/                     # Internal session utilities
```

## Module Dependencies

### Core Library (`github.com/thecodearcher/aegis`)

- Contains core interfaces, types, and configuration
- No external dependencies except standard library
- All other modules depend on this

### Database Adapters

- `github.com/thecodearcher/aegis/adapters/gorm` - Depends on GORM
- `github.com/thecodearcher/aegis/adapters/ent` - Depends on Ent
- `github.com/thecodearcher/aegis/adapters/sql` - Only standard library

### Authentication Plugins

- `github.com/thecodearcher/aegis/plugins/email-password` - Depends on golang.org/x/crypto
- `github.com/thecodearcher/aegis/plugins/oauth-google` - Depends on golang.org/x/oauth2
- `github.com/thecodearcher/aegis/plugins/oauth-github` - Depends on golang.org/x/oauth2
- `github.com/thecodearcher/aegis/plugins/two-factor` - Depends on TOTP libraries

## Usage Patterns

### Minimal Usage (Core + SQL Adapter)

```go
import (
    "github.com/thecodearcher/aegis"
    "github.com/thecodearcher/aegis/adapters/sql"
)
```

### With GORM and Email/Password

```go
import (
    "github.com/thecodearcher/aegis"
    "github.com/thecodearcher/aegis/adapters/gorm"
    "github.com/thecodearcher/aegis/plugins/email-password"
)
```

### Full Plugind (Multiple Plugins)

```go
import (
    "github.com/thecodearcher/aegis"
    "github.com/thecodearcher/aegis/adapters/gorm"
    "github.com/thecodearcher/aegis/plugins/email-password"
    "github.com/thecodearcher/aegis/plugins/oauth-google"
    "github.com/thecodearcher/aegis/plugins/two-factor"
)
```

## Development Workflow

### Working with the Workspace

The `go.work` file allows you to work on all modules simultaneously:

```bash
# Run tests for all modules
go test ./...

# Build all modules
go build ./...

# Add dependency to a specific module
cd adapters/gorm
go get gorm.io/gorm@latest
```

### Module Versioning

Each module can be versioned independently:

- Core library versions: `v1.0.0`, `v1.1.0`, etc.
- Adapter versions: `adapters/gorm/v1.0.0`, etc.
- Plugin versions: `plugins/email-password/v1.0.0`, etc.

### Publishing Modules

Modules are published to separate import paths:

```bash
# Tag core library
git tag v1.0.0

# Tag adapter
git tag adapters/gorm/v1.0.0

# Tag plugin
git tag plugins/email-password/v1.0.0
```

## Benefits of This Structure

1. **Minimal Dependencies**: Users only import what they need
2. **Independent Versioning**: Each module can be versioned separately
3. **Clear Separation**: Core, adapters, and plugins are clearly separated
4. **Easy Extension**: New adapters and plugins can be added easily
5. **Go-Idiomatic**: Follows Go module best practices
6. **Development Friendly**: Workspace allows working on all modules together
