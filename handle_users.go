package main

import (
	"encoding/json"
	"fmt"
	"github.com/MeMetoCoco3/goserver/internal/auth"
	"github.com/MeMetoCoco3/goserver/internal/database"
	"net/http"
)

func (cfg *apiConfig) handlePostUser(w http.ResponseWriter, r *http.Request) {
	type Req struct {
		Email          string `json:"email"`
		HashedPassword string `json:"password"`
	}

	req := Req{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if req.HashedPassword == "" {
		http.Error(w, `"error":"No password on request."`, http.StatusNotAcceptable)
		return
	}

	hashedPassword, err := auth.HashPassword(req.HashedPassword)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), http.StatusInternalServerError)
		return
	}

	userUpdated, err := cfg.db.CreateUser(r.Context(), database.CreateUserParams{
		Email:          req.Email,
		HashedPassword: hashedPassword,
	})
	user := User{
		ID:             userUpdated.ID,
		CreatedAt:      userUpdated.CreatedAt,
		UpdatedAt:      userUpdated.UpdatedAt,
		Email:          userUpdated.Email,
		HashedPassword: userUpdated.HashedPassword,
		IsRed:          userUpdated.IsChirpyRed,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err = json.NewEncoder(w).Encode(user); err != nil {
		http.Error(w, `{"error": "Failed to encode json data."}`, http.StatusInternalServerError)
		return
	}
}

func (cfg *apiConfig) handlePutUser(w http.ResponseWriter, r *http.Request) {
	type Req struct {
		Email          string `json:"email"`
		HashedPassword string `json:"password"`
	}

	req := Req{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if req.HashedPassword == "" {
		http.Error(w, `"error":"No password on request."`, http.StatusNotAcceptable)
		return
	}

	hashedPassword, err := auth.HashPassword(req.HashedPassword)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error1":"%s"}`, err), http.StatusInternalServerError)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error2": "%s"}`, err), http.StatusUnauthorized)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error3":"%s"}`, err), http.StatusUnauthorized)
		return
	}
	fmt.Println(userID.String())
	userUpdated, err := cfg.db.GetUserWithID(r.Context(), userID)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error4":"%s"}`, err), http.StatusUnauthorized)
		return
	}
	fmt.Println("Pre Set everything")
	if userUpdated.Email != req.Email {
		cfg.db.SetNewEmail(r.Context(), database.SetNewEmailParams{
			Email: req.Email,
			ID:    userID,
		})
		userUpdated.Email = req.Email
	}
	if userUpdated.HashedPassword != hashedPassword {
		cfg.db.SetNewPassword(r.Context(), database.SetNewPasswordParams{
			HashedPassword: hashedPassword,
			ID:             userID,
		})
		userUpdated.HashedPassword = hashedPassword
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	user := User{
		ID:             userUpdated.ID,
		CreatedAt:      userUpdated.CreatedAt,
		UpdatedAt:      userUpdated.UpdatedAt,
		Email:          userUpdated.Email,
		HashedPassword: userUpdated.HashedPassword,
		IsRed:          userUpdated.IsChirpyRed,
	}

	if err = json.NewEncoder(w).Encode(user); err != nil {
		http.Error(w, `{"error": "Failed to encode json data."}`, http.StatusInternalServerError)
		return
	}
}
