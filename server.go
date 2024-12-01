package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/MeMetoCoco3/goserver/internal/auth"
	"github.com/MeMetoCoco3/goserver/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	who            string
	jwtSecret      string
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	devEnv := os.Getenv("PLATFORM")
	jwtS := os.Getenv("JWT_SECRET")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Println(err)
		return
	}

	dbQueries := database.New(db)

	cfg := apiConfig{
		fileserverHits: atomic.Int32{},
		db:             dbQueries,
		who:            devEnv,
		jwtSecret:      jwtS,
	}
	handler := http.NewServeMux()

	fileServer := http.FileServer(http.Dir("."))

	//----------/app/---------------
	handler.Handle(frontPath, http.StripPrefix(frontPath, middlewareLog(cfg.middlewareMetricsInc(fileServer))))

	//----------/admin/metrics/---------------
	handler.Handle(fmt.Sprintf("GET %smetrics", adminPath), middlewareLog(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		count := cfg.fileserverHits.Load()
		w.Write([]byte(fmt.Sprintf(`
		<html>
		  <body>
			<h1>Welcome, Chirpy Admin</h1>
			<p>Chirpy has been visited %d times!</p>
		  </body>
		</html>`,
			count)))
	}))

	//----------/admin/reset/---------------
	handler.Handle(fmt.Sprintf("POST %sreset", adminPath), middlewareLog(func(w http.ResponseWriter, r *http.Request) {
		_ = cfg.fileserverHits.Swap(0)

		if cfg.who != "dev" {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("Reset is only allowed in dev environment."))
			return
		}

		cfg.db.DeleteUsers(r.Context())
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		count := cfg.fileserverHits.Load()
		w.Write([]byte(fmt.Sprintf("Hits: %d", count)))
	}))

	//----------/api/healthz/---------------
	handler.Handle(fmt.Sprintf("GET %shealthz", backPath), middlewareLog(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("OK"))
	}))

	//--------------/api/users/-------------------------
	handler.Handle(fmt.Sprintf("POST %susers", backPath), middlewareLog(func(w http.ResponseWriter, r *http.Request) {
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
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err = json.NewEncoder(w).Encode(user); err != nil {
			http.Error(w, `{"error": "Failed to encode json data."}`, http.StatusInternalServerError)
			return
		}
	}))

	// ------------/api/chirps/---------------------
	handler.Handle(fmt.Sprintf("POST %schirps", backPath), middlewareLog(func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.GetBearerToken(r.Header)
		fmt.Printf("%s\n", token)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), http.StatusUnauthorized)
			return
		}

		uuID, err := auth.ValidateJWT(token, cfg.jwtSecret)
		if err != nil {
			fmt.Printf("Error in validation JWT\n Token: %s", token)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), http.StatusUnauthorized)
			return
		}

		req := Req{}
		err = json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, `{"error": "Failed to decode body."}`, http.StatusInternalServerError)
			return
		}

		if req.Body, err = validateChirp(req.Body); err != nil {
			http.Error(w, fmt.Sprintf(`{"error3":"%s"}`, err), http.StatusNotAcceptable)
			return
		}
		newChirp, err := cfg.db.CreateChirp(r.Context(), database.CreateChirpParams{
			Body:   req.Body,
			UserID: uuID,
		})
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), http.StatusInternalServerError)
			return
		}
		chirp := Chirp{
			ID:        newChirp.ID,
			CreatedAt: newChirp.CreatedAt,
			UpdatedAt: newChirp.UpdatedAt,
			Body:      newChirp.Body,
			UserID:    newChirp.UserID,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err = json.NewEncoder(w).Encode(chirp); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), http.StatusInternalServerError)
			return
		}
	}))

	handler.Handle(fmt.Sprintf("GET %schirps", backPath), middlewareLog(func(w http.ResponseWriter, r *http.Request) {

		chirps := make([]Chirp, 0)
		newChirps, err := cfg.db.GetChirps(r.Context())
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), http.StatusInternalServerError)
			return
		}
		for _, chirp := range newChirps {
			chirps = append(chirps, Chirp{
				ID:        chirp.ID,
				CreatedAt: chirp.CreatedAt,
				UpdatedAt: chirp.UpdatedAt,
				Body:      chirp.Body,
				UserID:    chirp.UserID,
			})
		}
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		if err = json.NewEncoder(w).Encode(chirps); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), http.StatusInternalServerError)
			return
		}

	}))

	handler.Handle(fmt.Sprintf("GET %schirps/{id}", backPath), middlewareLog(func(w http.ResponseWriter, r *http.Request) {
		u, err := stringToUUID(r.PathValue("id"))

		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s}"`, err), http.StatusInternalServerError)
		}

		newChirp, err := cfg.db.GetChirp(r.Context(), u)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), http.StatusNotFound)
		}

		chirp := Chirp{
			ID:        newChirp.ID,
			CreatedAt: newChirp.CreatedAt,
			UpdatedAt: newChirp.UpdatedAt,
			Body:      newChirp.Body,
			UserID:    newChirp.UserID,
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		if err = json.NewEncoder(w).Encode(chirp); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), http.StatusInternalServerError)
			return
		}
	}))

	//-------------/api/revoke/-----------------
	handler.Handle(fmt.Sprintf("%srevoke", backPath), middlewareLog(func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), http.StatusInternalServerError)
			return
		}

		_, err = cfg.db.DeleteRefreshToken(r.Context(), string(token))
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
		return
	}))

	//-------------/api/refresh/-----------------
	handler.Handle(fmt.Sprintf("%srefresh", backPath), middlewareLog(func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), http.StatusInternalServerError)
			return
		}

		tokenData, err := cfg.db.GetRefreshToken(r.Context(), string(token))
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), http.StatusInternalServerError)
			return
		}
		user, err := cfg.db.GetUserWithToken(r.Context(), tokenData.Token)
		if err != nil {
			http.Error(w, `{"error":"Couldn't get user for refresh token"}`, http.StatusUnauthorized)
			return
		}
		accessToken, err := auth.MakeJWT(
			user.ID,
			cfg.jwtSecret,
			time.Hour,
		)
		if err != nil {
			http.Error(w, `{"error":"Couldnt validate token."}`, http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fmt.Sprintf(`{"token":"%s"}`, accessToken)))
		return
	}))
	//-------------/api/login/-----------------
	handler.Handle(fmt.Sprintf("%slogin", backPath), middlewareLog(func(w http.ResponseWriter, r *http.Request) {
		u := User{}
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
	}))

	server := http.Server{
		Handler: handler,
		Addr:    ":8080",
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server Failed to start: %v", err)
	}
}

func validateChirp(chirp string) (string, error) {
	if len(chirp) > 140 {
		return "", fmt.Errorf("Too long chirp")
	}

	badWords := map[string]struct{}{
		"kerfuffle": {},
		"sharbert":  {},
		"fornax":    {},
	}
	words := strings.Fields(chirp)

	for i, word := range words {
		lowerCaseWord := strings.ToLower(word)
		if _, ok := badWords[lowerCaseWord]; ok {
			words[i] = "****"
		}
	}
	return strings.Join(words, " "), nil

}
