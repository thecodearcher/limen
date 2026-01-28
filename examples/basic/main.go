package main

import (
	"context"
	"fmt"
	"log"
	"maps"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/thecodearcher/aegis"
	adapter "github.com/thecodearcher/aegis/adapters/gorm"
	"github.com/thecodearcher/aegis/examples/basic/pkg"
	credentialpassword "github.com/thecodearcher/aegis/features/credential-password"
	twofactor "github.com/thecodearcher/aegis/features/two-factor"
)

// GetConfig returns the aegis configuration
// This function is exported for use by the CLI tool
func GetConfig() *aegis.Config {
	// Return config without database connection for CLI usage
	// Database adapter is not needed for schema discovery
	return buildConfig(nil)
}

type UUIDGenerator struct {
}

func (g *UUIDGenerator) GetColumnType() aegis.ColumnType {
	return aegis.ColumnTypeUUID
}

func (g *UUIDGenerator) Generate(ctx context.Context) (any, error) {
	return uuid.New().String(), nil
}

// buildConfig builds the aegis configuration
func buildConfig(db *gorm.DB) *aegis.Config {
	var dbAdapter aegis.DatabaseAdapter
	if db != nil {
		dbAdapter = adapter.New(db)
	}

	return &aegis.Config{
		Database: dbAdapter,
		Features: []aegis.Feature{

			credentialpassword.New(
				credentialpassword.WithRequireEmailVerification(true),
				credentialpassword.WithSendVerificationEmail(func(email string, token string) {
					fmt.Printf("Sending verification email to %s\n", email)
					fmt.Printf("Verification token: %s\n", token)

				}),
				credentialpassword.WithSendPasswordResetEmail(func(email string, token string) {
					fmt.Printf("Sending password reset email to %s\n", email)
					fmt.Printf("Password reset token: %s\n", token)

				}),
				credentialpassword.WithUsernameSupport(true),
				credentialpassword.WithRequireUsernameOnSignUp(true),
			),
			twofactor.New(
				twofactor.WithTOTP(
					// twofactor.WithTOTPSecret([]byte("default_secret")),
					twofactor.WithTOTPIssuer("Aegis"),
					twofactor.WithTOTPTTL(30*time.Second),
					twofactor.WithTOTPDigits(twofactor.TOTPDigitsEight),
					twofactor.WithTOTPAlgorithm(twofactor.TOTPAlgorithmSHA1),
				),
				twofactor.WithBackupCodes(
					twofactor.WithBackupCodesCount(20),
					twofactor.WithBackupCodesLength(10),
				),
				twofactor.WithOTP(
					twofactor.WithOTPDigits(twofactor.TOTPDigitsSix),
					twofactor.WithOTPCodeExpiration(30*time.Second),
					twofactor.WithOTPSendCode(func(ctx context.Context, user *twofactor.UserWithTwoFactor, code string) {
						fmt.Printf("Sending OTP code to %s\n", user.Email)
						fmt.Printf("OTP code: %s\n", code)

					}),
				),
			),
		},
		CLI: &aegis.CLIConfig{
			Enabled: true,
		},
		Schema: aegis.NewDefaultSchemaConfig(
			// aegis.WithSchemaIDGenerator(&UUIDGenerator{}),
			aegis.WithSchemaUser(
				// aegis.WithUserTableName("usersz_from_personal_user_schema"),
				// aegis.WithUserFieldID("id_from_personal"),
				aegis.WithUserFieldEmailVerifiedAt("email_verified"),
				// aegis.WithUserFieldEmail("email_from_personal"),
				aegis.WithUserAdditionalFields(func(ctx *aegis.AdditionalFieldsContext) (map[string]any, error) {
					if ctx.IsEmpty("firstname") {
						return nil, aegis.NewAegisError("firstname is required", http.StatusBadRequest, nil)
					}
					if ctx.IsEmpty("lastname") {
						return nil, aegis.NewAegisError("lastname is required", http.StatusBadRequest, nil)
					}
					return map[string]any{
						"uuid":       "fbcb9690-0879-4595-bf03-09d21646c894",
						"first_name": ctx.GetBodyValue("firstname"),
						"last_name":  ctx.GetBodyValue("lastname"),
						"updated_at": time.Now().Format(time.RFC3339),
					}, nil
				}),

				// aegis.WithUserSerializer(func(data *aegis.User) map[string]any {
				// 	return map[string]any{
				// 		"id":                data.ID,
				// 		"email":             data.Email,
				// 		"password":          data.Password,
				// 		"email_verified_at": data.EmailVerifiedAt,
				// 	}
				// }),
			),
			aegis.WithSchemaVerification(
				aegis.WithVerificationAdditionalFields(func(ctx *aegis.AdditionalFieldsContext) (map[string]any, error) {
					return map[string]any{
						"uuid":       uuid.New().String(),
						"created_at": time.Now().Format(time.RFC3339),
						"updated_at": time.Now().Format(time.RFC3339),
					}, nil
				}),
			),
			// Example: Customize plugin schema table and field names

			aegis.WithPluginSchema(aegis.FeatureCredentialPassword, "something_map_name2",
				aegis.WithPluginFieldName("name", "name_from_plugin"),
			),
		),
		// Schema: aegis.SchemaConfig{
		// 	// AdditionalFields: func(ctx *schemas.AdditionalFieldsContext) map[string]any {
		// 	// 	return map[string]any{
		// 	// 		"uuid":       uuid.New().String(),
		// 	// 		"created_at": time.Now(),
		// 	// 		"updated_at": time.Now(),
		// 	// 	}
		// 	// },
		// 	User: aegis.UserSchema{
		// 		Fields: aegis.UserFields{
		// 			EmailVerifiedAt: "email_verified",
		// 		},
		// 		AdditionalFields: func(ctx *aegis.AdditionalFieldsContext) (map[string]any, *aegis.AegisError) {
		// 			return map[string]any{
		// 				"uuid":       uuid.New().String(),
		// 				"created_at": time.Now().Format(time.RFC3339),
		// 				"updated_at": time.Now().Format(time.RFC3339),
		// 				"first_name": ctx.GetBodyValue("firstname"),
		// 				"last_name":  ctx.GetBodyValue("lastname"),
		// 			}, nil
		// 		},
		// 	},
		// },
		// Session: aegis.NewDefaultSessionConfig(
		// 	// aegis.WithSessionStoreType(aegis.SessionStoreTypeMemory),
		// 	// aegis.WithSessionStrategy(aegis.SessionStrategyServerSide),
		// 	// aegis.WithSessionTokenDeliveryMethod(aegis.TokenDeliveryHeader),
		// ),
		HTTP: aegis.NewDefaultHTTPConfig(
			aegis.WithHTTPBasePath("/api/auth"),
			aegis.WithHTTPRateLimiter(aegis.WithRateLimiterMaxRequests(3)),
			aegis.WithHTTPCookieName("default_session"),
			aegis.WithHTTPRateLimiter(aegis.WithRateLimiterDisableForPaths("/me", "/signin/email")),
			// aegis.WithHTTPTrustedOrigins([]string{
			// 	"*.localhost:3000", "https://localhost:3000",
			// 	"myapp://",                             // Mobile app scheme
			// 	"chrome-extension://YOUR_EXTENSION_ID", // Browser extension
			// 	"exp://*/*",                            // Trust all Expo development URLs
			// 	"exp://10.0.0.*:*/*",                   // Trust 10.0.0.x IP range with any port,
			// 	// "*.example.com",
			// 	"https://*.example.com",
			// 	"http://*.dev.example.com",
			// }),
		),
		// 	aegis.WithRateLimiterWindow(time.Minute),

		// aegis.WithRateLimiterStore(aegis.RateLimiterStoreTypeDatabase),
	}
}

// Example showing basic usage of the aegis library
func main() {
	fmt.Println(pkg.SomeShi())
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

	config := buildConfig(db)

	auth, err := aegis.New(config)
	if err != nil {
		log.Fatalf("Failed to create aegis: %v", err)
	}

	handler := auth.Handler()
	// schemas, err := aegis.DiscoverAllSchemasFromConfig(config)
	// if err != nil {
	// 	log.Fatalf("Failed to discover all schemas: %v", err)
	// }

	// fmt.Printf("Schemas: %+v\n", schemas)
	// copyConfig := &config

	// migrations, err := aegis.GenerateMigrations(copyConfig, adapter.NewMigrationGenerator("postgres"))
	// if err != nil {
	// 	log.Fatalf("Failed to generate migrations: %v", err)
	// }
	// fmt.Printf("Migrations: %+v\n", migrations)
	// code, err := aegis.GenerateGoStructsFromConfig(config, aegis.GenerateOptions{
	// 	PackageName: "models",
	// 	Tags:        []string{"json", "gorm"},
	// })
	// if err != nil {
	// 	log.Fatalf("Failed to generate Go structs: %v", err)
	// }
	// fmt.Printf("Code: %+v\n", code)
	// 	fmt.Printf("Before request %s %s\n", ctx.Request.Method, ctx.Request.URL.Path)
	// 	fmt.Printf("Before request body: %+v\n", ctx.BodyData)
	// }),
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

	r.Any("/api/auth/*path", func(c *gin.Context) {
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
