package access

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAccessControlStatementsReturnsCopy(t *testing.T) {
	t.Parallel()

	ac := New(Statements{"project": {"read", "create"}})

	got := ac.Statements()
	assert.Equal(t, Statements{"project": {"create", "read"}}, got)

	got["project"] = nil
	got["injected"] = []string{"boom"}

	assert.Equal(t, Statements{"project": {"create", "read"}}, ac.Statements())
}

func TestMergeStatements(t *testing.T) {
	t.Parallel()

	base := Statements{
		"organization": {"create", "read", "update", "delete"},
		"member":       {"read", "update", "delete"},
	}
	app := Statements{
		"organization": {"create", "read", "archive"},
		"project":      {"create", "read"},
	}

	merged := MergeStatements(base, app)

	// Later set wins wholesale per resource.
	assert.Equal(t, []string{"create", "read", "archive"}, merged["organization"])
	assert.Equal(t, []string{"read", "update", "delete"}, merged["member"])
	assert.Equal(t, []string{"create", "read"}, merged["project"])

	// The result is a copy in both directions.
	merged["member"][0] = "mutated"
	assert.Equal(t, []string{"read", "update", "delete"}, base["member"])
}

func TestMergePermissions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		sets []Permissions
		want Permissions
	}{
		{
			name: "union of action lists",
			sets: []Permissions{
				{"organization": {"read"}},
				{"organization": {"update"}, "project": {"create"}},
			},
			want: Permissions{"organization": {"read", "update"}, "project": {"create"}},
		},
		{
			name: "union never drops grants",
			sets: []Permissions{
				{"organization": {"create", "read", "update", "delete"}},
				{"organization": {"archive"}},
			},
			want: Permissions{"organization": {"create", "read", "update", "delete", "archive"}},
		},
		{
			name: "wildcard absorbs other actions",
			sets: []Permissions{
				{"organization": {"read", "update"}},
				{"organization": {"*"}},
				{"organization": {"delete"}},
			},
			want: Permissions{"organization": {"*"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, MergePermissions(tt.sets...))
		})
	}
}

func TestMergePermissionsReturnsCopy(t *testing.T) {
	t.Parallel()

	base := Permissions{"project": {"read"}}
	merged := MergePermissions(base)

	merged["project"][0] = "mutated"
	merged["injected"] = []string{"boom"}

	assert.Equal(t, Permissions{"project": {"read"}}, base)
}

func TestHasRequiredPermissions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		subject  Permissions
		required Permissions
		want     bool
	}{
		{
			name:     "exact match",
			subject:  Permissions{"file": {"read"}},
			required: Permissions{"file": {"read"}},
			want:     true,
		},
		{
			name:     "wildcard covers any action",
			subject:  Permissions{"file": {"*"}},
			required: Permissions{"file": {"read", "write"}},
			want:     true,
		},
		{
			name:     "missing action",
			subject:  Permissions{"file": {"read"}},
			required: Permissions{"file": {"write"}},
			want:     false,
		},
		{
			name:     "missing resource",
			subject:  Permissions{"file": {"read"}},
			required: Permissions{"db": {"read"}},
			want:     false,
		},
		{
			name:     "empty required always passes",
			subject:  Permissions{},
			required: Permissions{},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, HasRequiredPermissions(tt.subject, tt.required))
		})
	}
}

func TestClampPermissions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		requested Permissions
		grantable Permissions
		want      Permissions
	}{
		{
			name:      "clamps to grantable subset",
			requested: Permissions{"file": {"read", "write"}},
			grantable: Permissions{"file": {"read"}},
			want:      Permissions{"file": {"read"}},
		},
		{
			name:      "wildcard grantable passes requested through",
			requested: Permissions{"file": {"read", "write"}},
			grantable: Permissions{"file": {"*"}},
			want:      Permissions{"file": {"read", "write"}},
		},
		{
			name:      "ungrantable resource dropped",
			requested: Permissions{"db": {"read"}},
			grantable: Permissions{"file": {"read"}},
			want:      Permissions{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, ClampPermissions(tt.requested, tt.grantable))
		})
	}
}

func TestP(t *testing.T) {
	t.Parallel()

	t.Run("parses specs", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name  string
			specs []string
			want  Permissions
		}{
			{
				name:  "single action",
				specs: []string{"invitation:cancel"},
				want:  Permissions{"invitation": {"cancel"}},
			},
			{
				name:  "comma-separated actions",
				specs: []string{"invitation:create,read"},
				want:  Permissions{"invitation": {"create", "read"}},
			},
			{
				name:  "multiple resources",
				specs: []string{"organization:read", "member:read"},
				want:  Permissions{"organization": {"read"}, "member": {"read"}},
			},
			{
				name:  "trims whitespace",
				specs: []string{" invitation : create , read "},
				want:  Permissions{"invitation": {"create", "read"}},
			},
			{
				name:  "merges duplicate resources",
				specs: []string{"invitation:create", "invitation:read"},
				want:  Permissions{"invitation": {"create", "read"}},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.want, P(tt.specs...))
			})
		}
	})

	t.Run("panics on invalid specs", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name  string
			specs []string
		}{
			{"no specs", nil},
			{"empty spec", []string{""}},
			{"missing colon", []string{"invitation"}},
			{"missing resource", []string{":read"}},
			{"missing actions", []string{"invitation:"}},
			{"empty action", []string{"invitation:create,"}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Panics(t, func() { P(tt.specs...) })
			})
		}
	})
}

func TestExpandPermissions(t *testing.T) {
	t.Parallel()

	ac := New(Statements{"project": {"read", "create"}, "member": {"read", "delete"}})

	got := ac.ExpandPermissions(Permissions{"project": {Wildcard}, "member": {"read"}})
	assert.Equal(t, Permissions{"project": {"create", "read"}, "member": {"read"}}, got)

	got["project"][0] = "mutated"
	assert.Equal(t, []string{"create", "read"}, ac.Statements()["project"])
}

func TestPermissionsSpecs(t *testing.T) {
	t.Parallel()

	perms := Permissions{"project": {"create", "read"}, "member": {"delete"}}

	specs := perms.Specs()
	assert.Equal(t, []string{"member:delete", "project:create", "project:read"}, specs)
	assert.Equal(t, perms, P(specs...))
}
