package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/google/uuid"

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
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Info)})
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	config := &aegis.Config{
		Database: adapter.New(db),
		Features: []aegis.Feature{
			emailpassword.New(
				emailpassword.WithResetTokenExpiration(1*time.Minute),
				emailpassword.WithRemoveExpiredVerifications(false),
			),
		},
		JWT: aegis.NewDefaultJWTConfig(
			aegis.WithJWTSecret("test-secret"),
			aegis.WithClaimsSubjectField("uuid"),
		),
		Schema: aegis.SchemaConfig{
			SoftDeleteField: "soft_delete",
			AdditionalFields: func(ctx context.Context) map[string]any {
				return map[string]any{
					"uuid":       uuid.New().String(),
					"created_at": time.Now(),
					"updated_at": time.Now(),
				}
			},
		},
	}

	auth, err := aegis.New(config)
	if err != nil {
		log.Fatalf("Failed to create aegis: %v", err)
	}

	fmt.Printf("%+v\n", auth)
	fmt.Println("Aegis instance created successfully!")
	uuid := uuid.New().String()
	fmt.Printf("UUID: %s\n", uuid)
	response, err := auth.EmailPassword.SignInWithEmailAndPassword(context.Background(), "johndoe4@gmail.com", "SecurePassword123@")
	if err != nil {
		log.Fatalf("Failed to sign in: %v", err)
	}

	fmt.Printf("User: %+v\n", response.User)

	verification, err := auth.EmailPassword.RequestPasswordReset(context.Background(), "johndoe4@gmail.com")
	if err != nil {
		log.Fatalf("Failed to request password reset: %v", err)
	}

	err = auth.EmailPassword.ResetPassword(context.Background(), verification.Value, "SecurePassword123@")
	if err != nil {
		log.Fatalf("Failed to reset password: %v", err)
	}

	fmt.Printf("Password reset: %+v\n", verification)

}
