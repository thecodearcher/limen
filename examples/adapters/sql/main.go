package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"

	"github.com/thecodearcher/limen"
	sqladapter "github.com/thecodearcher/limen/adapters/sql"
	credentialpassword "github.com/thecodearcher/limen/plugins/credential-password"
)

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
		Plugins: []limen.Plugin{
			credentialpassword.New(),
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/api/auth/", auth.Handler())

	log.Println("sql adapter example listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
