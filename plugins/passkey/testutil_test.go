package passkey

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/thecodearcher/limen"
)

var handleSeq atomic.Uint64

func nextHandle() string {
	return fmt.Sprintf("handle_%03d", handleSeq.Add(1))
}

func nextPublicID() string {
	return fmt.Sprintf("pid_%03d", handleSeq.Add(1))
}

func testPasskeyPublicIDConfig() limen.PublicIDConfig {
	return limen.PublicIDConfig{
		Generator: func(context.Context, limen.SchemaName) (string, error) {
			return nextPublicID(), nil
		},
		Matcher: func(_ limen.SchemaName, value string) bool {
			return strings.HasPrefix(value, "pid_")
		},
	}
}

func newTestPasskeyPlugin(t *testing.T, opts ...ConfigOption) (*passkeyPlugin, *limen.Limen) {
	t.Helper()

	plugin := New(opts...)
	schema := limen.NewDefaultSchemaConfig(limen.WithPublicIDs(testPasskeyPublicIDConfig()))
	l, _ := limen.NewTestLimenWithSchema(t, schema, plugin)
	return plugin, l
}

func newTestPublicRegistrationPlugin(
	t *testing.T,
	createUser CreateRegistrationUserFunc,
	opts ...ConfigOption,
) (*passkeyPlugin, *limen.Limen) {
	t.Helper()

	base := []ConfigOption{
		WithRequireSession(false),
		WithPrepareRegistration(func(ctx context.Context, r *http.Request, registrationContext string) (*RegistrationIntent, error) {
			return &RegistrationIntent{Email: registrationContext}, nil
		}),
		WithCreateRegistrationUser(createUser),
	}
	return newTestPasskeyPlugin(t, append(base, opts...)...)
}

func seedTestPasskey(t *testing.T, plugin *passkeyPlugin, userID any, credentialID string) *Passkey {
	t.Helper()

	passkey, err := plugin.core.CreateAndReturn(t.Context(), plugin.passkeySchema, &Passkey{
		UserID:       userID,
		CredentialID: credentialID,
		PublicKey:    "dGVzdC1wdWJsaWMta2V5",
		AAGUID:       "00000000-0000-0000-0000-000000000000",
	}, nil, PasskeySchemaCredentialIDField)
	require.NoError(t, err)
	return passkey.(*Passkey)
}

func seedFixturePasskey(t *testing.T, plugin *passkeyPlugin, userID any) *Passkey {
	t.Helper()

	cred := loadFixtureCredential(t)
	passkey, err := plugin.core.CreateAndReturn(t.Context(), plugin.passkeySchema, &Passkey{
		UserID:         userID,
		CredentialID:   cred.CredentialID,
		PublicKey:      cred.PublicKey,
		SignCount:      cred.SignCount,
		AAGUID:         cred.AAGUID,
		BackupEligible: cred.BackupEligible,
		BackupState:    cred.BackupState,
	}, nil, PasskeySchemaCredentialIDField)
	require.NoError(t, err)
	return passkey.(*Passkey)
}

type fixtureCredential struct {
	UserHandle     string `json:"user_handle"`
	CredentialID   string `json:"credential_id"`
	PublicKey      string `json:"public_key"`
	SignCount      uint32 `json:"sign_count"`
	AAGUID         string `json:"aaguid"`
	BackupEligible bool   `json:"backup_eligible"`
	BackupState    bool   `json:"backup_state"`
}

type ceremonyFixture struct {
	Challenge PasskeyChallenge `json:"challenge"`
	Response  json.RawMessage  `json:"response"`
}

type passkeyFixtures struct {
	Credential   fixtureCredential `json:"credential"`
	Registration ceremonyFixture   `json:"registration"`
	Assertion    ceremonyFixture   `json:"assertion"`
}

var (
	fixturesOnce sync.Once
	fixtures     passkeyFixtures
	fixturesErr  error
)

func loadFixtures(t *testing.T) *passkeyFixtures {
	t.Helper()
	fixturesOnce.Do(func() {
		raw, err := os.ReadFile(filepath.Join("testdata", "fixtures.json"))
		if err != nil {
			fixturesErr = err
			return
		}
		fixturesErr = json.Unmarshal(raw, &fixtures)
	})
	require.NoError(t, fixturesErr)
	return &fixtures
}

func loadFixtureCredential(t *testing.T) fixtureCredential {
	t.Helper()
	return loadFixtures(t).Credential
}

func loadRegistrationChallenge(t *testing.T) *PasskeyChallenge {
	t.Helper()
	challenge := loadFixtures(t).Registration.Challenge
	return &challenge
}

func loadRegistrationResponse(t *testing.T) []byte {
	t.Helper()
	return append([]byte(nil), loadFixtures(t).Registration.Response...)
}

func loadAssertionChallenge(t *testing.T) *PasskeyChallenge {
	t.Helper()
	challenge := loadFixtures(t).Assertion.Challenge
	return &challenge
}

func loadAssertionResponse(t *testing.T) []byte {
	t.Helper()
	return append([]byte(nil), loadFixtures(t).Assertion.Response...)
}

func requestWithChallenge(
	t *testing.T,
	plugin *passkeyPlugin,
	method, path string,
	body []byte,
	challenge *PasskeyChallenge,
) *http.Request {
	t.Helper()

	recorder := httptest.NewRecorder()
	require.NoError(t, plugin.setChallengeCookie(recorder, challenge))

	req := httptest.NewRequestWithContext(t.Context(), method, path, bytes.NewReader(body))
	for _, cookie := range recorder.Result().Cookies() {
		req.AddCookie(cookie)
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

type stringIDGenerator struct {
	next atomic.Uint64
}

func (g *stringIDGenerator) Generate(context.Context) (any, error) {
	return fmt.Sprintf("uid_%03d", g.next.Add(1)), nil
}

func (g *stringIDGenerator) GetColumnType() limen.ColumnType {
	return limen.ColumnTypeString
}
