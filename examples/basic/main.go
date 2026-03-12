package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/joho/godotenv"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/thecodearcher/aegis"
	gormadapter "github.com/thecodearcher/aegis/adapters/gorm"
	sqladapter "github.com/thecodearcher/aegis/adapters/sql"
	"github.com/thecodearcher/aegis/examples/basic/pkg"
	credentialpassword "github.com/thecodearcher/aegis/plugins/credential-password"
	"github.com/thecodearcher/aegis/plugins/oauth"
	oauthgeneric "github.com/thecodearcher/aegis/plugins/oauth-generic"
	oauthgithub "github.com/thecodearcher/aegis/plugins/oauth-github"
	oauthgoogle "github.com/thecodearcher/aegis/plugins/oauth-google"
	sessionjwt "github.com/thecodearcher/aegis/plugins/session-jwt"
	twofactor "github.com/thecodearcher/aegis/plugins/two-factor"
)

type UUIDGenerator struct {
}

func (g *UUIDGenerator) GetColumnType() aegis.ColumnType {
	return aegis.ColumnTypeUUID
}

func (g *UUIDGenerator) Generate(ctx context.Context) (any, error) {
	return uuid.New().String(), nil
}

// strFromRaw returns a string from a JSON value (string or number).
func strFromRaw(v any) string {
	if v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	case float64:
		return fmt.Sprintf("%.0f", s)
	default:
		return fmt.Sprint(v)
	}
}

// discordMapUserInfo maps Discord's /users/@me response to oauth.ProviderUserInfo.
func discordMapUserInfo(raw map[string]any) (*oauth.ProviderUserInfo, error) {
	id := strFromRaw(raw["id"])
	username := strFromRaw(raw["username"])
	email := strFromRaw(raw["email"])
	avatar := strFromRaw(raw["avatar"])
	avatarURL := ""
	if id != "" && avatar != "" {
		avatarURL = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", id, avatar)
	}
	return &oauth.ProviderUserInfo{
		ID:            id,
		Email:         email,
		EmailVerified: false,
		Name:          username,
		AvatarURL:     avatarURL,
	}, nil
}

// oidcMapUserInfo maps standard OIDC claims (id_token or userinfo) to oauth.ProviderUserInfo.
// Works with Keycloak, Auth0, and other OpenID Connect providers.
func oidcMapUserInfo(raw map[string]any) (*oauth.ProviderUserInfo, error) {
	sub := strFromRaw(raw["sub"])
	if sub == "" {
		return nil, fmt.Errorf("oidc: missing sub claim")
	}
	email := strFromRaw(raw["email"])
	name := strFromRaw(raw["name"])
	if name == "" {
		name = strFromRaw(raw["preferred_username"])
	}
	picture := strFromRaw(raw["picture"])
	emailVerified := false
	if v, ok := raw["email_verified"]; ok {
		switch b := v.(type) {
		case bool:
			emailVerified = b
		case string:
			emailVerified = b == "true" || b == "1"
		}
	}
	return &oauth.ProviderUserInfo{
		ID:            sub,
		Email:         email,
		EmailVerified: emailVerified,
		Name:          name,
		AvatarURL:     picture,
	}, nil
}

// buildOAuthOptions returns OAuth plugin options, including generic Discord and Keycloak (discovery) providers when credentials are set.
func buildOAuthOptions(googleClientID, googleClientSecret, githubClientID, githubClientSecret, discordClientID, discordClientSecret, keycloakDiscoveryURL, keycloakClientID, keycloakClientSecret string) []oauth.ConfigOption {
	opts := []oauth.ConfigOption{
		oauth.WithProvider(oauthgoogle.New(
			oauthgoogle.WithClientID(googleClientID),
			oauthgoogle.WithClientSecret(googleClientSecret),
			oauthgoogle.WithOption("access_type", "offline"),
			oauthgoogle.WithOption("prompt", "consent"),
		)),
		oauth.WithProvider(oauthgithub.New(
			oauthgithub.WithClientID(githubClientID),
			oauthgithub.WithClientSecret(githubClientSecret),
		)),
	}
	if discordClientID != "" && discordClientSecret != "" {
		opts = append(opts, oauth.WithProvider(oauthgeneric.New(
			oauthgeneric.WithName("discord"),
			oauthgeneric.WithClientID(discordClientID),
			oauthgeneric.WithClientSecret(discordClientSecret),
			oauthgeneric.WithAuthorizationURL("https://discord.com/api/oauth2/authorize"),
			oauthgeneric.WithTokenURL("https://discord.com/api/oauth2/token"),
			oauthgeneric.WithUserInfoURL("https://discord.com/api/users/@me"),
			oauthgeneric.WithScopes("identify", "email"),
			oauthgeneric.WithMapUserInfo(discordMapUserInfo),
		)))
	}
	if keycloakDiscoveryURL != "" && keycloakClientID != "" && keycloakClientSecret != "" {
		opts = append(opts, oauth.WithProvider(oauthgeneric.New(
			oauthgeneric.WithName("keycloak"),
			oauthgeneric.WithClientID(keycloakClientID),
			oauthgeneric.WithClientSecret(keycloakClientSecret),
			oauthgeneric.WithDiscoveryURL(keycloakDiscoveryURL),
			oauthgeneric.WithMapUserInfo(oidcMapUserInfo),
		)))
	}
	opts = append(opts, oauth.WithMapProfileToUser(func(info *aegis.OAuthAccountProfile) map[string]any {
		fmt.Printf("Mapping OAuth profile to user additional fields: %+v\n", info)
		switch info.Provider {
		case "google":
			firstName, _ := info.Raw["given_name"].(string)
			lastName, _ := info.Raw["family_name"].(string)
			return map[string]any{"first_name": firstName, "last_name": lastName}
		case "discord":
			username, _ := info.Raw["username"].(string)
			return map[string]any{"first_name": username, "last_name": ""}
		case "keycloak":
			name, _ := info.Raw["name"].(string)
			return map[string]any{"first_name": name, "last_name": ""}
		default:
			name, _ := info.Raw["name"].(string)
			return map[string]any{"first_name": name, "last_name": ""}
		}
	}))
	return opts
}

// buildConfig builds the aegis configuration
func buildConfig(db aegis.DatabaseAdapter) *aegis.Config {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Failed to load .env file: %v", err)
	}
	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	googleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	githubClientID := os.Getenv("GITHUB_CLIENT_ID")
	githubClientSecret := os.Getenv("GITHUB_CLIENT_SECRET")
	discordClientID := os.Getenv("DISCORD_CLIENT_ID")
	discordClientSecret := os.Getenv("DISCORD_CLIENT_SECRET")
	keycloakDiscoveryURL := os.Getenv("KEYCLOAK_DISCOVERY_URL") // e.g. https://keycloak.example.com/realms/master/.well-known/openid-configuration
	keycloakClientID := os.Getenv("KEYCLOAK_CLIENT_ID")
	keycloakClientSecret := os.Getenv("KEYCLOAK_CLIENT_SECRET")

	fmt.Printf("Google Client ID: %s\n", googleClientID)

	return &aegis.Config{
		BaseURL:  "http://localhost:8080",
		Database: db,
		Secret:   []byte("rNH8JSJcbiyoPhXk5hQEjbI86SaSIgzw"), // 32 bytes for cookies + plugins (OAuth, 2FA) when they omit their own
		Plugins: []aegis.Plugin{
			sessionjwt.New(
			// sessionjwt.WithRefreshToken(false),
			// sessionjwt.WithSigningKey([]byte("rNH8JSJcbiyoPhXk5hQEjbI86SaSIgzw")),
			// sessionjwt.WithSigningMethod(jwt.SigningMethodRS512),
			// sessionjwt.WithAccessTokenDuration(15*time.Minute),
			// sessionjwt.WithRefreshTokenDuration(7*24*time.Hour),
			// sessionjwt.WithRefreshTokenRotation(true),
			// sessionjwt.WithBlacklist(true),
			),

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
				credentialpassword.WithRequireUsernameOnSignUp(false),
			),
			oauth.New(buildOAuthOptions(googleClientID, googleClientSecret, githubClientID, githubClientSecret, discordClientID, discordClientSecret, keycloakDiscoveryURL, keycloakClientID, keycloakClientSecret)...),
			twofactor.New(
				// twofactor.WithCookieExpiration(2*time.Minute),
				twofactor.WithSecret("aegis_2fa_totp_secret_1234567890"),
				twofactor.WithTOTP(
					twofactor.WithTOTPIssuer("Aegis"),
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
					// if ctx.IsEmpty("firstname") {
					// 	return nil, aegis.NewAegisError("firstname is required", http.StatusBadRequest, nil)
					// }
					// if ctx.IsEmpty("lastname") {
					// 	return nil, aegis.NewAegisError("lastname is required", http.StatusBadRequest, nil)
					// }
					return map[string]any{
						// "uuid":       "fbcb9690-0879-4595-bf03-09d21646c894",
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
						// "uuid":       uuid.New().String(),
						"created_at": time.Now().Format(time.RFC3339),
						"updated_at": time.Now().Format(time.RFC3339),
					}, nil
				}),
			),
			// Example: Customize plugin schema table and field names

			aegis.WithPluginSchema(aegis.PluginCredentialPassword, "something_map_name2",
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
		Session: aegis.NewDefaultSessionConfig(
			// aegis.WithSessionStoreType(aegis.SessionStoreTypeMemory),
			// aegis.WithSessionStrategy(aegis.SessionStrategyServerSide),
			aegis.WithSessionUpdateAge(10 * time.Second),
		),
		HTTP: aegis.NewDefaultHTTPConfig(
			aegis.WithHTTPBasePath("/api/auth"),
			aegis.WithHTTPRateLimiter(aegis.WithRateLimiterMaxRequests(3)),
			aegis.WithHTTPSessionCookieName("session"),
			aegis.WithHTTPCookieSecure(false),
			aegis.WithHTTPRateLimiter(aegis.WithRateLimiterDisableForPaths("/me", "/signin/email")),

			aegis.WithHTTPSessionTransformer(sessionTransformer),
			aegis.WithHTTPTrustedOrigins([]string{
				"*",
				"*.localhost:3000", "http://localhost:3000",
				"myapp://",                             // Mobile app scheme
				"chrome-extension://YOUR_EXTENSION_ID", // Browser extension
				"exp://*/*",                            // Trust all Expo development URLs
				"exp://10.0.0.*:*/*",                   // Trust 10.0.0.x IP range with any port,
				// "*.example.com",
				"https://*.example.com",
				"http://*.dev.example.com",
			}),
			aegis.WithHTTPHooks(&aegis.Hooks{
				Before: []*aegis.Hook{
					{
						PathMatcher: func(ctx *aegis.HookContext) bool {
							return true
						},
						Run: func(ctx *aegis.HookContext) bool {
							fmt.Printf("Before request %s %s\n", ctx.Method(), ctx.Path())
							fmt.Printf("Before request route pattern: %+v\n", ctx.RoutePattern())
							return true
						},
					},
					{
						PathMatcher: func(ctx *aegis.HookContext) bool {
							return ctx.RouteID() == "signup"
						},
						Run: func(ctx *aegis.HookContext) bool {
							email, ok := ctx.GetJSONBodyValue("email").(string)
							if !ok {
								return true
							}

							if !strings.Contains(email, "@example.com") {
								ctx.WriteErrorResponse(aegis.NewAegisError("email domain not allowed", http.StatusBadRequest, nil))
								return false
							}

							return true
						},
					},
				},
			}),
		),
		// 	aegis.WithRateLimiterWindow(time.Minute),

		// aegis.WithRateLimiterStore(aegis.RateLimiterStoreTypeDatabase),
	}
}

// ANSI color codes for terminal output (no dependency)
const (
	_reset   = "\033[0m"
	_dim     = "\033[2m"
	_red     = "\033[31m"
	_green   = "\033[32m"
	_yellow  = "\033[33m"
	_blue    = "\033[34m"
	_magenta = "\033[35m"
	_cyan    = "\033[36m"
)

type slogger struct {
	sqladapter.QueryLogger
}

func (l *slogger) LogQuery(ctx context.Context, query string, args any, duration time.Duration, err error) {
	fmt.Printf("%s[SQL]%s %s%s%s\n", _cyan, _reset, _green, query, _reset)
	fmt.Printf("  %sargs:%s %v\n", _dim, _reset, args)
	fmt.Printf("  %sduration:%s %s\n", _dim, _reset, _yellow+duration.String()+_reset)
	if err != nil {
		fmt.Printf("  %serr:%s %s%v%s\n", _dim, _reset, _red, err, _reset)
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

	gormdb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Info)})

	// mysqlDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", "root", "", "localhost", "3306", "aegis")
	// db, err := sql.Open("mysql", mysqlDSN)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	// db = sqldblogger.OpenDriver(mysqlDSN, db.Driver(), loggerAdapter /*, using_default_options*/) // db is STILL *sql.DB

	// sqldbAdapter := sqladapter.NewMySQL(db).WithLogger(&logger{})
	config := buildConfig(gormadapter.New(gormdb))

	auth, err := aegis.New(config)
	if err != nil {
		log.Fatalf("Failed to create aegis: %v", err)
	}

	// Type-safe plugin access via Use() -- chainable, panics if plugin not registered.
	// Assign once, call many times:
	//   cp := credentialpassword.Use(auth)
	//   result, err := cp.SignInWithCredentialAndPassword(ctx, "user@example.com", "password")
	//
	// One-liner chaining:
	//   oauth.Use(auth).GetAuthorizationURL(ctx, "google", &oauth.OAuthAuthorizeURLData{})
	//   twofactor.Use(auth).InitiateTwoFactorSetup(ctx, user, "password")
	//
	// Safe variant (returns bool instead of panicking):
	//   oauthAPI, ok := aegis.UsePlugin[oauth.API](auth, aegis.PluginOAuth)
	_ = credentialpassword.Use(auth)
	_ = oauth.Use(auth)
	_ = twofactor.Use(auth)

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
		http.Redirect(c.Writer, c.Request, "http://localhost:3000", 302)
	})

	r.Any("/api/auth/*path", func(c *gin.Context) {
		handler.ServeHTTP(c.Writer, c.Request)
	})

	http.ListenAndServe(":8080", r)

}

func sessionTransformer(user map[string]any, sessionResult *aegis.SessionResult) (map[string]any, error) {
	payload := map[string]any{
		"user": user,
	}
	if sessionResult != nil {
		payload["token"] = sessionResult.Token
	}
	return payload, nil
}
