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

func seedBaselineUser(t *testing.T, l *Limen) {
	t.Helper()
	verifiedAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	err := l.core.Create(context.Background(), l.core.Schema.User,
		&User{Email: "base@test.com", EmailVerifiedAt: &verifiedAt},
		map[string]any{"first_name": "Test"})
	require.NoError(t, err)
}

func TestDatabaseHelper_Update_Payload(t *testing.T) {
	t.Parallel()

	verifiedAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name            string
		data            any
		wantEmail       string
		wantVerifiedNil bool
	}{
		{
			name:            "model writes set field and preserves unset columns",
			data:            &User{Email: "new@test.com"},
			wantEmail:       "new@test.com",
			wantVerifiedNil: false,
		},
		{
			name:            "all-zero model is a no-op",
			data:            &User{},
			wantEmail:       "base@test.com",
			wantVerifiedNil: false,
		},
		{
			name:            "map writes an empty string",
			data:            map[SchemaField]any{UserSchemaEmailField: ""},
			wantEmail:       "",
			wantVerifiedNil: false,
		},
		{
			name:            "map writes nil as NULL",
			data:            map[SchemaField]any{UserSchemaEmailVerifiedAtField: nil},
			wantEmail:       "base@test.com",
			wantVerifiedNil: true,
		},
		{
			name:            "map updates only the listed column",
			data:            map[SchemaField]any{UserSchemaEmailField: "changed@test.com"},
			wantEmail:       "changed@test.com",
			wantVerifiedNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l := newTestLimen(t)
			ctx := context.Background()
			userSchema := l.core.Schema.User
			seedBaselineUser(t, l)

			err := l.core.Update(ctx, userSchema, tt.data, []Where{
				Eq(userSchema.GetIDField(), int64(1)),
			})
			require.NoError(t, err)

			found, err := l.core.FindOne(ctx, userSchema, []Where{
				Eq(userSchema.GetIDField(), int64(1)),
			}, nil)
			require.NoError(t, err)

			user := found.(*User)
			assert.Equal(t, tt.wantEmail, user.Email, "email column")
			if tt.wantVerifiedNil {
				assert.Nil(t, user.EmailVerifiedAt, "email_verified_at should be NULL")
			} else {
				require.NotNil(t, user.EmailVerifiedAt, "email_verified_at should be preserved")
				assert.Equal(t, verifiedAt, *user.EmailVerifiedAt)
			}
			assert.Equal(t, "Test", user.Raw()["first_name"], "first_name must be untouched")
		})
	}
}

func TestDatabaseHelper_Update_Errors(t *testing.T) {
	t.Parallel()

	validConds := func(l *Limen) []Where {
		return []Where{Eq(l.core.Schema.User.GetIDField(), int64(1))}
	}

	tests := []struct {
		name         string
		data         any
		conditions   func(*Limen) []Where
		wantErrIs    error
		wantErrMatch string
	}{
		{
			name:       "missing conditions",
			data:       &User{Email: "x@test.com"},
			conditions: func(*Limen) []Where { return nil },
			wantErrIs:  ErrMissingConditions,
		},
		{
			name:         "unknown field in map",
			data:         map[SchemaField]any{SchemaField("not_a_real_field"): "x"},
			conditions:   validConds,
			wantErrMatch: "unknown field",
		},
		{
			name:         "unsupported data type",
			data:         "not a model or map",
			conditions:   validConds,
			wantErrMatch: "unsupported data type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l := newTestLimen(t)
			err := l.core.Update(context.Background(), l.core.Schema.User, tt.data, tt.conditions(l))
			require.Error(t, err)
			if tt.wantErrIs != nil {
				assert.ErrorIs(t, err, tt.wantErrIs)
			}
			if tt.wantErrMatch != "" {
				assert.Contains(t, err.Error(), tt.wantErrMatch)
			}
		})
	}
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
	beforeUpdated := before.(*User).Raw()[userSchema.GetField(SchemaUpdatedAtField)].(time.Time)

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

func TestDatabaseHelper_UpdateAndReturn(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	ctx := context.Background()
	userSchema := l.core.Schema.User
	seedBaselineUser(t, l)

	returned, err := l.core.UpdateAndReturn(ctx, userSchema, map[SchemaField]any{
		UserSchemaEmailField: "returned@test.com",
	}, []Where{Eq(userSchema.GetIDField(), int64(1))}, int64(1))
	require.NoError(t, err)

	assert.Equal(t, "returned@test.com", returned.(*User).Email)
}

func newTestLimenWithUserAdditionalFields(t *testing.T, fn AdditionalFieldsFunc) *Limen {
	t.Helper()

	l, _ := NewTestLimenWithSchema(t, NewDefaultSchemaConfig(
		WithSchemaUser(WithUserAdditionalFields(fn)),
	))
	return l
}

func TestDatabaseHelper_AdditionalFields_RunOnCreateAndUpdate(t *testing.T) {
	t.Parallel()

	var updateTable SchemaTableName
	l := newTestLimenWithUserAdditionalFields(t, func(ctx *AdditionalFieldsContext) (map[string]any, error) {
		if ctx.IsCreate() {
			return map[string]any{"first_name": "created"}, nil
		}
		updateTable = ctx.TableName()
		return map[string]any{"first_name": "updated"}, nil
	})

	ctx := context.Background()
	userSchema := l.core.Schema.User
	conditions := []Where{Eq(userSchema.GetEmailField(), "both@test.com")}

	require.NoError(t, l.core.Create(ctx, userSchema, &User{Email: "both@test.com"}, nil))

	created, err := l.core.FindOne(ctx, userSchema, conditions, nil)
	require.NoError(t, err)
	assert.Equal(t, "created", created.Raw()["first_name"])

	require.NoError(t, l.core.Update(ctx, userSchema, &User{Password: ptr("hashed")}, conditions))

	updated, err := l.core.FindOne(ctx, userSchema, conditions, nil)
	require.NoError(t, err)
	assert.Equal(t, "updated", updated.Raw()["first_name"])
	assert.Equal(t, userSchema.GetTableName(), updateTable, "the schema being written is exposed to the function")
}

func TestDatabaseHelper_Update_WritesAdditionalFieldsWithoutData(t *testing.T) {
	t.Parallel()

	l := newTestLimenWithUserAdditionalFields(t, func(ctx *AdditionalFieldsContext) (map[string]any, error) {
		if ctx.IsCreate() {
			return nil, nil
		}
		return map[string]any{"first_name": "from-func"}, nil
	})

	ctx := context.Background()
	userSchema := l.core.Schema.User
	conditions := []Where{Eq(userSchema.GetEmailField(), "nodata@test.com")}

	require.NoError(t, l.core.Create(ctx, userSchema, &User{Email: "nodata@test.com"}, nil))
	require.NoError(t, l.core.Update(ctx, userSchema, &User{}, conditions))

	updated, err := l.core.FindOne(ctx, userSchema, conditions, nil)
	require.NoError(t, err)
	assert.Equal(t, "from-func", updated.Raw()["first_name"], "additional fields alone are enough to write")
}

func TestDatabaseHelper_Update_AdditionalFieldsPrecedence(t *testing.T) {
	t.Parallel()

	l := newTestLimenWithUserAdditionalFields(t, func(ctx *AdditionalFieldsContext) (map[string]any, error) {
		return map[string]any{"first_name": "from-func"}, nil
	})

	ctx := context.Background()
	userSchema := l.core.Schema.User
	conditions := []Where{Eq(userSchema.GetIDField(), int64(1))}

	require.NoError(t, l.core.Create(ctx, userSchema, &User{Email: "order@test.com"}, nil))

	err := l.core.Update(ctx, userSchema, &User{Email: "from-data@test.com"}, conditions,
		WithUpdateAdditionalFields(map[string]any{
			"first_name":               "from-option",
			userSchema.GetEmailField(): "from-option@test.com",
		}))
	require.NoError(t, err)

	updated, err := l.core.FindOne(ctx, userSchema, conditions, nil)
	require.NoError(t, err)
	assert.Equal(t, "from-option", updated.Raw()["first_name"], "the option overrides the function")
	assert.Equal(t, "from-data@test.com", updated.(*User).Email, "data overrides the option")
}

func TestDatabaseHelper_Update_SkipsImmutableAdditionalFields(t *testing.T) {
	t.Parallel()

	t.Run("id column", func(t *testing.T) {
		t.Parallel()

		var idColumn string
		l := newTestLimenWithUserAdditionalFields(t, func(ctx *AdditionalFieldsContext) (map[string]any, error) {
			if ctx.IsCreate() {
				return nil, nil
			}
			return map[string]any{idColumn: int64(99), "first_name": "kept"}, nil
		})

		ctx := context.Background()
		userSchema := l.core.Schema.User
		idColumn = userSchema.GetIDField()
		require.NoError(t, l.core.Create(ctx, userSchema, &User{Email: "immutable@test.com"}, nil))

		conditions := []Where{Eq(userSchema.GetEmailField(), "immutable@test.com")}
		require.NoError(t, l.core.Update(ctx, userSchema, &User{Password: ptr("hashed")}, conditions,
			WithUpdateAdditionalFields(map[string]any{idColumn: int64(42)})))

		updated, err := l.core.FindOne(ctx, userSchema, conditions, nil)
		require.NoError(t, err)
		assert.Equal(t, int64(1), updated.(*User).ID, "the id is not rewritten from either source")
		assert.Equal(t, "kept", updated.Raw()["first_name"], "the remaining additional fields still apply")
	})

	t.Run("created at column", func(t *testing.T) {
		t.Parallel()

		createdAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		var createdAtColumn string
		l := newTestLimenWithUserAdditionalFields(t, func(ctx *AdditionalFieldsContext) (map[string]any, error) {
			if ctx.IsCreate() {
				return map[string]any{createdAtColumn: createdAt}, nil
			}
			return map[string]any{createdAtColumn: time.Now()}, nil
		})

		ctx := context.Background()
		userSchema := l.core.Schema.User
		createdAtColumn = userSchema.GetField(SchemaCreatedAtField)
		conditions := []Where{Eq(userSchema.GetEmailField(), "stamp@test.com")}

		require.NoError(t, l.core.Create(ctx, userSchema, &User{Email: "stamp@test.com"}, nil))
		require.NoError(t, l.core.Update(ctx, userSchema, &User{Password: ptr("hashed")}, conditions,
			WithUpdateAdditionalFields(map[string]any{createdAtColumn: time.Now()})))

		updated, err := l.core.FindOne(ctx, userSchema, conditions, nil)
		require.NoError(t, err)
		assert.Equal(t, createdAt, updated.Raw()[createdAtColumn],
			"the insert timestamp is not replayed from either source")
	})

	t.Run("public ID column", func(t *testing.T) {
		t.Parallel()

		var publicIDColumn string
		l, userSchema := newPublicIDFixture(t, usersOnlyPublicIDConfig(noopPublicIDGenerator()),
			WithSchemaUser(WithUserAdditionalFields(func(ctx *AdditionalFieldsContext) (map[string]any, error) {
				if ctx.IsCreate() {
					return map[string]any{publicIDColumn: "original"}, nil
				}
				return map[string]any{publicIDColumn: "rewritten"}, nil
			})))
		publicIDColumn = userSchema.GetField(SchemaPublicIDField)

		ctx := context.Background()
		conditions := []Where{Eq(l.core.Schema.User.GetEmailField(), "public@test.com")}
		require.NoError(t, l.core.Create(ctx, userSchema, &User{Email: "public@test.com"}, nil))

		require.NoError(t, l.core.Update(ctx, userSchema, &User{Password: ptr("hashed")}, conditions))

		updated, err := l.core.FindOne(ctx, userSchema, conditions, nil)
		require.NoError(t, err)
		assert.Equal(t, "original", updated.Raw()[publicIDColumn],
			"the public ID supplied on insert is not rewritten on update")
	})
}

func seedRateLimitRow(t *testing.T, l *Limen, key string, count int32, lastRequestAt int64) {
	t.Helper()

	schema := l.core.Schema.RateLimit
	_, err := l.core.db.Create(t.Context(), schema.GetTableName(), map[string]any{
		schema.GetKeyField():           key,
		schema.GetCountField():         count,
		schema.GetLastRequestAtField(): lastRequestAt,
	})
	require.NoError(t, err)
}

func TestDatabaseHelper_Update_MixesArithmeticAndAssignments(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	schema := l.core.Schema.RateLimit
	seedRateLimitRow(t, l, "mixed-update", 4, 100)

	err := l.core.Update(t.Context(), schema, map[SchemaField]any{
		RateLimitSchemaCountField:         IncrementBy(3),
		RateLimitSchemaLastRequestAtField: int64(200),
	}, []Where{
		Eq(schema.GetKeyField(), "mixed-update"),
	})
	require.NoError(t, err)

	found, err := l.core.FindOne(t.Context(), schema, []Where{
		Eq(schema.GetKeyField(), "mixed-update"),
	}, nil)
	require.NoError(t, err)

	rateLimit := found.(*RateLimit)
	assert.Equal(t, 7, rateLimit.Count)
	assert.Equal(t, int64(200), rateLimit.LastRequestAt)
}

func TestDatabaseHelper_Update_RejectsInvalidArithmeticAmount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		update ArithmeticUpdate
	}{
		{name: "zero increment", update: IncrementBy(0)},
		{name: "negative increment", update: IncrementBy(-1)},
		{name: "zero decrement", update: DecrementBy(0)},
		{name: "negative decrement", update: DecrementBy(-1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l := newTestLimen(t)
			schema := l.core.Schema.RateLimit
			err := l.core.Update(t.Context(), schema, map[SchemaField]any{
				RateLimitSchemaCountField: tt.update,
			}, []Where{
				Eq(schema.GetKeyField(), "missing"),
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "amount must be greater than zero")
		})
	}
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

func TestDatabaseHelper_CreateAndReturn(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	ctx := context.Background()
	userSchema := l.core.Schema.User

	created, err := l.core.CreateAndReturn(ctx, userSchema, &User{Email: "ret@test.com"}, nil, UserSchemaEmailField)
	require.NoError(t, err)

	user := created.(*User)
	assert.Equal(t, "ret@test.com", user.Email)
	assert.NotNil(t, user.ID)

	raw := user.Raw()
	updatedAt, ok := raw[userSchema.GetField(SchemaUpdatedAtField)].(time.Time)
	require.True(t, ok, "updated_at should be set on the returned model")
	assert.False(t, updatedAt.IsZero())
}
