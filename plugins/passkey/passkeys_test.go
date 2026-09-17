package passkey

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thecodearcher/limen"
)

func TestBeginRegistrationExcludeListContainsExistingCredential(t *testing.T) {
	t.Parallel()

	plugin, l := newTestPasskeyPlugin(t)
	user := limen.SeedTestUser(t, l, "exclude@example.com")
	const credentialID = "ZXhjbHVkZS1jcmVkZW50aWFs"
	seedTestPasskey(t, plugin, user.ID, credentialID)

	creation, _, err := plugin.BeginRegistration(
		httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/passkeys/begin-registration", http.NoBody),
		user,
		&RegisterPasskeyRequest{},
	)
	require.NoError(t, err)
	require.Len(t, creation.Response.CredentialExcludeList, 1)

	want, err := base64.RawURLEncoding.DecodeString(credentialID)
	require.NoError(t, err)
	assert.Equal(t, want, []byte(creation.Response.CredentialExcludeList[0].CredentialID))
}

func TestBeginRegistrationAuthenticatorAttachmentCopiesUVAndResidentKey(t *testing.T) {
	t.Parallel()

	plugin, l := newTestPasskeyPlugin(t,
		WithAuthenticatorSelection(AuthenticatorSelection{
			UserVerification: UserVerificationRequired,
			ResidentKey:      ResidentKeyRequired,
		}),
	)
	user := limen.SeedTestUser(t, l, "attachment@example.com")

	creation, _, err := plugin.BeginRegistration(
		httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/passkeys/begin-registration", http.NoBody),
		user,
		&RegisterPasskeyRequest{AuthenticatorAttachment: string(protocol.Platform)},
	)
	require.NoError(t, err)

	selection := creation.Response.AuthenticatorSelection
	assert.Equal(t, protocol.Platform, selection.AuthenticatorAttachment)
	assert.Equal(t, protocol.VerificationRequired, selection.UserVerification)
	assert.Equal(t, protocol.ResidentKeyRequirementRequired, selection.ResidentKey)
}

func TestBeginPublicRegistrationFailedAttestationDoesNotCreateUser(t *testing.T) {
	t.Parallel()

	createCalls := 0
	var plugin *passkeyPlugin
	plugin, _ = newTestPublicRegistrationPlugin(t, func(
		ctx context.Context,
		r *http.Request,
		intent *RegistrationIntent,
		reservedHandle string,
	) (*limen.User, error) {
		createCalls++
		return &limen.User{Email: intent.Email}, nil
	})

	creation, challenge, err := plugin.BeginPublicRegistration(
		httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/passkeys/begin-registration", http.NoBody),
		&RegisterPasskeyRequest{Context: "junk@example.com"},
	)
	require.NoError(t, err)
	require.NotNil(t, creation)
	require.NotEmpty(t, challenge.ReservedHandle)

	req := requestWithChallenge(t, plugin, http.MethodPost, "/passkeys/finish-registration", []byte(`{"id":"nope"}`), challenge)
	_, _, err = plugin.FinishPublicRegistration(req, &FinishRegistrationRequest{})
	require.Error(t, err)
	assert.Equal(t, 0, createCalls)
}

func TestFinishPublicRegistrationHandleMatchAndMismatch(t *testing.T) {
	t.Parallel()

	cred := loadFixtureCredential(t)
	challenge := loadRegistrationChallenge(t)
	challenge.ReservedHandle = cred.UserHandle
	challenge.Intent = &RegistrationIntent{Email: "fixture@example.com", ID: cred.UserHandle}

	t.Run("matching handle succeeds without insert in hook", func(t *testing.T) {
		t.Parallel()

		var plugin *passkeyPlugin
		plugin, _ = newTestPublicRegistrationPlugin(t, func(
			ctx context.Context,
			r *http.Request,
			intent *RegistrationIntent,
			reservedHandle string,
		) (*limen.User, error) {
			column := plugin.core.PublicIDColumn(plugin.core.Schema.User)
			return plugin.core.Schema.User.FromStorage(map[string]any{
				plugin.core.Schema.User.GetEmailField(): intent.Email,
				column:                                  reservedHandle,
			}).(*limen.User), nil
		})

		req := requestWithChallenge(
			t,
			plugin,
			http.MethodPost,
			"/passkeys/finish-registration",
			loadRegistrationResponse(t),
			challenge,
		)
		user, passkey, err := plugin.FinishPublicRegistration(req, &FinishRegistrationRequest{})
		require.NoError(t, err)
		require.NotNil(t, user)
		require.NotNil(t, passkey)
		assert.Equal(t, cred.CredentialID, passkey.CredentialID)
	})

	t.Run("mismatched handle returns ErrHandleMismatch", func(t *testing.T) {
		t.Parallel()

		var plugin *passkeyPlugin
		plugin, _ = newTestPublicRegistrationPlugin(t, func(
			ctx context.Context,
			r *http.Request,
			intent *RegistrationIntent,
			reservedHandle string,
		) (*limen.User, error) {
			column := plugin.core.PublicIDColumn(plugin.core.Schema.User)
			return plugin.core.Schema.User.FromStorage(map[string]any{
				plugin.core.Schema.User.GetEmailField(): intent.Email,
				column:                                  "different-handle",
			}).(*limen.User), nil
		})

		req := requestWithChallenge(
			t,
			plugin,
			http.MethodPost,
			"/passkeys/finish-registration",
			loadRegistrationResponse(t),
			challenge,
		)
		_, _, err := plugin.FinishPublicRegistration(req, &FinishRegistrationRequest{})
		require.ErrorIs(t, err, ErrHandleMismatch)
	})
}

func TestBeginPublicRegistrationIntentIDIsReservedHandle(t *testing.T) {
	t.Parallel()

	explicit := nextHandle()
	var plugin *passkeyPlugin
	plugin, _ = newTestPublicRegistrationPlugin(t, func(
		ctx context.Context,
		r *http.Request,
		intent *RegistrationIntent,
		reservedHandle string,
	) (*limen.User, error) {
		return &limen.User{Email: intent.Email}, nil
	}, WithPrepareRegistration(func(ctx context.Context, r *http.Request, registrationContext string) (*RegistrationIntent, error) {
		return &RegistrationIntent{Email: registrationContext, ID: explicit}, nil
	}))

	_, challenge, err := plugin.BeginPublicRegistration(
		httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/passkeys/begin-registration", http.NoBody),
		&RegisterPasskeyRequest{Context: "intent-id@example.com"},
	)
	require.NoError(t, err)
	assert.Equal(t, explicit, challenge.ReservedHandle)
	assert.Equal(t, []byte(explicit), challenge.Session.UserID)
}

func TestInternalHandleModeUsesGeneratedID(t *testing.T) {
	t.Parallel()

	generator := &stringIDGenerator{}
	plugin := New(
		WithRequireSession(false),
		WithAllowInternalUserIDAsHandle(true),
		WithPrepareRegistration(func(ctx context.Context, r *http.Request, registrationContext string) (*RegistrationIntent, error) {
			return &RegistrationIntent{Email: registrationContext}, nil
		}),
		WithCreateRegistrationUser(func(
			ctx context.Context,
			r *http.Request,
			intent *RegistrationIntent,
			reservedHandle string,
		) (*limen.User, error) {
			return &limen.User{ID: reservedHandle, Email: intent.Email}, nil
		}),
	)
	schema := limen.NewDefaultSchemaConfig(limen.WithSchemaIDGenerator(generator))
	_, _ = limen.NewTestLimenWithSchema(t, schema, plugin)

	_, challenge, err := plugin.BeginPublicRegistration(
		httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/passkeys/begin-registration", http.NoBody),
		&RegisterPasskeyRequest{Context: "generated@example.com"},
	)
	require.NoError(t, err)
	assert.Equal(t, "uid_001", challenge.ReservedHandle)
	assert.Equal(t, []byte("uid_001"), challenge.Session.UserID)
}

func TestInternalHandleModeWithoutGeneratorOrIntentIDFails(t *testing.T) {
	t.Parallel()

	plugin := New(
		WithRequireSession(false),
		WithAllowInternalUserIDAsHandle(true),
		WithPrepareRegistration(func(ctx context.Context, r *http.Request, registrationContext string) (*RegistrationIntent, error) {
			return &RegistrationIntent{Email: registrationContext}, nil
		}),
		WithCreateRegistrationUser(func(
			ctx context.Context,
			r *http.Request,
			intent *RegistrationIntent,
			reservedHandle string,
		) (*limen.User, error) {
			return &limen.User{Email: intent.Email}, nil
		}),
	)
	_, core := limen.NewTestLimen(t)
	require.NoError(t, plugin.Initialize(core))

	_, err := plugin.reserveRegistrationHandle(t.Context(), &RegistrationIntent{Email: "nogenerator@example.com"})
	require.ErrorIs(t, err, ErrIDGeneratorRequired)
}

func TestBeginRegistrationWithoutOpaqueIDFails(t *testing.T) {
	t.Parallel()

	plugin, _ := newTestPasskeyPlugin(t)
	user := plugin.core.Schema.User.FromStorage(map[string]any{
		plugin.core.Schema.User.GetEmailField(): "opaque@example.com",
		plugin.core.Schema.User.GetIDField():    "1",
	}).(*limen.User)

	_, _, err := plugin.BeginRegistration(
		httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/passkeys/begin-registration", http.NoBody),
		user,
		&RegisterPasskeyRequest{},
	)
	require.ErrorIs(t, err, ErrHandleMismatch)
}

func TestListPasskeysReturnsOnlyCallerPasskeys(t *testing.T) {
	t.Parallel()

	plugin, l := newTestPasskeyPlugin(t)
	owner := limen.SeedTestUser(t, l, "owner@example.com")
	other := limen.SeedTestUser(t, l, "other@example.com")
	seedTestPasskey(t, plugin, owner.ID, "b3duZXItY3JlZA")
	seedTestPasskey(t, plugin, other.ID, "b3RoZXItY3JlZA")

	page, err := plugin.ListPasskeys(t.Context(), owner, nil)
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, "b3duZXItY3JlZA", page.Items[0].CredentialID)
}

func TestUpdateAndDeleteOtherUsersPasskeyNotFound(t *testing.T) {
	t.Parallel()

	plugin, l := newTestPasskeyPlugin(t)
	owner := limen.SeedTestUser(t, l, "owner-manage@example.com")
	intruder := limen.SeedTestUser(t, l, "intruder@example.com")
	passkey := seedTestPasskey(t, plugin, owner.ID, "bWFuYWdlLWNyZWQ")

	_, err := plugin.UpdatePasskey(t.Context(), intruder, passkey.ID, &UpdatePasskeyRequest{Name: "stolen"})
	require.ErrorIs(t, err, ErrPasskeyNotFound)

	err = plugin.DeletePasskey(t.Context(), intruder, passkey.ID)
	require.ErrorIs(t, err, ErrPasskeyNotFound)

	found, err := plugin.FindPasskeyByCredentialID(t.Context(), passkey.CredentialID)
	require.NoError(t, err)
	assert.Equal(t, owner.ID, found.UserID)
}

func TestRegistrationExtensionsAppearInBeginOptions(t *testing.T) {
	t.Parallel()

	plugin, l := newTestPasskeyPlugin(t, WithRegistrationExtensions(AuthenticationExtensions{
		CredProps: true,
	}))
	user := limen.SeedTestUser(t, l, "ext@example.com")

	creation, _, err := plugin.BeginRegistration(
		httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/passkeys/begin-registration", http.NoBody),
		user,
		&RegisterPasskeyRequest{},
	)
	require.NoError(t, err)
	assert.True(t, creation.Response.Extensions.CredProps)
}

func TestRegistrationExtensionsResolverOutputAndError(t *testing.T) {
	t.Parallel()

	t.Run("resolver output appears", func(t *testing.T) {
		t.Parallel()

		plugin, l := newTestPasskeyPlugin(t, WithRegistrationExtensionsResolver(
			func(ctx context.Context, r *http.Request) (AuthenticationExtensions, error) {
				return AuthenticationExtensions{CredProps: true}, nil
			},
		))
		user := limen.SeedTestUser(t, l, "resolver@example.com")

		creation, _, err := plugin.BeginRegistration(
			httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/passkeys/begin-registration", http.NoBody),
			user,
			&RegisterPasskeyRequest{},
		)
		require.NoError(t, err)
		assert.True(t, creation.Response.Extensions.CredProps)
	})

	t.Run("resolver error fails begin", func(t *testing.T) {
		t.Parallel()

		resolverErr := assert.AnError
		plugin, l := newTestPasskeyPlugin(t, WithRegistrationExtensionsResolver(
			func(ctx context.Context, r *http.Request) (AuthenticationExtensions, error) {
				return AuthenticationExtensions{}, resolverErr
			},
		))
		user := limen.SeedTestUser(t, l, "resolver-err@example.com")

		_, _, err := plugin.BeginRegistration(
			httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/passkeys/begin-registration", http.NoBody),
			user,
			&RegisterPasskeyRequest{},
		)
		require.ErrorIs(t, err, resolverErr)
	})
}
