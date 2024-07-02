package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
)

type PolkaEvent struct {
	Event string `json:"event"`
	Data  struct {
		UserID int `json:"user_id"`
	} `json:"data"`
}

func userUpgrade(w http.ResponseWriter, r *http.Request) {
	header := r.Header.Get("authorization")
	if header == "" {
		w.WriteHeader(401)
		return
	}
	apiKey := strings.Replace(header, "ApiKey ", "", 1)
	storedApiKey := os.Getenv("POLKA_KEY")

	if apiKey != storedApiKey {
		w.WriteHeader(401)
		return
	}

	decoder := json.NewDecoder(r.Body)
	params := PolkaEvent{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		w.WriteHeader(400)
		return
	}

	if params.Event != "user.upgraded" {
		w.WriteHeader(204)
		return
	}

	users := readUsers(userDbFile)
	targetUser := User{}
	userFound := false

	for _, val := range users.Users {
		if val.Id == params.Data.UserID {
			targetUser = val
			targetUser.IsChirpyRed = true
			userFound = true
		}
	}

	if !userFound {
		w.WriteHeader(404)
		return
	}

	users.Users[targetUser.Id] = targetUser
	saveUsers(userDbFile, users)
	w.WriteHeader(204)
	return
}
