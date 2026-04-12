package limen

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabaseHelper_FindOne(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	ctx := context.Background()

	seedUser(t, l, "find@test.com")

	user, err := l.core.FindOne(ctx, l.core.Schema.User, []Where{
		Eq(l.core.Schema.User.GetEmailField(), "find@test.com"),
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, "find@test.com", user.(*User).Email)
}

func TestDatabaseHelper_Create(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	ctx := context.Background()

	err := l.core.Create(ctx, l.core.Schema.User, &User{Email: "new@test.com"}, nil)
	require.NoError(t, err)

	user, err := l.core.FindOne(ctx, l.core.Schema.User, []Where{
		Eq(l.core.Schema.User.GetEmailField(), "new@test.com"),
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, "new@test.com", user.(*User).Email)
}

func TestDatabaseHelper_Create_SetsUpdatedAt(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	ctx := context.Background()
	userSchema := l.core.Schema.User

	err := l.core.Create(ctx, userSchema, &User{Email: "ts@test.com"}, nil)
	require.NoError(t, err)

	found, err := l.core.FindOne(ctx, userSchema, []Where{
		Eq(userSchema.GetEmailField(), "ts@test.com"),
	}, nil)
	require.NoError(t, err)

	raw := found.(*User).Raw()
	updatedAt, ok := raw[userSchema.GetField(SchemaUpdatedAtField)].(time.Time)
	require.True(t, ok, "updated_at should be set and typed as time.Time")
	assert.False(t, updatedAt.IsZero())
}

func TestDatabaseHelper_Create_WithAdditionalFields(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	ctx := context.Background()

	extra := map[string]any{"first_name": "John"}
	err := l.core.Create(ctx, l.core.Schema.User, &User{Email: "extra@test.com"}, extra)
	require.NoError(t, err)

	user, err := l.core.FindOne(ctx, l.core.Schema.User, []Where{
		Eq(l.core.Schema.User.GetEmailField(), "extra@test.com"),
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, "John", user.Raw()["first_name"])
}

func TestDatabaseHelper_Update(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	ctx := context.Background()
	seedUser(t, l, "update@test.com")

	err := l.core.Update(ctx, l.core.Schema.User, &User{Email: "updated@test.com"}, []Where{
		Eq(l.core.Schema.User.GetEmailField(), "update@test.com"),
	})
	require.NoError(t, err)

	user, err := l.core.FindOne(ctx, l.core.Schema.User, []Where{
		Eq(l.core.Schema.User.GetEmailField(), "updated@test.com"),
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, "updated@test.com", user.(*User).Email)
}

func TestDatabaseHelper_Update_SetsUpdatedAt(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	ctx := context.Background()
	userSchema := l.core.Schema.User

	seedUser(t, l, "touch@test.com")
	before, err := l.core.FindOne(ctx, userSchema, []Where{
		Eq(userSchema.GetEmailField(), "touch@test.com"),
	}, nil)
	require.NoError(t, err)
	beforeRaw := before.(*User).Raw()
	beforeUpdated := beforeRaw[userSchema.GetField(SchemaUpdatedAtField)].(time.Time)

	time.Sleep(2 * time.Millisecond)

	err = l.core.Update(ctx, userSchema, &User{Email: "touched@test.com"}, []Where{
		Eq(userSchema.GetEmailField(), "touch@test.com"),
	})
	require.NoError(t, err)

	after, err := l.core.FindOne(ctx, userSchema, []Where{
		Eq(userSchema.GetEmailField(), "touched@test.com"),
	}, nil)
	require.NoError(t, err)
	afterUpdated := after.(*User).Raw()[userSchema.GetField(SchemaUpdatedAtField)].(time.Time)

	assert.True(t, afterUpdated.After(beforeUpdated), "updated_at should advance on update")
}

func TestDatabaseHelper_Update_NoConditions(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	ctx := context.Background()
	err := l.core.Update(ctx, l.core.Schema.User, &User{Email: "updated@test.com"}, nil)
	assert.ErrorIs(t, err, ErrMissingConditions)
}

func TestDatabaseHelper_UpdateRaw_KeepsEmptyValues(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	ctx := context.Background()
	seedUser(t, l, "raw@test.com")

	err := l.core.UpdateRaw(ctx, l.core.Schema.User, &User{Email: ""}, []Where{
		Eq(l.core.Schema.User.GetEmailField(), "raw@test.com"),
	}, false)
	require.NoError(t, err)

	user, err := l.core.FindOne(ctx, l.core.Schema.User, []Where{
		Eq(l.core.Schema.User.GetIDField(), int64(1)),
	}, nil)
	require.NoError(t, err)
	assert.Empty(t, user.(*User).Email, "UpdateRaw with removeEmptyValues=false should keep empty values")
}

func TestDatabaseHelper_FindMany(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	ctx := context.Background()

	seedUser(t, l, "a@test.com")
	seedUser(t, l, "a@test.com")
	seedUser(t, l, "b@test.com")

	models, err := l.core.FindMany(ctx, l.core.Schema.User, []Where{
		Eq(l.core.Schema.User.GetEmailField(), "a@test.com"),
	})
	require.NoError(t, err)
	assert.Len(t, models, 2)
}

func TestDatabaseHelper_Count(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	ctx := context.Background()

	seedUser(t, l, "c1@test.com")
	seedUser(t, l, "c2@test.com")
	seedUser(t, l, "c3@test.com")

	seedUser(t, l, "c@test.com")
	seedUser(t, l, "c@test.com")
	seedUser(t, l, "c@test.com")

	count, err := l.core.Count(ctx, l.core.Schema.User, []Where{
		Eq(l.core.Schema.User.GetEmailField(), "c@test.com"),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func TestDatabaseHelper_Exists(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	ctx := context.Background()

	exists, err := l.core.Exists(ctx, l.core.Schema.User, []Where{
		Eq(l.core.Schema.User.GetEmailField(), "missing@test.com"),
	})
	require.NoError(t, err)
	assert.False(t, exists)

	seedUser(t, l, "exists@test.com")

	exists, err = l.core.Exists(ctx, l.core.Schema.User, []Where{
		Eq(l.core.Schema.User.GetEmailField(), "exists@test.com"),
	})
	require.NoError(t, err)
	assert.True(t, exists)
}
