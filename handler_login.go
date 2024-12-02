package main

import (
	"encoding/json"
	"fmt"
	"github.com/MeMetoCoco3/goserver/internal/auth"
	"github.com/MeMetoCoco3/goserver/internal/database"
	"net/http"
	"time"
)

func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {

	u := User{}
	fmt.Println("LOGIN!")
	err := json.NewDecoder(r.Body).Decode(&u)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), http.StatusInternalServerError)
		return
	}

	if u.ExpiresInSeconds == 0 || u.ExpiresInSeconds > defaultExpSeconds {
		u.ExpiresInSeconds = defaultExpSeconds
	}

	user, err := cfg.db.GetUser(r.Context(), u.Email)
	if err != nil {
		http.Error(w, `{"error":"Incorrect email"}`, http.StatusUnauthorized)
		return
	}

	err = auth.CheckPasswordHash(user.HashedPassword, u.HashedPassword)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"Incorrect email or password, %s"}`, err), http.StatusUnauthorized)
		return
	}

	token, err := auth.MakeJWT(user.ID, cfg.jwtSecret, time.Duration(u.ExpiresInSeconds))
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), http.StatusInternalServerError)
		return
	}
	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	u = User{
		ID:           user.ID,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		Email:        user.Email,
		Token:        token,
		RefreshToken: refreshToken,
	}
	_, err = cfg.db.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token:     refreshToken,
		UserID:    user.ID,
		ExpiresAt: time.Now().UTC().Add(time.Hour * 24 * 60),
	})
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), http.StatusInternalServerError)
		return
	}

	if err = json.NewEncoder(w).Encode(&u); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), http.StatusInternalServerError)
		return
	}
}
