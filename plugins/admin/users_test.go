package admin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thecodearcher/limen"
)

func TestAdminPlugin_ListUsers_Search(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAdminPlugin(t)
	actor := seedAdmin(t, l, plugin, "actor@test.com")
	limen.SeedTestUser(t, l, "ada@test.com")
	limen.SeedTestUser(t, l, "bob@test.com")

	page, err := plugin.ListUsers(t.Context(), actor, &ListUsersQuery{Search: "ada"}, nil)
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, "ada@test.com", page.Items[0].Email)
}

func newNicknamePlugin(t *testing.T) limen.Plugin {
	t.Helper()
	return &nicknamePlugin{}
}

type nicknamePlugin struct{}

func (p *nicknamePlugin) Name() limen.PluginName {
	return "nickname-test"
}

func (p *nicknamePlugin) Initialize(*limen.LimenCore) error {
	return nil
}

func (p *nicknamePlugin) PluginHTTPConfig() limen.PluginHTTPConfig {
	return limen.PluginHTTPConfig{}
}

func (p *nicknamePlugin) RegisterRoutes(*limen.LimenHTTPCore, *limen.RouteBuilder) {}

func (p *nicknamePlugin) GetSchemas(schema *limen.SchemaConfig) []limen.SchemaIntrospector {
	return []limen.SchemaIntrospector{
		limen.NewSchemaDefinitionForExtension(limen.CoreSchemaUsers, schema.User,
			limen.WithSchemaField("nickname", limen.ColumnTypeString, limen.WithNullable(true)),
		),
	}
}

func TestAdminPlugin_ListUsers_SearchExtraColumn(t *testing.T) {
	t.Parallel()

	plugin := New(WithUserSearchColumns("nickname"))
	l, _ := limen.NewTestLimen(t, plugin, newNicknamePlugin(t))
	actor := seedAdmin(t, l, plugin, "actor@test.com")
	_, err := plugin.CreateUser(t.Context(), actor, &CreateUserRequest{Email: "ada@test.com", AdditionalFields: map[string]any{"nickname": "acme"}})
	require.NoError(t, err)

	page, err := plugin.ListUsers(t.Context(), actor, &ListUsersQuery{Search: "acme"}, nil)
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, "ada@test.com", page.Items[0].Email)
}

func TestAdminPlugin_ListUsers_PermissionDenied(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAdminPlugin(t)
	actor := limen.SeedTestUser(t, l, "member@test.com")

	_, err := plugin.ListUsers(t.Context(), actor, nil, nil)
	assert.ErrorIs(t, err, ErrInsufficientPermission)
}

func TestAdminPlugin_CreateUser(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAdminPlugin(t)
	actor := seedAdmin(t, l, plugin, "actor@test.com")

	user, err := plugin.CreateUser(t.Context(), actor, &CreateUserRequest{
		Email:            "new@test.com",
		EmailVerified:    true,
		AdditionalFields: map[string]any{"first_name": "Ada"},
	})
	require.NoError(t, err)
	assert.NotNil(t, user.EmailVerifiedAt)
	assert.Equal(t, "Ada", user.Raw()["first_name"])
}

func TestAdminPlugin_CreateUser_EmailAlreadyExists(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAdminPlugin(t)
	actor := seedAdmin(t, l, plugin, "actor@test.com")
	limen.SeedTestUser(t, l, "taken@test.com")

	_, err := plugin.CreateUser(t.Context(), actor, &CreateUserRequest{Email: "Taken@test.com"})
	assert.ErrorIs(t, err, ErrEmailAlreadyExists)
}

func TestAdminPlugin_UpdateUser(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAdminPlugin(t)
	actor := seedAdmin(t, l, plugin, "actor@test.com")
	target, err := plugin.CreateUser(t.Context(), actor, &CreateUserRequest{Email: "target@test.com", EmailVerified: true})
	require.NoError(t, err)

	unverified := false
	user, err := plugin.UpdateUser(t.Context(), actor, target.ID, &UpdateUserRequest{
		EmailVerified:    &unverified,
		AdditionalFields: map[string]any{"first_name": "Ada"},
	})
	require.NoError(t, err)
	assert.Nil(t, user.EmailVerifiedAt)
	assert.Equal(t, "Ada", user.Raw()["first_name"])
}

func TestAdminPlugin_UpdateUser_EmptyNameClears(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAdminPlugin(t)
	actor := seedAdmin(t, l, plugin, "actor@test.com")
	ada := "Ada"
	target, err := plugin.CreateUser(t.Context(), actor, &CreateUserRequest{Email: "target@test.com", FirstName: &ada})
	require.NoError(t, err)

	empty := ""
	user, err := plugin.UpdateUser(t.Context(), actor, target.ID, &UpdateUserRequest{FirstName: &empty})
	require.NoError(t, err)
	assert.Nil(t, user.Raw()["first_name"])
}

func TestAdminPlugin_CreateUser_NameFieldsDisabled(t *testing.T) {
	t.Parallel()

	plugin := New()
	l, _ := limen.NewTestLimenWithSchema(t, limen.NewDefaultSchemaConfig(limen.WithSchemaUser(limen.WithUserIncludeNameFields(false))), plugin)
	actor := seedAdmin(t, l, plugin, "actor@test.com")
	ada := "Ada"

	_, err := plugin.CreateUser(t.Context(), actor, &CreateUserRequest{Email: "target@test.com", FirstName: &ada})
	assert.NoError(t, err)
}

func TestAdminPlugin_SetEmail_EmailAlreadyExists(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAdminPlugin(t)
	actor := seedAdmin(t, l, plugin, "actor@test.com")
	target := limen.SeedTestUser(t, l, "target@test.com")
	limen.SeedTestUser(t, l, "taken@test.com")

	_, err := plugin.SetEmail(t.Context(), actor, target.ID, &SetEmailRequest{Email: "Taken@Test.com"})
	assert.ErrorIs(t, err, ErrEmailAlreadyExists)
}

func TestAdminPlugin_SetEmail_VerifiedState(t *testing.T) {
	t.Parallel()

	testData := []struct {
		name         string
		req          SetEmailRequest
		wantVerified bool
		wantToken    bool
	}{
		{name: "new email is unverified", req: SetEmailRequest{Email: "new@test.com"}},
		{name: "marked verified", req: SetEmailRequest{Email: "new@test.com", EmailVerified: true}, wantVerified: true},
		{name: "send verification wins", req: SetEmailRequest{Email: "new@test.com", EmailVerified: true, SendVerification: true}, wantToken: true},
	}
	for _, tt := range testData {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l, plugin := newTestAdminPlugin(t)
			actor := seedAdmin(t, l, plugin, "actor@test.com")
			target, err := plugin.CreateUser(t.Context(), actor, &CreateUserRequest{Email: "old@test.com", EmailVerified: true})
			require.NoError(t, err)

			user, err := plugin.SetEmail(t.Context(), actor, target.ID, &tt.req)
			require.NoError(t, err)
			assert.Equal(t, "new@test.com", user.Email)
			assert.Equal(t, tt.wantVerified, user.EmailVerifiedAt != nil)

			_, err = plugin.core.DBAction.FindVerificationByAction(t.Context(), limen.EmailVerificationAction, "new@test.com")
			assert.Equal(t, tt.wantToken, err == nil)
		})
	}
}

func TestAdminPlugin_SetEmail_SameEmailKeepsVerified(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAdminPlugin(t)
	actor := seedAdmin(t, l, plugin, "actor@test.com")
	target, err := plugin.CreateUser(t.Context(), actor, &CreateUserRequest{Email: "same@test.com", EmailVerified: true})
	require.NoError(t, err)

	user, err := plugin.SetEmail(t.Context(), actor, target.ID, &SetEmailRequest{Email: "Same@Test.com"})
	require.NoError(t, err)
	assert.NotNil(t, user.EmailVerifiedAt)
}

func TestAdminPlugin_DeleteUser(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAdminPluginWithPublicIDs(t)
	actor := seedAdmin(t, l, plugin, "actor@test.com")
	target := limen.SeedTestUser(t, l, "target@test.com")
	limen.SeedTestSession(t, l, target.ID, target.Email)
	publicID, ok := plugin.core.EncodePublicID(plugin.core.Schema.User, target)
	require.True(t, ok)

	require.NoError(t, plugin.DeleteUser(t.Context(), actor, publicID))

	_, err := plugin.core.DBAction.FindUserByID(t.Context(), target.ID)
	assert.ErrorIs(t, err, limen.ErrRecordNotFound)
	sessions, err := plugin.core.SessionManager.ListSessions(t.Context(), target.ID)
	assert.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestAdminPlugin_DeleteUser_Refused(t *testing.T) {
	t.Parallel()

	testData := []struct {
		name    string
		self    bool
		wantErr error
	}{
		{name: "self", self: true, wantErr: ErrCannotDeleteSelf},
		{name: "last admin", self: false, wantErr: ErrLastAdmin},
	}
	for _, tt := range testData {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l, plugin := newTestAdminPlugin(t, WithAdminUserIDs("root-admin"))
			err := plugin.core.DBAction.CreateUser(t.Context(), &limen.User{Email: "root@test.com"}, map[string]any{"id": "root-admin"})
			require.NoError(t, err)
			root, err := plugin.core.DBAction.FindUserByID(t.Context(), "root-admin")
			require.NoError(t, err)
			admin := seedAdmin(t, l, plugin, "admin@test.com")

			actor, target := root, admin
			if tt.self {
				actor, target = admin, admin
			}

			err = plugin.DeleteUser(t.Context(), actor, target.ID)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}
