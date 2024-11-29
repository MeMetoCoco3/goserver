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

	"github.com/MeMetoCoco3/goserver/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	who            string
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func middlewareLog(next interface{}) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Log: %s %s %s\n", r.Method, r.URL.Path, r.Host)

		switch h := next.(type) {
		case http.Handler:
			h.ServeHTTP(w, r)
		case http.HandlerFunc:
			h(w, r)
		case func(http.ResponseWriter, *http.Request):
			h(w, r)
		default:
			log.Printf("Unsupported handler type")
		}
	})
}

func main() {
	const frontPath = "/app/"
	const backPath = "/api/"
	const adminPath = "/admin/"

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

	//----------/api/validate_chirp---------------
	handler.Handle(fmt.Sprintf("POST %svalidate_chirp", backPath), middlewareLog(func(w http.ResponseWriter, r *http.Request) {
		type Req struct {
			Body string `json:"body"`
		}
		type Resp struct {
			CleanedBody string `json:"cleaned_body"`
		}

		decoder := json.NewDecoder(r.Body)
		req := Req{}
		err := decoder.Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			fmt.Printf("Error Decoding parameters: %s\n", err)
			w.WriteHeader(500)
			w.Write([]byte(`"error":"Something went wrong`))
			return
		}
		if len(req.Body) > 140 {
			w.WriteHeader(400)
			w.Write([]byte(`"error":"Chirp is too long"`))
			return
		}

		badWords := map[string]struct{}{
			"kerfuffle": {},
			"sharbert":  {},
			"fornax":    {},
		}
		words := strings.Fields(req.Body)

		for i, word := range words {
			lowerCaseWord := strings.ToLower(word)
			if _, ok := badWords[lowerCaseWord]; ok {
				words[i] = "****"
			}
		}
		resp := Resp{
			CleanedBody: strings.Join(words, " "),
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
		return
	}))

	//----------/api/healthz/---------------
	handler.Handle(fmt.Sprintf("GET %shealthz", backPath), middlewareLog(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("OK"))
	}))

	server := http.Server{
		Handler: handler,
		Addr:    ":8080",
	}
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
		}
	}))

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server Failed to start: %v", err)
	}
}
