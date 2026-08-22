package limen

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdditionalFieldsContext_Pick(t *testing.T) {
	t.Parallel()

	ctx := &AdditionalFieldsContext{body: map[string]any{
		"firstname": "Ada",
		"lastname":  nil,
	}}

	fields := ctx.Pick(map[string]string{
		"firstname": "first_name",
		"lastname":  "last_name",
		"nickname":  "nick_name",
	})

	assert.Equal(t, map[string]any{
		"first_name": "Ada",
		"last_name":  nil,
	}, fields, "omitted keys are skipped, keys sent as null are kept")
}

func TestAdditionalFieldsContext_WithoutRequest(t *testing.T) {
	t.Parallel()

	ctx := &AdditionalFieldsContext{}

	assert.Equal(t, "", ctx.GetHeader("X-Tenant"))
	assert.Nil(t, ctx.GetHeaders())
	assert.Equal(t, "", ctx.Method())
	assert.Equal(t, "", ctx.Path())
}
