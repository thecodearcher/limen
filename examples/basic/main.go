package main

import (
	"fmt"
	"log"
	"maps"
	"net/http"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/gin-gonic/gin"
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
				emailpassword.WithRequireEmailVerification(true),
				emailpassword.WithSendVerificationEmail(func(email string, token string) error {
					fmt.Printf("Sending verification email to %s\n", email)
					fmt.Printf("Verification token: %s\n", token)
					return nil
				}),
				emailpassword.WithSendPasswordResetEmail(func(email string, token string) error {
					fmt.Printf("Sending password reset email to %s\n", email)
					fmt.Printf("Password reset token: %s\n", token)
					return nil
				}),
			),
		},
		JWT: aegis.NewDefaultJWTConfig(
			aegis.WithJWTSecret("test-secret"),
			aegis.WithClaimsSubjectField("uuid"),
		),
		Schema: aegis.SchemaConfig{
			// AdditionalFields: func(ctx *schemas.AdditionalFieldsContext) map[string]any {
			// 	return map[string]any{
			// 		"uuid":       uuid.New().String(),
			// 		"created_at": time.Now(),
			// 		"updated_at": time.Now(),
			// 	}
			// },
			User: aegis.UserSchema{
				Fields: aegis.UserFields{
					EmailVerifiedAt: "email_verified",
				},
				AdditionalFields: func(ctx *aegis.AdditionalFieldsContext) (map[string]any, *aegis.AegisError) {
					return map[string]any{
						"uuid":       uuid.New().String(),
						"created_at": time.Now().Format(time.RFC3339),
						"updated_at": time.Now().Format(time.RFC3339),
						"first_name": ctx.GetBodyValue("firstname"),
						"last_name":  ctx.GetBodyValue("lastname"),
					}, nil
				},
			},
		},
		Session: aegis.NewDefaultSessionConfig(aegis.WithSessionStoreType(aegis.SessionStoreTypeDatabase)),
	}

	auth, err := aegis.New(config)
	if err != nil {
		log.Fatalf("Failed to create aegis: %v", err)
	}

	handler := auth.Handler(aegis.WithHTTPBasePath("/api/auth"))
	// 		fmt.Printf("Before request %s %s\n", ctx.Request.Method, ctx.Request.URL.Path)
	// 		fmt.Printf("Before request body: %+v\n", ctx.BodyData)
	// 	}),
	// 	After: httpx.HookFunc(func(ctx *httpx.HookContext) {
	// 		fmt.Printf("After request %s %s\n", ctx.Request.Method, ctx.Request.URL.Path)
	// 		fmt.Printf("After request status code: %d\n", ctx.StatusCode)
	// 	}),
	// }),

	r := gin.Default()
	r.GET("/", func(c *gin.Context) {
		// session, err := auth.GetSession(c.Request)
		// if err != nil {
		// 	c.JSON(500, gin.H{"message": "Failed to get session"})
		// 	return
		// }
		// fmt.Printf("Session: %+v\n", session)
		c.JSON(200, gin.H{"message": "Hello, World!"})
	})

	r.Any("/api/*path", func(c *gin.Context) {
		handler.ServeHTTP(c.Writer, c.Request)
	})

	http.ListenAndServe(":8080", r)

}

func sessionTransformer(user map[string]any, pendingActions []aegis.PendingAction, token string, refreshToken string) (map[string]any, *aegis.AegisError) {
	payload := map[string]any{
		"pending_actions": pendingActions,
		"token":           token,
		"refresh_token":   refreshToken,
		"user":            user,
	}
	maps.Copy(payload, user)
	return payload, nil
}
