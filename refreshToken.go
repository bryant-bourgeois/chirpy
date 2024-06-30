package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
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

func newToken() string {
	bytes := 32
	b := make([]byte, bytes)
	_, err := rand.Read(b)
	if err != nil {
		fmt.Printf("Error generating random bytes")
	}
	encoded := hex.EncodeToString(b)
	return encoded
}
