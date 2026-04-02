package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/joho/godotenv"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/thecodearcher/limen"
	gormadapter "github.com/thecodearcher/limen/adapters/gorm"
	sqladapter "github.com/thecodearcher/limen/adapters/sql"
	"github.com/thecodearcher/limen/examples/basic/pkg"
	credentialpassword "github.com/thecodearcher/limen/plugins/credential-password"
	"github.com/thecodearcher/limen/plugins/oauth"
	oauthdiscord "github.com/thecodearcher/limen/plugins/oauth-discord"
	oauthfacebook "github.com/thecodearcher/limen/plugins/oauth-facebook"
	oauthgeneric "github.com/thecodearcher/limen/plugins/oauth-generic"
	oauthgithub "github.com/thecodearcher/limen/plugins/oauth-github"
	oauthgoogle "github.com/thecodearcher/limen/plugins/oauth-google"
	oauthlinkedin "github.com/thecodearcher/limen/plugins/oauth-linkedin"
	oauthmicrosoft "github.com/thecodearcher/limen/plugins/oauth-microsoft"
	oauthspotify "github.com/thecodearcher/limen/plugins/oauth-spotify"
	oauthtwitch "github.com/thecodearcher/limen/plugins/oauth-twitch"
	oauthtwitter "github.com/thecodearcher/limen/plugins/oauth-twitter"
	twofactor "github.com/thecodearcher/limen/plugins/two-factor"
)

type UUIDGenerator struct {
}

func (g *UUIDGenerator) GetColumnType() limen.ColumnType {
	return limen.ColumnTypeUUID
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
func buildOAuthOptions(googleClientID, googleClientSecret, githubClientID, githubClientSecret, facebookClientID, facebookClientSecret, discordClientID, discordClientSecret, microsoftClientID, microsoftClientSecret, twitterClientID, twitterClientSecret, linkedinClientID, linkedinClientSecret, twitchClientID, twitchClientSecret, spotifyClientID, spotifyClientSecret, keycloakDiscoveryURL, keycloakClientID, keycloakClientSecret string) []oauth.ConfigOption {
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
	if facebookClientID != "" && facebookClientSecret != "" {
		opts = append(opts, oauth.WithProvider(oauthfacebook.New(
			oauthfacebook.WithClientID(facebookClientID),
			oauthfacebook.WithClientSecret(facebookClientSecret),
		)))
	}
	if discordClientID != "" && discordClientSecret != "" {
		opts = append(opts, oauth.WithProvider(oauthdiscord.New(
			oauthdiscord.WithClientID(discordClientID),
			oauthdiscord.WithClientSecret(discordClientSecret),
		)))
	}
	if microsoftClientID != "" && microsoftClientSecret != "" {
		opts = append(opts, oauth.WithProvider(oauthmicrosoft.New(
			oauthmicrosoft.WithClientID(microsoftClientID),
			oauthmicrosoft.WithClientSecret(microsoftClientSecret),
		)))
	}
	if twitterClientID != "" && twitterClientSecret != "" {
		opts = append(opts, oauth.WithProvider(oauthtwitter.New(
			oauthtwitter.WithClientID(twitterClientID),
			oauthtwitter.WithClientSecret(twitterClientSecret),
		)))
	}
	if linkedinClientID != "" && linkedinClientSecret != "" {
		opts = append(opts, oauth.WithProvider(oauthlinkedin.New(
			oauthlinkedin.WithClientID(linkedinClientID),
			oauthlinkedin.WithClientSecret(linkedinClientSecret),
		)))
	}
	if twitchClientID != "" && twitchClientSecret != "" {
		opts = append(opts, oauth.WithProvider(oauthtwitch.New(
			oauthtwitch.WithClientID(twitchClientID),
			oauthtwitch.WithClientSecret(twitchClientSecret),
		)))
	}
	if spotifyClientID != "" && spotifyClientSecret != "" {
		opts = append(opts, oauth.WithProvider(oauthspotify.New(
			oauthspotify.WithClientID(spotifyClientID),
			oauthspotify.WithClientSecret(spotifyClientSecret),
			oauthspotify.WithRedirectURL("http://127.0.0.1:8080/api/auth/oauth/spotify/callback"),
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
	opts = append(opts, oauth.WithMapProfileToUser(func(info *limen.OAuthAccountProfile) map[string]any {
		fmt.Printf("Mapping OAuth profile to user additional fields: %+v\n", info)
		switch info.Provider {
		case "google":
			firstName, _ := info.Raw["given_name"].(string)
			lastName, _ := info.Raw["family_name"].(string)
			return map[string]any{"first_name": firstName, "last_name": lastName}
		case "facebook":
			name, _ := info.Raw["name"].(string)
			return map[string]any{"first_name": name, "last_name": ""}
		case "discord":
			username, _ := info.Raw["username"].(string)
			return map[string]any{"first_name": username, "last_name": ""}
		case "microsoft":
			displayName, _ := info.Raw["displayName"].(string)
			return map[string]any{"first_name": displayName, "last_name": ""}
		case "twitter":
			name, _ := info.Raw["name"].(string)
			return map[string]any{"first_name": name, "last_name": ""}
		case "linkedin":
			firstName, _ := info.Raw["given_name"].(string)
			lastName, _ := info.Raw["family_name"].(string)
			return map[string]any{"first_name": firstName, "last_name": lastName}
		case "twitch":
			// OIDC id_token uses preferred_username for display name
			displayName, _ := info.Raw["preferred_username"].(string)
			return map[string]any{"first_name": displayName, "last_name": ""}
		case "spotify":
			displayName, _ := info.Raw["display_name"].(string)
			return map[string]any{"first_name": displayName, "last_name": ""}
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

// buildConfig builds the limen configuration
func buildConfig(db limen.DatabaseAdapter) *limen.Config {

	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	googleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	githubClientID := os.Getenv("GITHUB_CLIENT_ID")
	githubClientSecret := os.Getenv("GITHUB_CLIENT_SECRET")
	facebookClientID := os.Getenv("FACEBOOK_CLIENT_ID")
	facebookClientSecret := os.Getenv("FACEBOOK_CLIENT_SECRET")
	discordClientID := os.Getenv("DISCORD_CLIENT_ID")
	discordClientSecret := os.Getenv("DISCORD_CLIENT_SECRET")
	microsoftClientID := os.Getenv("MICROSOFT_CLIENT_ID")
	microsoftClientSecret := os.Getenv("MICROSOFT_CLIENT_SECRET")
	twitterClientID := os.Getenv("TWITTER_CLIENT_ID")
	twitterClientSecret := os.Getenv("TWITTER_CLIENT_SECRET")
	linkedinClientID := os.Getenv("LINKEDIN_CLIENT_ID")
	linkedinClientSecret := os.Getenv("LINKEDIN_CLIENT_SECRET")
	twitchClientID := os.Getenv("TWITCH_CLIENT_ID")
	twitchClientSecret := os.Getenv("TWITCH_CLIENT_SECRET")
	spotifyClientID := os.Getenv("SPOTIFY_CLIENT_ID")
	spotifyClientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")
	keycloakDiscoveryURL := os.Getenv("KEYCLOAK_DISCOVERY_URL") // e.g. https://keycloak.example.com/realms/master/.well-known/openid-configuration
	keycloakClientID := os.Getenv("KEYCLOAK_CLIENT_ID")
	keycloakClientSecret := os.Getenv("KEYCLOAK_CLIENT_SECRET")

	fmt.Printf("Google Client ID: %s\n", googleClientID)

	return &limen.Config{
		BaseURL: os.Getenv("BASE_URL"),
		// BaseURL:  "https://bat-concise-chamois.ngrok-free.app",
		Database: db,
		Secret:   []byte("rNH8JSJcbiyoPhXk5hQEjbI86SaSIgzw"), // 32 bytes for cookies + plugins (OAuth, 2FA) when they omit their own
		Plugins: []limen.Plugin{
			// sessionjwt.New(
			// // sessionjwt.WithRefreshToken(false),
			// // sessionjwt.WithSigningKey([]byte("rNH8JSJcbiyoPhXk5hQEjbI86SaSIgzw")),
			// // sessionjwt.WithSigningMethod(jwt.SigningMethodRS512),
			// // sessionjwt.WithAccessTokenDuration(15*time.Minute),
			// // sessionjwt.WithRefreshTokenDuration(7*24*time.Hour),
			// // sessionjwt.WithRefreshTokenRotation(true),
			// // sessionjwt.WithBlacklist(true),
			// ),

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
			oauth.New(buildOAuthOptions(googleClientID, googleClientSecret, githubClientID, githubClientSecret, facebookClientID, facebookClientSecret, discordClientID, discordClientSecret, microsoftClientID, microsoftClientSecret, twitterClientID, twitterClientSecret, linkedinClientID, linkedinClientSecret, twitchClientID, twitchClientSecret, spotifyClientID, spotifyClientSecret, keycloakDiscoveryURL, keycloakClientID, keycloakClientSecret)...),
			twofactor.New(
				// twofactor.WithCookieExpiration(2*time.Minute),
				twofactor.WithSecret("limen_2fa_totp_secret_1234567890"),
				twofactor.WithTOTP(
					twofactor.WithTOTPIssuer("Limen"),
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
		CLI: &limen.CLIConfig{
			Enabled: true,
		},
		Schema: limen.NewDefaultSchemaConfig(
			// limen.WithSchemaIDGenerator(&UUIDGenerator{}),
			limen.WithSchemaUser(
				// limen.WithUserTableName("usersz_from_personal_user_schema"),
				// limen.WithUserFieldID("id_from_personal"),
				limen.WithUserFieldEmailVerifiedAt("email_verified"),
				// limen.WithUserFieldEmail("email_from_personal"),
				limen.WithUserAdditionalFields(func(ctx *limen.AdditionalFieldsContext) (map[string]any, error) {
					// if ctx.IsEmpty("firstname") {
					// 	return nil, limen.NewLimenError("firstname is required", http.StatusBadRequest, nil)
					// }
					// if ctx.IsEmpty("lastname") {
					// 	return nil, limen.NewLimenError("lastname is required", http.StatusBadRequest, nil)
					// }
					return map[string]any{
						"uuid":       uuid.New().String(),
						"first_name": ctx.GetBodyValue("firstname"),
						"last_name":  ctx.GetBodyValue("lastname"),
						"updated_at": time.Now().Format(time.RFC3339),
					}, nil
				}),

				// limen.WithUserSerializer(func(data *limen.User) map[string]any {
				// 	return map[string]any{
				// 		"id":                data.ID,
				// 		"email":             data.Email,
				// 		"password":          data.Password,
				// 		"email_verified_at": data.EmailVerifiedAt,
				// 	}
				// }),
			),
			limen.WithSchemaVerification(
				limen.WithVerificationAdditionalFields(func(ctx *limen.AdditionalFieldsContext) (map[string]any, error) {
					return map[string]any{
						// "uuid":       uuid.New().String(),
						"created_at": time.Now().Format(time.RFC3339),
						"updated_at": time.Now().Format(time.RFC3339),
					}, nil
				}),
			),
			// Example: Customize plugin schema table and field names

			limen.WithPluginSchema(limen.PluginCredentialPassword, "something_map_name2",
				limen.WithPluginFieldName("name", "name_from_plugin"),
			),
		),
		// Schema: limen.SchemaConfig{
		// 	// AdditionalFields: func(ctx *schemas.AdditionalFieldsContext) map[string]any {
		// 	// 	return map[string]any{
		// 	// 		"uuid":       uuid.New().String(),
		// 	// 		"created_at": time.Now(),
		// 	// 		"updated_at": time.Now(),
		// 	// 	}
		// 	// },
		// 	User: limen.UserSchema{
		// 		Fields: limen.UserFields{
		// 			EmailVerifiedAt: "email_verified",
		// 		},
		// 		AdditionalFields: func(ctx *limen.AdditionalFieldsContext) (map[string]any, *limen.LimenError) {
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
		Session: limen.NewDefaultSessionConfig(
			// limen.WithSessionStoreType(limen.SessionStoreTypeMemory),
			// limen.WithSessionStrategy(limen.SessionStrategyServerSide),
			limen.WithSessionUpdateAge(10 * time.Second),
		),
		HTTP: limen.NewDefaultHTTPConfig(
			limen.WithHTTPBasePath("/api/auth"),
			limen.WithHTTPSessionCookieName("session"),
			limen.WithHTTPCookieSecure(false),
			limen.WithHTTPRateLimiter(limen.WithRateLimiterDisableForPaths("/me", "/signin/email")),

			limen.WithHTTPSessionTransformer(sessionTransformer),
			limen.WithHTTPTrustedOrigins([]string{
				"*",
				"http://localhost:8080",
				"*.localhost:3000", "http://localhost:3000",
				"myapp://",                             // Mobile app scheme
				"chrome-extension://YOUR_EXTENSION_ID", // Browser extension
				"exp://*/*",                            // Trust all Expo development URLs
				"exp://10.0.0.*:*/*",                   // Trust 10.0.0.x IP range with any port,
				// "*.example.com",
				"https://*.example.com",
				"http://*.dev.example.com",
			}),
			limen.WithHTTPHooks(&limen.Hooks{
				Before: []*limen.Hook{
					{
						PathMatcher: func(ctx *limen.HookContext) bool {
							return true
						},
						Run: func(ctx *limen.HookContext) bool {
							fmt.Printf("Before request %s %s\n", ctx.Method(), ctx.Path())
							fmt.Printf("Before request route pattern: %+v\n", ctx.RoutePattern())
							return true
						},
					},
					{
						PathMatcher: func(ctx *limen.HookContext) bool {
							return ctx.RouteID() == "signup"
						},
						Run: func(ctx *limen.HookContext) bool {
							email, ok := ctx.GetJSONBodyValue("email").(string)
							if !ok {
								return true
							}

							if !strings.Contains(email, "@example.com") {
								ctx.WriteErrorResponse(limen.NewLimenError("email domain not allowed", http.StatusBadRequest, nil))
								return false
							}

							return true
						},
					},
				},
			}),
		),
		// 	limen.WithRateLimiterWindow(time.Minute),

		// limen.WithRateLimiterStore(limen.RateLimiterStoreTypeDatabase),
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

// Example showing basic usage of the limen library
func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Printf("Failed to load .env file: %v", err)
	}
	fmt.Println(pkg.SomeShi())
	fmt.Println("Limen Authentication Library - Basic Example")
	// dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
	// 	"localhost",
	// 	"root",
	// 	"root",
	// 	"limen",
	// 	"5432",
	// )
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatalf("DATABASE_URL is not set")
	}

	gormdb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Info)})

	// mysqlDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", "root", "", "localhost", "3306", "limen")
	// db, err := sql.Open("mysql", mysqlDSN)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	// db = sqldblogger.OpenDriver(mysqlDSN, db.Driver(), loggerAdapter /*, using_default_options*/) // db is STILL *sql.DB

	// sqldbAdapter := sqladapter.NewMySQL(db).WithLogger(&logger{})
	config := buildConfig(gormadapter.New(gormdb))

	auth, err := limen.New(config)
	if err != nil {
		log.Fatalf("Failed to create limen: %v", err)
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
	//   oauthAPI, ok := limen.UsePlugin[oauth.API](auth, limen.PluginOAuth)
	_ = credentialpassword.Use(auth)
	_ = oauth.Use(auth)
	_ = twofactor.Use(auth)

	handler := auth.Handler()

	// schemas, err := limen.DiscoverAllSchemasFromConfig(config)
	// if err != nil {
	// 	log.Fatalf("Failed to discover all schemas: %v", err)
	// }

	// fmt.Printf("Schemas: %+v\n", schemas)
	// copyConfig := &config

	// migrations, err := limen.GenerateMigrations(copyConfig, adapter.NewMigrationGenerator("postgres"))
	// if err != nil {
	// 	log.Fatalf("Failed to generate migrations: %v", err)
	// }
	// fmt.Printf("Migrations: %+v\n", migrations)
	// code, err := limen.GenerateGoStructsFromConfig(config, limen.GenerateOptions{
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
	r.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			return strings.HasPrefix(origin, "http://localhost:") ||
				strings.HasPrefix(origin, "https://localhost:") ||
				strings.HasSuffix(origin, ".ngrok-free.app") ||
				strings.HasSuffix(origin, "appleid.apple.com") ||
				strings.HasSuffix(origin, ".limenauth.dev")
		},
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodHead,
			http.MethodOptions,
		},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept", "X-Requested-With"},
		ExposeHeaders:    []string{"Set-Auth-Token", "Set-Refresh-Token"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.POST("/", func(c *gin.Context) {
		// session, err := auth.GetSession(c.Request)
		// if err != nil {
		// 	c.JSON(500, gin.H{"message": "Failed to get session"})
		// 	return
		// }
		// fmt.Printf("Session: %+v\n", session)
		form := url.Values{}

		c.Bind(&form)
		fmt.Printf("Request: %+v\n", c.Request)
		fmt.Printf("Request Body: %+v\n", form)
		fmt.Printf("Request Form: %+v\n", form)
		fmt.Printf("Request Multipart Form: %+v\n", c.Request.MultipartForm)
		fmt.Printf("Request URL: %+v\n", c.Request.URL)
		fmt.Printf("Request Header: %+v\n", c.Request.Header)
		fmt.Printf("Request Host: %+v\n", c.Request.Host)
		fmt.Printf("Request Content Length: %+v\n", c.Request.ContentLength)
		fmt.Printf("Request Transfer Encoding: %+v\n", c.Request.TransferEncoding)
		http.Redirect(c.Writer, c.Request, "http://localhost:3001", 302)
	})

	r.Any("/api/auth/*path", func(c *gin.Context) {
		handler.ServeHTTP(c.Writer, c.Request)
	})

	http.ListenAndServe(":8080", r)

}

func sessionTransformer(user map[string]any, sessionResult *limen.SessionResult) (map[string]any, error) {
	payload := map[string]any{
		"user": user,
	}
	if sessionResult != nil {
		payload["token"] = sessionResult.Token
	}
	return payload, nil
}
