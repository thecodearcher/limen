package passkey

import (
	"context"
	"fmt"
	"strings"
)

func (p *passkeyPlugin) validateHandleConfig() error {
	if p.config.allowInternalUserIDAsHandle {
		return nil
	}
	if p.core.PublicIDColumn(p.core.Schema.User) == "" {
		return ErrHandleModeRequired
	}
	return nil
}

func (p *passkeyPlugin) reserveRegistrationHandle(ctx context.Context, intent *RegistrationIntent) (string, error) {
	if id := strings.TrimSpace(intent.ID); id != "" {
		return id, nil
	}

	if p.config.allowInternalUserIDAsHandle {
		if p.core.Schema.IDGenerator == nil {
			return "", ErrIDGeneratorRequired
		}
		id, err := p.core.Schema.IDGenerator.Generate(ctx)
		if err != nil {
			return "", fmt.Errorf("passkey: generate id: %w", err)
		}
		return opaqueIDString(id)
	}

	return p.core.GeneratePublicID(ctx, p.core.Schema.User)
}

func opaqueIDString(id any) (string, error) {
	switch v := id.(type) {
	case string:
		if v == "" {
			return "", ErrHandleMismatch
		}
		return v, nil
	case []byte:
		if len(v) == 0 {
			return "", ErrHandleMismatch
		}
		return string(v), nil
	default:
		return "", fmt.Errorf("passkey: opaque user id must be a string, got %T", id)
	}
}
