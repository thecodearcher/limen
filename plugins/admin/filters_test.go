package admin

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thecodearcher/limen"
)

func setUserColumns(t *testing.T, plugin *adminPlugin, userID any, data map[limen.SchemaField]any) {
	t.Helper()
	schema := plugin.core.Schema.User
	err := plugin.core.Update(t.Context(), schema, data, []limen.Where{limen.Eq(schema.GetIDField(), userID)})
	require.NoError(t, err)
}

func listEmails(t *testing.T, plugin *adminPlugin, actor *limen.User, filters map[string]string) []string {
	t.Helper()
	page, err := plugin.ListUsers(t.Context(), actor, &ListUsersQuery{Filters: filters}, nil)
	require.NoError(t, err)
	emails := make([]string, 0, len(page.Items))
	for _, user := range page.Items {
		emails = append(emails, user.Email)
	}
	return emails
}

func TestAdminPlugin_ListUsers_EmailVerifiedFilter(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAdminPlugin(t)
	actor := seedAdmin(t, l, plugin, "actor@test.com")
	verified := limen.SeedTestUser(t, l, "verified@test.com")
	limen.SeedTestUser(t, l, "unverified@test.com")
	setUserColumns(t, plugin, verified.ID, map[limen.SchemaField]any{limen.UserSchemaEmailVerifiedAtField: time.Now()})

	assert.Equal(t, []string{"verified@test.com"}, listEmails(t, plugin, actor, map[string]string{"email_verified": "true"}))
}

func TestAdminPlugin_ListUsers_BannedFilter(t *testing.T) {
	t.Parallel()

	now := time.Now()
	bans := map[string]map[limen.SchemaField]any{
		"active@test.com":    {UserSchemaBannedAtField: now, UserSchemaBanExpiresAtField: now.Add(time.Hour)},
		"permanent@test.com": {UserSchemaBannedAtField: now},
		"expired@test.com":   {UserSchemaBannedAtField: now.Add(-2 * time.Hour), UserSchemaBanExpiresAtField: now.Add(-time.Hour)},
	}

	testData := []struct {
		name  string
		value string
		want  []string
	}{
		{name: "banned", value: "true", want: []string{"active@test.com", "permanent@test.com"}},
		{name: "not banned", value: "false", want: []string{"actor@test.com", "expired@test.com", "never@test.com"}},
	}
	for _, tt := range testData {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l, plugin := newTestAdminPlugin(t)
			actor := seedAdmin(t, l, plugin, "actor@test.com")
			limen.SeedTestUser(t, l, "never@test.com")
			for email, columns := range bans {
				user := limen.SeedTestUser(t, l, email)
				setUserColumns(t, plugin, user.ID, columns)
			}

			assert.ElementsMatch(t, tt.want, listEmails(t, plugin, actor, map[string]string{"banned": tt.value}))
		})
	}
}

func TestAdminPlugin_ListUsers_CreatedAfterFilter(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAdminPlugin(t)
	actor := seedAdmin(t, l, plugin, "actor@test.com")
	recent := limen.SeedTestUser(t, l, "recent@test.com")
	created := map[any]time.Time{
		actor.ID:  time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		recent.ID: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	for id, at := range created {
		setUserColumns(t, plugin, id, map[limen.SchemaField]any{limen.SchemaCreatedAtField: at})
	}

	assert.Equal(t, []string{"recent@test.com"}, listEmails(t, plugin, actor, map[string]string{"created_after": "2025-01-01T00:00:00Z"}))
}

func TestAdminPlugin_ListUsers_CreatedDateFilters(t *testing.T) {
	t.Parallel()

	created := map[string]time.Time{
		"start@test.com": time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		"end@test.com":   time.Date(2025, 1, 1, 23, 59, 59, 500_000_000, time.UTC),
		"next@test.com":  time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
	}

	testData := []struct {
		name    string
		filters map[string]string
		want    []string
	}{
		{name: "after start of day", filters: map[string]string{"created_after": "2025-01-01"}, want: []string{"start@test.com", "end@test.com", "next@test.com"}},
		{name: "after next day", filters: map[string]string{"created_after": "2025-01-02"}, want: []string{"next@test.com"}},
		{name: "before end of day", filters: map[string]string{"created_before": "2025-01-01"}, want: []string{"actor@test.com", "start@test.com", "end@test.com"}},
	}
	for _, tt := range testData {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l, plugin := newTestAdminPlugin(t)
			actor := seedAdmin(t, l, plugin, "actor@test.com")
			setUserColumns(t, plugin, actor.ID, map[limen.SchemaField]any{limen.SchemaCreatedAtField: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)})
			for email, at := range created {
				user := limen.SeedTestUser(t, l, email)
				setUserColumns(t, plugin, user.ID, map[limen.SchemaField]any{limen.SchemaCreatedAtField: at})
			}
			assert.ElementsMatch(t, tt.want, listEmails(t, plugin, actor, tt.filters))
		})
	}
}

func TestAdminPlugin_ListUsers_AppFilter(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAdminPlugin(t, WithUserFilter("first_name", EqualFilter("first_name")))
	actor := seedAdmin(t, l, plugin, "actor@test.com")
	for email, firstName := range map[string]string{"ada@test.com": "Ada", "bob@test.com": "Bob"} {
		_, err := plugin.CreateUser(t.Context(), actor, &CreateUserRequest{Email: email, AdditionalFields: map[string]any{"first_name": firstName}})
		require.NoError(t, err)
	}

	assert.Equal(t, []string{"ada@test.com"}, listEmails(t, plugin, actor, map[string]string{"first_name": "Ada"}))
}

func TestAdminPlugin_ListUsers_InvalidFilter(t *testing.T) {
	t.Parallel()

	testData := []struct {
		name    string
		filters map[string]string
	}{
		{name: "invalid value", filters: map[string]string{"email_verified": "maybe"}},
		{name: "unknown filter", filters: map[string]string{"plan": "pro"}},
	}
	for _, tt := range testData {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l, plugin := newTestAdminPlugin(t)
			actor := seedAdmin(t, l, plugin, "actor@test.com")

			_, err := plugin.ListUsers(t.Context(), actor, &ListUsersQuery{Filters: tt.filters}, nil)
			var limenErr *limen.LimenError
			require.True(t, errors.As(err, &limenErr))
			assert.Equal(t, "invalid_filter", limenErr.Code())
		})
	}
}
