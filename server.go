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

	"github.com/MeMetoCoco3/goserver/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	who            string
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
		user := User{}
		err := json.NewDecoder(r.Body).Decode(&user)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		userUpdated, err := cfg.db.CreateUser(r.Context(), user.Email)
		user = User{
			ID:        userUpdated.ID,
			CreatedAt: userUpdated.CreatedAt,
			UpdatedAt: userUpdated.UpdatedAt,
			Email:     userUpdated.Email,
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
		req := Req{}
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, `{"error": "Failed to decode body."}`, http.StatusInternalServerError)
			return
		}

		if req.Body, err = validateChirp(req.Body); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), http.StatusNotAcceptable)
			return
		}

		newChirp, err := cfg.db.CreateChirp(r.Context(), database.CreateChirpParams{
			Body:   req.Body,
			UserID: req.UserID,
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
			http.Error(w, fmt.Sprintf(`"error":"%s"`, err), http.StatusInternalServerError)
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
