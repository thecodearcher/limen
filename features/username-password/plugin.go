package usernamepassword

import (
	"context"
	"fmt"
	"maps"
	"time"

	"github.com/thecodearcher/aegis"
	"github.com/thecodearcher/aegis/pkg/httpx"
)

type usernamePasswordFeature struct {
	core       *aegis.AegisCore
	config     *config
	userSchema *UsernamePasswordUserSchema
	dbAction   *aegis.DatabaseActionHelper
}

func (p *usernamePasswordFeature) Name() aegis.FeatureName {
	return aegis.FeatureUsernamePassword
}

const (
	UserSchemaUsernameField aegis.SchemaField = "username"
)

func (p *usernamePasswordFeature) GetSchemas(schema *aegis.SchemaConfig) []aegis.SchemaIntrospector {
	userWithUsername := NewUsernamePasswordUserSchema(schema)
	p.userSchema = userWithUsername
	extension1 := aegis.NewSchemaDefinitionForExtension(
		aegis.CoreSchemaUsers,
		userWithUsername,
		aegis.WithSchemaField("username", aegis.ColumnTypeString),
		aegis.WithSchemaIndex("idx_users_username", []aegis.SchemaField{UserSchemaUsernameField}),
	)

	// extension2 := aegis.NewPluginSchemaForExtension(
	// 	aegis.CoreSchemaVerifications,
	// 	p.core.Schema.Verification,
	// 	aegis.WithPluginSchemaField("id_token", aegis.ColumnTypeString),
	// )

	// table := aegis.NewSchemaDefinitionForTable(
	// 	aegis.SchemaName("users_username"),
	// 	aegis.SchemaTableName("users_username"),
	// 	p.userSchema,
	// 	aegis.WithSchemaField("username", aegis.ColumnTypeString),
	// 	aegis.WithSchemaIndex("idx_users_username", []aegis.SchemaField{UserSchemaUsernameField}),
	// )

	return []aegis.SchemaIntrospector{extension1}
}

func (p *usernamePasswordFeature) Initialize(core *aegis.AegisCore) error {
	p.core = core
	p.dbAction = core.DBAction

	if p.config == nil {
		return fmt.Errorf("config is required")
	}

	if p.config.usernameMinLength < 1 {
		return fmt.Errorf("username min length must be at least 1")
	}

	if p.config.usernameMaxLength < p.config.usernameMinLength {
		return fmt.Errorf("username max length must be greater than or equal to min length")
	}

	// Note: EmailPasswordFeature reference will be set after all features are initialized
	// via SetEmailPasswordFeature() method

	return nil
}

func (i *usernamePasswordFeature) FindUserByUsername(ctx context.Context, username string) (*UserWithUsername, error) {
	user, err := i.core.FindOne(ctx, i.userSchema, []aegis.Where{
		aegis.Eq(i.userSchema.GetUsernameField(), username),
	}, nil)

	if err != nil {
		return nil, err
	}

	return user.(*UserWithUsername), nil
}

func (p *usernamePasswordFeature) SignInWithUsernameAndPassword(ctx context.Context, username string, password string) (*aegis.AuthenticationResult, error) {
	if p.config.emailPassword == nil {
		return nil, ErrEmailPasswordNotEnabled
	}

	user, err := p.FindUserByUsername(ctx, username)
	if err != nil {
		// hash the password to avoid timing attacks when the user is not found
		// this allows constant response time for both valid and invalid credentials
		_, _ = p.config.emailPassword.HashPassword(password)
		return nil, ErrUsernameNotFound
	}

	isValid, err := p.config.emailPassword.ComparePassword(password, user.Password)
	if err != nil {
		return nil, err
	}

	if !isValid {
		return nil, ErrAPIInvalidCredentials
	}

	// Check for email verification requirement (delegate to email-password config if available)
	// For now, we'll return empty pending actions since username plugin doesn't manage email verification
	pendingActions := []aegis.PendingAction{}

	return &aegis.AuthenticationResult{
		User:           user.User,
		PendingActions: pendingActions,
	}, nil
}

func (p *usernamePasswordFeature) SignUpWithUsernameAndPassword(ctx context.Context, user *aegis.User, username string, additionalFields map[string]any) (*aegis.AuthenticationResult, error) {
	if p.config.emailPassword == nil {
		return nil, ErrEmailPasswordNotEnabled
	}

	if err := p.validateUser(username, user); err != nil {
		return nil, err
	}

	// Check if username already exists
	usernameExists, err := p.core.Exists(ctx, p.userSchema, []aegis.Where{
		aegis.Eq(p.userSchema.GetUsernameField(), username),
	})
	if err != nil {
		return nil, err
	}

	if usernameExists {
		return nil, ErrUsernameAlreadyExists
	}

	// Check if email already exists (required for password reset via email-password)
	if user.Email == "" {
		return nil, fmt.Errorf("email is required for username signup (needed for password reset)")
	}

	emailExists, err := p.core.Exists(ctx, p.userSchema, []aegis.Where{
		aegis.Eq(p.userSchema.GetEmailField(), user.Email),
	})
	if err != nil {
		return nil, err
	}

	if emailExists {
		return nil, fmt.Errorf("email already exists")
	}

	// Hash password using email-password plugin
	hashedPassword, err := p.config.emailPassword.HashPassword(user.Password)
	if err != nil {
		return nil, err
	}

	payload := make(map[string]any)
	maps.Copy(payload, additionalFields)
	maps.Copy(payload, map[string]any{
		p.userSchema.GetUsernameField(): username,
	})

	// Username should already be in additionalFields from the API layer
	// Create user with username in additionalFields
	err = p.dbAction.CreateUser(ctx, &aegis.User{
		Email:    user.Email,
		Password: hashedPassword,
	}, payload)

	if err != nil {
		return nil, err
	}

	// Fetch the created user
	createdUser, err := p.FindUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	pendingActions := []aegis.PendingAction{}

	return &aegis.AuthenticationResult{
		User:           createdUser.User,
		PendingActions: pendingActions,
	}, nil
}

func (p *usernamePasswordFeature) validateUsername(username string) error {
	if username == "" {
		return ErrUsernameRequired
	}

	if len(username) < p.config.usernameMinLength {
		return ErrUsernameTooShort
	}

	if len(username) > p.config.usernameMaxLength {
		return ErrUsernameTooLong
	}

	if p.config.usernameValidationRegex != nil && !p.config.usernameValidationRegex.MatchString(username) {
		return ErrUsernameInvalidFormat
	}

	return nil
}

func (p *usernamePasswordFeature) validateUser(username string, user *aegis.User) error {
	if err := p.validateUsername(username); err != nil {
		return err
	}

	if user.Email == "" {
		return fmt.Errorf("email is required")
	}

	if user.Password == "" {
		return fmt.Errorf("password is required")
	}

	// Delegate password validation to email-password plugin
	// Note: We can't directly validate password here since email-password doesn't expose validatePassword
	// But password will be validated when hashing via email-password's HashPassword method

	return nil
}

func (p *usernamePasswordFeature) PluginHTTPConfig() aegis.PluginHTTPConfig {
	return aegis.PluginHTTPConfig{
		Middleware: []httpx.Middleware{},
		RateLimitRules: []*aegis.RateLimitRule{
			aegis.NewRateLimitRule("/signin/username", 5, 10*time.Second),
			aegis.NewRateLimitRule("/signup/username", 5, 10*time.Second),
		},
	}
}

func (p *usernamePasswordFeature) RegisterRoutes(httpCore *aegis.AegisHTTPCore, routeBuilder *aegis.RouteBuilder) {
	api := NewUsernamePasswordAPI(p, httpCore, routeBuilder)
	routes(api)
}
