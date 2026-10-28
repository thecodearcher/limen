package admin

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
)

var publicIDSeq atomic.Uint64

func testPublicIDConfig() limen.PublicIDConfig {
	return limen.PublicIDConfig{
		Generator: func(context.Context, limen.SchemaName) (string, error) {
			return fmt.Sprintf("usr_%03d", publicIDSeq.Add(1)), nil
		},
		Matcher: func(_ limen.SchemaName, value string) bool {
			return strings.HasPrefix(value, "usr_")
		},
	}
}

func newTestAdminPlugin(t *testing.T, opts ...ConfigOption) (*limen.Limen, *adminPlugin) {
	t.Helper()
	plugin := New(opts...)
	l, _ := limen.NewTestLimen(t, plugin)
	return l, plugin
}

func newTestAdminPluginWithPublicIDs(t *testing.T, opts ...ConfigOption) (*limen.Limen, *adminPlugin) {
	t.Helper()
	plugin := New(opts...)
	l, _ := limen.NewTestLimenWithSchema(t, limen.NewDefaultSchemaConfig(limen.WithPublicIDs(testPublicIDConfig())), plugin)
	return l, plugin
}

func seedAdmin(t *testing.T, l *limen.Limen, plugin *adminPlugin, email string) *limen.User {
	t.Helper()
	user := limen.SeedTestUser(t, l, email)
	assignRole(t, plugin, user.ID, roleNameAdmin)
	return user
}

func assignRole(t *testing.T, plugin *adminPlugin, userID any, role string) {
	t.Helper()
	err := plugin.core.Create(t.Context(), plugin.userRoleSchema, &UserRole{UserID: userID, Role: role}, nil)
	require.NoError(t, err)
}

func roleNames(roles []access.Role) []string {
	names := make([]string, 0, len(roles))
	for _, role := range roles {
		names = append(names, role.Name())
	}
	return names
}
