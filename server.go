package main

import (
	"encoding/json"
	"fmt"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
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

	cfg := apiConfig{
		fileserverHits: atomic.Int32{},
	}

	handler := http.NewServeMux()

	fileServer := http.FileServer(http.Dir("."))

	handler.Handle(frontPath, http.StripPrefix(frontPath, middlewareLog(cfg.middlewareMetricsInc(fileServer))))

	handler.Handle(fmt.Sprintf("GET %shealthz", backPath), middlewareLog(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("OK"))
	}))
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

	handler.Handle(fmt.Sprintf("POST %sreset", adminPath), middlewareLog(func(w http.ResponseWriter, r *http.Request) {
		_ = cfg.fileserverHits.Swap(0)
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		count := cfg.fileserverHits.Load()
		w.Write([]byte(fmt.Sprintf("Hits: %d", count)))
	}))

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

	server := http.Server{
		Handler: handler,
		Addr:    ":8080",
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server Failed to start: %v", err)
	}
}
