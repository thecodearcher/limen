package main

import (
	"log"
	"net/http"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/thecodearcher/limen"
	gormadapter "github.com/thecodearcher/limen/adapters/gorm"
	credentialpassword "github.com/thecodearcher/limen/plugins/credential-password"
	twofactor "github.com/thecodearcher/limen/plugins/two-factor"
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
			credentialpassword.New(),
			twofactor.New(
				twofactor.WithTOTP(
					twofactor.WithTOTPIssuer("Limen Example"),
				),
			),
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/api/auth/", auth.Handler())

	log.Println("two-factor example listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
