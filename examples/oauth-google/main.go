package main

import (
	"log"
	"net/http"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/thecodearcher/limen"
	gormadapter "github.com/thecodearcher/limen/adapters/gorm"
	"github.com/thecodearcher/limen/plugins/oauth"
	oauthgoogle "github.com/thecodearcher/limen/plugins/oauth-google"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("set DATABASE_URL")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	auth, err := limen.New(&limen.Config{
		BaseURL:  "http://localhost:8080",
		Database: gormadapter.New(db),
		Secret:   []byte("0123456789abcdef0123456789abcdef"),
		Plugins: []limen.Plugin{
			oauth.New(
				oauth.WithProviders(
					oauthgoogle.New(
						oauthgoogle.WithClientID(os.Getenv("GOOGLE_CLIENT_ID")),
						oauthgoogle.WithClientSecret(os.Getenv("GOOGLE_CLIENT_SECRET")),
					),
				),
			),
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/api/auth/", auth.Handler())

	log.Println("oauth-google example listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
