package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"

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

	handler.Handle(frontPath, http.StripPrefix(frontPath, middlewareLog(cfg.middlewareMetricsInc(fileServer))))

	handler.Handle(fmt.Sprintf("GET %smetrics", adminPath), middlewareLog(cfg.handleMetrics))
	handler.Handle(fmt.Sprintf("POST %sreset", adminPath), middlewareLog(cfg.handleReset))
	handler.Handle(fmt.Sprintf("GET %shealthz", backPath), middlewareLog(cfg.handleHealthz))

	handler.Handle(fmt.Sprintf("POST %susers", backPath), middlewareLog(cfg.handlePostUser))
	handler.Handle(fmt.Sprintf("PUT %susers", backPath), middlewareLog(cfg.handlePostUser))

	handler.Handle(fmt.Sprintf("POST %schirps", backPath), middlewareLog(cfg.handlePostChirp))
	handler.Handle(fmt.Sprintf("GET %schirps", backPath), middlewareLog(cfg.handleGetChirps))
	handler.Handle(fmt.Sprintf("GET %schirps/{id}", backPath), middlewareLog(cfg.handleGetChirp))
	handler.Handle(fmt.Sprintf("DELETE %schirps/{id}", backPath), middlewareLog(cfg.handleDeleteChirps))

	handler.Handle(fmt.Sprintf("POST %srevoke", backPath), middlewareLog(cfg.handleRevoke))
	handler.Handle(fmt.Sprintf("POST %srefresh", backPath), middlewareLog(cfg.handlerRefresh))
	handler.Handle(fmt.Sprintf("POST %slogin", backPath), middlewareLog(cfg.handlerLogin))
	handler.Handle(fmt.Sprintf("POST %spolka/webhooks", backPath), middlewareLog(cfg.handlerWebhook))
	server := http.Server{
		Handler: handler,
		Addr:    ":8080",
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server Failed to start: %v", err)
	}
}
