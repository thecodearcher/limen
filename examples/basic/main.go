package main

import (
	"context"
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/thecodearcher/aegis"
	adapter "github.com/thecodearcher/aegis/adapters/gorm"
	emailpassword "github.com/thecodearcher/aegis/features/email-password"
)

// Example showing basic usage of the aegis library
func main() {
	fmt.Println("Aegis Authentication Library - Basic Example")
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		"localhost",
		"root",
		"root",
		"aegis",
		"5432",
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	defaultConfig := emailpassword.DefaultConfig()
	defaultConfig.PasswordHasherConfig.Parallel = 4
	// Create configuration

	config := &aegis.Config{
		Database: adapter.New(db),
		Features: []aegis.Feature{
			emailpassword.New(defaultConfig),
		},
		JWT: aegis.NewDefaultJWTConfig(
			aegis.WithJWTSecret("test-secret"),
			aegis.WithClaimsSubjectField("uuid"),
		),
	}

	aegis, err := aegis.New(config)
	if err != nil {
		log.Fatalf("Failed to create aegis: %v", err)
	}

	fmt.Printf("%+v\n", aegis)
	fmt.Println("Aegis instance created successfully!")

	response, err := aegis.EmailPassword.SignInWithEmailAndPassword(context.Background(), "johndoe@gmail.com", "SecurePassword123@")
	if err != nil {
		log.Fatalf("Failed to sign in: %v", err)
	}

	fmt.Printf("Sign in response: %+v\n", response)
	validatedToken, err := aegis.JWT.VerifyToken(response.AccessToken)
	if err != nil {
		log.Fatalf("Failed to verify token: %v", err)
	}
	fmt.Printf("Validated token: %+v\n", validatedToken)
	// aegis.signIn.WithEmailAndPassword(ctx, "test@test.com", "password")

	// aegis.Email.WithPassword()

	// emailPasswordPlugin := aegis.RegisterPlugin(emailpassword.New())
	// emailPasswordPlugin.WithPassword('sdjnndsccd')

}
