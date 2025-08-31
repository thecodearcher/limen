package aegis

import (
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid configuration",
			config: &Config{
				Database: NewMockDatabaseAdapter(gomock.NewController(t)),
			},
			wantErr: false,
		},
		{
			name:    "nil configuration",
			config:  nil,
			wantErr: true,
			errMsg:  "missing configuration",
		},
		{
			name: "configuration with nil adapter",
			config: &Config{
				Database: nil,
			},
			wantErr: true,
			errMsg:  "invalid configuration: database adapter is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aegis, err := New(tt.config)

			if tt.wantErr {
				assert.Error(t, err, "New() expected error but got none")
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg, "Error message should contain expected text")
				}
				assert.Nil(t, aegis, "Expected nil Aegis instance when error occurs")
			} else {
				assert.NoError(t, err, "New() unexpected error")
				assert.NotNil(t, aegis, "Expected Aegis instance but got nil")
				// Verify that the aegis instance was created successfully
				assert.NotNil(t, aegis)
			}
		})
	}
}
