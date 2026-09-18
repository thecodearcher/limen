package passkey

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatAAGUIDGroupsBytesAsAUUID(t *testing.T) {
	t.Parallel()

	raw, _ := hex.DecodeString("adce000235bcc60a648b0b25f1f05503")

	assert.Equal(t, "adce0002-35bc-c60a-648b-0b25f1f05503", formatAAGUID(raw))
}
