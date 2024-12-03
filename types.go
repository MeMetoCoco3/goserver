package main

import (
	"github.com/google/uuid"
	"time"
)

const frontPath = "/app/"
const backPath = "/api/"
const adminPath = "/admin/"
const defaultExpSeconds = 3600

type User struct {
	ID               uuid.UUID   `json:"id"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
	Email            string      `json:"email"`
	HashedPassword   string      `json:"password"`
	ExpiresInSeconds int         `json:"expires_in_seconds"`
	Token            interface{} `json:"token"`
	RefreshToken     string      `json:"refresh_token"`
	IsRed            bool        `json:"is_chirpy_red"`
}

type Req struct {
	Body   string    `json:"body"`
	UserID uuid.UUID `json:"user_id"`
}
type Resp struct {
	CleanedBody string `json:"cleaned_body"`
}
type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

func stringToUUID(s string) (uuid.UUID, error) {
	u, err := uuid.Parse(s)
	return u, err
}
