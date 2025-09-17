package session

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/thecodearcher/aegis"
	"github.com/thecodearcher/aegis/internal/database"
	"github.com/thecodearcher/aegis/schemas"
)

type JWTStrategy struct {
	jwtHandler *aegis.JwtHandler
	config     *aegis.SessionConfig
	dbAction   *database.DatabaseActionHelper
}

func NewJWTStrategy(core *aegis.AegisCore, config *aegis.SessionConfig) *JWTStrategy {
	return &JWTStrategy{
		jwtHandler: core.JWT,
		config:     config,
		dbAction:   database.NewCommonDatabaseActionsHelper(core),
	}
}

func (s *JWTStrategy) GetName() string {
	return string(aegis.SessionStrategyJWT)
}

func (s *JWTStrategy) IsStateful() bool {
	return false
}

func (s *JWTStrategy) Create(ctx context.Context, user *schemas.User) (*aegis.SessionResult, error) {
	sessionID, err := GenerateCryptoSecureRandomString()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	token, refreshToken, err := s.jwtHandler.GenerateAccessToken(sessionID, user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate JWT token: %w", err)
	}

	return &aegis.SessionResult{
		Token:        token,
		RefreshToken: refreshToken,
	}, nil
}

func (s *JWTStrategy) Validate(ctx context.Context, request *http.Request) (*aegis.SessionValidateResult, error) {
	token, err := s.extractJWTToken(request)
	if err != nil {
		return nil, aegis.ErrSessionNotFound
	}

	claims, err := s.jwtHandler.VerifyToken(token)
	if err != nil {
		if err == aegis.ErrTokenExpired {
			return nil, aegis.ErrSessionExpired
		}
		return nil, aegis.ErrSessionInvalid
	}
	fmt.Println(claims)

	return &aegis.SessionValidateResult{
		UserID:   claims["sub"],
		Metadata: claims,
	}, nil
}

func (s *JWTStrategy) Refresh(ctx context.Context, request *http.Request) (*aegis.SessionRefreshResult, error) {
	refreshToken, err := s.extractRefreshToken(request)
	if err != nil {
		return nil, fmt.Errorf("failed to extract refresh token: %w", err)
	}

	claims, err := s.jwtHandler.VerifyToken(refreshToken)
	if err != nil {
		if err == aegis.ErrTokenExpired {
			return nil, aegis.ErrSessionExpired
		}
		return nil, aegis.ErrSessionInvalid
	}

	userID, ok := claims["sub"].(string)
	if !ok || userID == "" {
		return nil, aegis.ErrSessionInvalid
	}

	sessionID, ok := claims["jti"].(string)
	if !ok || sessionID == "" {
		return nil, aegis.ErrSessionInvalid
	}

	sessionID = strings.TrimSuffix(sessionID, "_refresh")

	user, err := s.findUserWithSubject(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user data: %w", err)
	}

	newSessionID, err := GenerateCryptoSecureRandomString()
	if err != nil {
		return nil, fmt.Errorf("failed to generate new session ID: %w", err)
	}

	token, newRefreshToken, err := s.jwtHandler.GenerateAccessToken(newSessionID, user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new JWT tokens: %w", err)
	}

	return &aegis.SessionRefreshResult{
		Token:          token,
		StaleSessionID: sessionID,
		UserID:         userID,
		ShouldStore:    false,
		RefreshToken:   newRefreshToken,
	}, nil
}

func (s *JWTStrategy) extractJWTToken(request *http.Request) (string, error) {
	if cookie, err := request.Cookie(s.config.CookieOptions.Name); err == nil {
		token := strings.TrimSpace(cookie.Value)
		if token != "" {
			return token, nil
		}
	}

	authHeader := request.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			token := strings.TrimSpace(parts[1])
			if token != "" {
				return token, nil
			}
		}
	}

	return "", fmt.Errorf("no JWT token found in request")
}

func (s *JWTStrategy) extractRefreshToken(request *http.Request) (string, error) {
	if cookie, err := request.Cookie(s.config.CookieOptions.Name + "_refresh"); err == nil {
		token := strings.TrimSpace(cookie.Value)
		if token != "" {
			return token, nil
		}
	}

	if request.Method == "POST" {
		if err := request.ParseForm(); err == nil {
			refreshToken := request.Form.Get("refresh_token")
			if refreshToken != "" {
				return refreshToken, nil
			}
		}
	}

	return "", fmt.Errorf("no refresh token found in request")
}

func (s *JWTStrategy) findUserWithSubject(ctx context.Context, userID string) (*schemas.User, error) {
	if userFn := s.jwtHandler.CustomUserFromSubjectFn(); userFn != nil {
		user, err := userFn(userID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch user data: %w", err)
		}
		return user, nil
	}

	user, err := s.dbAction.FindUserByEmail(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user data: %w", err)
	}
	return user, nil
}
