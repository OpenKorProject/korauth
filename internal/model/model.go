package model

import (
	"time"

	"github.com/google/uuid"
)

type Tenant struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type User struct {
	ID                  uuid.UUID  `json:"id"`
	TenantID            uuid.UUID  `json:"tenant_id"`
	Username            string     `json:"username"`
	FirstName           *string    `json:"first_name"`
	LastName            *string    `json:"last_name"`
	Email               *string    `json:"email"`
	PasswordHash        string     `json:"-"`
	Roles               []string   `json:"roles"`
	ForcePasswordChange bool       `json:"force_password_change,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	DeletedAt           *time.Time `json:"-"`
}

type Role struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}
