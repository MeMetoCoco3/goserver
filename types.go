package main

import (
	"github.com/google/uuid"
	"time"
)

const frontPath = "/app/"
const backPath = "/api/"
const adminPath = "/admin/"

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
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
