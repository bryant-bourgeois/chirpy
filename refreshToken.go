package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type RefreshToken struct {
	UserId        int
	Token         string
	ExirationDate time.Time
}

type RefreshTokens struct {
	Tokens map[int]RefreshToken
}

func saveTokens(file string, tokens RefreshTokens) {
	db, err := os.OpenFile(file, os.O_WRONLY, 0666)
	if err != nil {
		fmt.Printf("There was an error opening DB for reading: %s\n", err)
	}
	defer db.Close()

	data, marshallError := json.Marshal(&tokens)
	if marshallError != nil {
		fmt.Printf("Error marshalling token data to JSON: %s\n", marshallError)
	}
	_, err = db.Write(data)
	if err != nil {
		fmt.Printf("Problem writing tokens to db: %s\n", err)
	}
}

func readTokens(file string) RefreshTokens {
	db, err := os.Open(file)
	if err != nil {
		fmt.Printf("There was an error opening DB for reading: %s\n", err)
	}
	defer db.Close()

	scanner := bufio.NewScanner(db)
	tokens := RefreshTokens{}

	for scanner.Scan() {
		data := scanner.Text()
		err := json.Unmarshal([]byte(data), &tokens)
		if err != nil {
			fmt.Printf("Error unmarshalling JSON from DB: %s\n", err)
		}
	}
	return tokens
}

func newRefreshToken() string {
	bytes := 32
	b := make([]byte, bytes)
	_, err := rand.Read(b)
	if err != nil {
		fmt.Printf("Error generating random bytes")
	}
	encoded := hex.EncodeToString(b)
	return encoded
}

func refreshUserAuth(w http.ResponseWriter, r *http.Request) {
	header := r.Header.Get("authorization")
	if header == "" {
		out := "Request wasn't made with header 'Authorization: Bearer <my_auth_token>'"
		w.Write([]byte(out))
		w.WriteHeader(400)
		return
	}
	bearerToken := strings.Replace(header, "Bearer ", "", 1)
	refreshTokens := readTokens(refreshTokenDbFile)
	tokenExists := false
	tokenValid := false
	targetToken := RefreshToken{}

	for _, val := range refreshTokens.Tokens {
		if val.Token == bearerToken {
			targetToken = val
			tokenExists = true
			if val.ExirationDate.After(time.Now().UTC()) {
				tokenValid = true
			}
			break
		}
	}

	if !tokenExists || !tokenValid {
		w.WriteHeader(401)
		return
	}

	type parameters struct {
		Token string `json:"token"`
	}
	authToken, err := produceJWT(3600, targetToken.UserId)
	if err != nil {
		fmt.Printf("Error creating JWT with supplied parameters: %s\n", err)
	}

	responseData := parameters{
		Token: authToken,
	}

	data, err := json.Marshal(&responseData)
	if err != nil {
		fmt.Printf("Error marshalling JWT response to JSON: %s\n", err)
	}

	w.Write(data)
	w.WriteHeader(200)
	return
}

func revokeUserAuth(w http.ResponseWriter, r *http.Request) {
}
