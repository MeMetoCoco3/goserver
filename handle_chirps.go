package main

import (
	"encoding/json"
	"fmt"
	"github.com/MeMetoCoco3/goserver/internal/auth"
	"github.com/MeMetoCoco3/goserver/internal/database"
	"github.com/google/uuid"
	"net/http"
	"strings"
)

func (cfg *apiConfig) handleGetChirps(w http.ResponseWriter, r *http.Request) {
	chirps := make([]Chirp, 0)

	authorID := r.URL.Query().Get("author_id")
	if authorID == "" {
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

	} else {
		authorUUID, err := uuid.Parse(authorID)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), http.StatusInternalServerError)
			return
		}
		newChirps, err := cfg.db.GetChirpByAuthor(r.Context(), authorUUID)
		for _, chirp := range newChirps {
			chirps = append(chirps, Chirp{
				ID:        chirp.ID,
				CreatedAt: chirp.CreatedAt,
				UpdatedAt: chirp.UpdatedAt,
				Body:      chirp.Body,
				UserID:    chirp.UserID,
			})
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(chirps); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), http.StatusInternalServerError)
		return
	}
}

func (cfg *apiConfig) handleGetChirp(w http.ResponseWriter, r *http.Request) {
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
}

func (cfg *apiConfig) handlePostChirp(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), http.StatusUnauthorized)
		return
	}

	uuID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
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
}

func (cfg *apiConfig) handleDeleteChirps(w http.ResponseWriter, r *http.Request) {
	chirpID, err := stringToUUID(r.PathValue("id"))

	chirpData, err := cfg.db.GetChirp(r.Context(), chirpID)
	if err != nil {
		http.Error(w, `{"error": "Failed to get chirp data."}`, http.StatusNotFound)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		http.Error(w, `{"error": "Failed to get bearer token."}`, http.StatusUnauthorized)
		return
	}
	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		http.Error(w, `{"error": "Failed to get bearer token."}`, http.StatusUnauthorized)
		return
	}

	if userID != chirpData.UserID {
		http.Error(w, `{"error": "User not allowed to procede with delete."}`, 403)
		return
	}

	err = cfg.db.DeleteChirpByID(r.Context(), chirpID)
	if err != nil {
		http.Error(w, `{"error": "Failed to delete chirp."}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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
