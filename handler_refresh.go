package main

import (
	"fmt"
	"github.com/MeMetoCoco3/goserver/internal/auth"
	"net/http"
	"time"
)

func (cfg *apiConfig) handlerRefresh(w http.ResponseWriter, r *http.Request) {
	fmt.Println(" ENTER REFRESH")
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), http.StatusInternalServerError)
		return
	}
	fmt.Printf("We got the token : %v \n", token)
	tokenData, err := cfg.db.GetRefreshToken(r.Context(), string(token))
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), http.StatusInternalServerError)
		return
	}
	fmt.Printf("We validated the token\n")
	user, err := cfg.db.GetUserWithToken(r.Context(), tokenData.Token)
	if err != nil {
		http.Error(w, `{"error":"Couldn't get user for refresh token"}`, http.StatusUnauthorized)
		return
	}
	fmt.Println("We got the user")
	accessToken, err := auth.MakeJWT(
		user.ID,
		cfg.jwtSecret,
		time.Hour,
	)

	if err != nil {
		http.Error(w, `{"error":"Couldnt validate token."}`, http.StatusUnauthorized)
		return
	}
	fmt.Println("We made the token!")
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(fmt.Sprintf(`{"token":"%s"}`, accessToken)))
	return
}
