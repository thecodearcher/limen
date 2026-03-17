# [WIP] Limen Authentication Library

A modern, plugin-first authentication library for Go applications, inspired by better-auth from the TypeScript ecosystem.

## plugins

- 🔌 **Plugin-first architecture** - Only import what you need
- 🏗️ **Struct-based configuration** - Simple, strongly-typed setup
- 🗃️ **Flexible database integration** - Works with your existing database
- 🔍 **Typed query conditions** - Type-safe database operations
- 🚀 **Framework agnostic** - Works with Gin, Echo, Chi, net/http
- 🔒 **Security first** - Secure defaults with customizable options

## Quick Start

```go
package main

import (
    "github.com/thecodearcher/limen"
    "github.com/thecodearcher/limen/adapters/gorm"
    "github.com/thecodearcher/limen/plugins/email-password"
)

func main() {
    // Set up your database adapter
    db, _ := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    adapter := gorm.New(db)

    // Configure limen
    config := &limen.Config{
        User: limen.UserConfig{
            ModelName: "users",
            Fields: limen.UserFieldMappings{
                ID:    "user_id",
                Email: "email_address",
            },
        },
        Database: limen.DatabaseConfig{
            Adapter: adapter,
        },
    }

    // Create plugin registry
    registry := limen.NewPluginRegistry()

    // Register authentication plugins
    emailPasswordPlugin := credentialpassword.New()
    registry.RegisterPlugin(emailPasswordPlugin)

    // Use with your web framework
    // ... middleware setup
}
```

## Architecture

Limen follows a layered architecture with clear separation of concerns:

- **Framework Adapters** - Integration with web frameworks
- **Middleware Layer** - Authentication and authorization middleware
- **Core Services** - Business logic and coordination
- **Provider Layer** - Authentication providers (plugins)
- **Storage Layer** - Database adapters

## Configuration

Limen uses struct-based configuration similar to better-auth:

```go
config := &limen.Config{
    User: limen.UserConfig{
        ModelName: "app_users",
        Fields: limen.UserFieldMappings{
            ID:        "user_id",
            Email:     "email_address",
            CreatedAt: "created_timestamp",
            UpdatedAt: "modified_timestamp",
        },
    },
    Session: limen.SessionConfig{
        ModelName: "user_sessions",
        Duration:  24 * time.Hour,
        Fields: limen.SessionFieldMappings{
            ID:        "session_id",
            UserID:    "user_id",
            ExpiresAt: "expires_timestamp",
        },
    },
}
```

## Database Integration

Limen works with your existing database through adapters:

```go
// Using GORM
import "github.com/thecodearcher/limen/adapters/gorm"
db, _ := gorm.Open(postgres.Open(dsn), &gorm.Config{})
adapter := gorm.New(db)

// Using Ent
import "github.com/thecodearcher/limen/adapters/ent"
client, _ := ent.Open("postgres", dsn)
adapter := ent.New(client)

// Using database/sql
import "github.com/thecodearcher/limen/adapters/sql"
db, _ := sql.Open("postgres", dsn)
adapter := sql.NewPostgreSQL(db)

config.Database.Adapter = adapter
```

## Typed Query Conditions

Limen provides type-safe query conditions:

```go
// Simple conditions
conditions := []limen.Where{
    limen.Eq("email", "user@example.com"),
    limen.Gt("created_at", time.Now().AddDate(0, -1, 0)),
}

// Complex conditions with OR logic
conditions := []limen.Where{
    limen.Contains("email", "gmail"),
    limen.Contains("name", "john").Or(),
    limen.In("status", []string{"active", "pending"}),
}
```

## Multi-Module Architecture

Limen uses a multi-module architecture where each adapter and plugin is a separate Go module:

```
github.com/thecodearcher/limen                    # Core library
github.com/thecodearcher/limen/adapters/gorm      # GORM adapter
github.com/thecodearcher/limen/adapters/ent       # Ent adapter
github.com/thecodearcher/limen/adapters/sql       # SQL adapter
github.com/thecodearcher/limen/plugins/email-password  # Email/password auth
github.com/thecodearcher/limen/plugins/oauth-google    # Google OAuth
github.com/thecodearcher/limen/plugins/two-factor      # 2FA
```

### Plugin System

Authentication methods are implemented as separate modules:

```go
import (
    "github.com/thecodearcher/limen/plugins/email-password"
    "github.com/thecodearcher/limen/plugins/oauth-google"
    "github.com/thecodearcher/limen/plugins/two-factor"
)

// Only import the plugins you need
registry := limen.NewPluginRegistry()
registry.RegisterPlugin(credentialpassword.New())
registry.RegisterPlugin(oauthgoogle.New(googleConfig))
registry.RegisterPlugin(twofactor.New())
```

## Development

```bash
# Install development tools
make install-tools

# Run tests
make test

# Run tests with coverage
make test-cover

# Format and lint code
make fmt lint

# Run CI pipeline locally
make ci
```

## Contributing

Contributions are welcome! Please read our contributing guidelines and submit pull requests.

## License

MIT License - see LICENSE file for details.
