package main

import (
	"database/sql"
	"embed"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"

	"github.com/thecodearcher/limen"
	sqladapter "github.com/thecodearcher/limen/adapters/sql"
	credentialpassword "github.com/thecodearcher/limen/plugins/credential-password"
	"github.com/thecodearcher/limen/plugins/passkey"
)

//go:embed index.html
var staticFS embed.FS

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("set DATABASE_URL")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	auth, err := limen.New(&limen.Config{
		BaseURL:  "http://localhost:8080",
		Database: sqladapter.NewPostgreSQL(db),
		Secret:   []byte("0123456789abcdef0123456789abcdef"),
		HTTP:     limen.NewDefaultHTTPConfig(limen.WithHTTPBasePath("/api/auth")),
		CLI:      &limen.CLIConfig{Enabled: true},
		Plugins: []limen.Plugin{
			// We pair passkey with credential-password so testers can
			// create an account first, then register a passkey against
			// that account.
			credentialpassword.New(),
			passkey.New(
				passkey.WithRPID("localhost"),
				passkey.WithOrigins("http://localhost:8080"),
				passkey.WithRPName("Limen Passkey Demo"),
			),
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/api/auth/", auth.Handler())
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	log.Println("passkey example listening on http://localhost:8080")
	log.Println("  open http://localhost:8080 in Chrome and use the virtual authenticator")
	log.Println("  (DevTools → ... → More tools → WebAuthn → Enable virtual authenticator)")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
