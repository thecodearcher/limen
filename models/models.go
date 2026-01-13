package models

import (
	"time"
)

// RateLimits represents the rate_limits table
type RateLimits struct {
	Id int64 `json:"id"` // primary key
	Key string `json:"key"`
	Count int `json:"count"`
	LastRequestAt int64 `json:"last_request_at"`
}

// Sessions represents the sessions table
type Sessions struct {
	Id int64 `json:"id"` // primary key
	Token string `json:"token"`
	UserId int64 `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	LastAccess time.Time `json:"last_access"`
	Metadata *map[string]any `json:"metadata"`
}

// UserszFromPersonalUserSchema represents the usersz_from_personal_user_schema table
type UserszFromPersonalUserSchema struct {
	IdFromPersonal int64 `json:"id"` // primary key
	Email string `json:"email"`
	Password string `json:"-"`
	EmailVerified *time.Time `json:"email_verified_at"`
	FirstName string `json:"first_name"`
	LastName *string `json:"last_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Username2 string `json:"username2"`
}

// UsersUsernameFromPlugin represents the users_username_from_plugin table
// This schema is provided by plugin: username-password
type UsersUsernameFromPlugin struct {
	UsernameBrian string `json:"username_brian"`
}

// Verifications represents the verifications table
type Verifications struct {
	Id int64 `json:"id"` // primary key
	Subject string `json:"subject"`
	Value string `json:"value"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

