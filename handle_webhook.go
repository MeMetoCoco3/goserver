package main

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerWebhook(w http.ResponseWriter, r *http.Request) {
	type Params struct {
		Event string `json:"event"`
		Data  struct {
			UserID uuid.UUID `json:"user_id"`
		} `json:"data"`
	}
	params := Params{}
	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		http.Error(w, `{"error":"Error decoding json data."}`, http.StatusInternalServerError)
		return
	}

	if params.Event != "user.upgraded" {
		http.Error(w, `{"error":"Event not allowed."}`, http.StatusNoContent)
		return
	}

	_, err = cfg.db.SetRedUser(r.Context(), params.Data.UserID)
	if err != nil {
		http.Error(w, `{"error":"Event not allowed."}`, http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)

}
