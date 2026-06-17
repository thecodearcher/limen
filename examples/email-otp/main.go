package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"

	"github.com/thecodearcher/limen"
	sqladapter "github.com/thecodearcher/limen/adapters/sql"
	emailotp "github.com/thecodearcher/limen/plugins/email-otp"
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
		HTTP:     limen.NewDefaultHTTPConfig(limen.WithHTTPBasePath("/api/auth")),
		CLI:      &limen.CLIConfig{Enabled: true},
		Plugins: []limen.Plugin{
			emailotp.New(
				emailotp.WithSendOTP(func(msg emailotp.EmailOTPMessage) {
					// In production, send via your transactional email
					// provider. Here we just log so you can copy the OTP
					// out of the terminal during local testing.
					log.Printf("email-otp: type=%s email=%s otp=%s", msg.Type, msg.Email, msg.OTP)
				}),
			),
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/api/auth/", auth.Handler())

	mux.HandleFunc("GET /api/profile", func(w http.ResponseWriter, r *http.Request) {
		session, err := auth.GetSession(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"user":    session.User,
			"session": session.Session,
		})
	})

	log.Println("email-otp example listening on :8080")
	log.Println("  POST /api/auth/email-otp/send-otp      {\"email\":\"you@example.com\"}")
	log.Println("  POST /api/auth/email-otp/sign-in       {\"email\":\"you@example.com\",\"otp\":\"123456\"}")
	log.Println("  POST /api/auth/email-otp/verify-email  {\"email\":\"you@example.com\",\"otp\":\"123456\"}")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
