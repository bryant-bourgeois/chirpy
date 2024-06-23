package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
)

type User struct {
	Id           int    `json:"id"`
	Email        string `json:"email"`
	PasswordHash []byte `json:"password"`
}

type UserInfo struct {
	Id    int    `json:"id"`
	Email string `json:"email"`
}

type UserData struct {
	Users map[int]User `json:"users"`
}

func readUsers(file string) UserData {
	db, err := os.Open(file)
	if err != nil {
		fmt.Printf("There was an error opening DB for reading: %s\n", err)
	}
	defer db.Close()

	scanner := bufio.NewScanner(db)
	users := UserData{}

	for scanner.Scan() {
		data := scanner.Text()
		err := json.Unmarshal([]byte(data), &users)
		if err != nil {
			fmt.Printf("Error unmarshalling JSON from DB: %s\n", err)
		}
	}
	return users
}

func saveUsers(file string, users UserData) {
	db, err := os.OpenFile(file, os.O_WRONLY, 0666)
	if err != nil {
		fmt.Printf("There was an error opening DB for reading: %s\n", err)
	}
	defer db.Close()

	data, marshallError := json.Marshal(&users)
	if marshallError != nil {
		fmt.Printf("Error marshalling chirpdata to JSON: %s\n", marshallError)
	}
	_, err = db.Write(data)
	if err != nil {
		fmt.Printf("Problem writing chirps to db: %s\n", err)
	}
}

func validateEmail(email string) bool {
	regex := regexp.MustCompile(`^(([^<>()[\]\\.,;:\s@"]+(\.[^<>()[\]\\.,;:\s@"]+)*)|(".+"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$`)
	if regex.MatchString(email) {
		return true
	}
	return false
}

func duplicateUserCheck(users UserData, email string) bool {
	for _, val := range users.Users {
		if strings.ToLower(val.Email) == strings.ToLower(email) {
			return true
		}
	}
	return false
}

func newUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		w.WriteHeader(500)
		return
	}

	users := readUsers(userDbFile)

	if duplicateUserCheck(users, params.Email) {
		out := fmt.Sprintf("Email address %s already exists\n", params.Email)
		w.Write([]byte(out))
		w.WriteHeader(400)
		return
	}

	if !validateEmail(params.Email) {
		out := fmt.Sprintf("%s is not a valid email address\n", params.Email)
		w.Write([]byte(out))
		w.WriteHeader(400)
		return
	}

	if validateEmail(params.Email) {
		users := readUsers(userDbFile)
		hash, err := bcrypt.GenerateFromPassword([]byte(params.Password), 2)
		if err != nil {
			fmt.Printf("There was an error generating a password hash: %s", err)
		}
		user := User{
			Id:           (len(users.Users) + 1),
			Email:        params.Email,
			PasswordHash: hash,
		}
		userResp := UserInfo{
			Id:    (len(users.Users) + 1),
			Email: params.Email,
		}
		users.Users[user.Id] = user
		saveUsers(userDbFile, users)
		data, err := json.Marshal(userResp)
		if err != nil {
			fmt.Printf("Error marshalling userInfo to JSON: %s\n", err)
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		w.Write(data)
		return
	}
}

func authenticateUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	params := parameters{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&params)
	if err != nil {
		fmt.Printf("There was an error decoding user request from JSON: %s\n", err)
		w.WriteHeader(500)
		return
	}
	users := readUsers(userDbFile)
	storedUserData := User{}
	for _, val := range users.Users {
		if strings.ToLower(val.Email) == strings.ToLower(params.Email) {
			storedUserData = val
		}
	}
	if storedUserData.Email == "" {
		w.WriteHeader(401)
		w.Write([]byte("User does not exist or password was incorrect: 401 Unauthorized"))
		return
	}
	err = bcrypt.CompareHashAndPassword(storedUserData.PasswordHash, []byte(params.Password))
	if err != nil {
		w.WriteHeader(401)
		w.Write([]byte("User does not exist or password was incorrect: 401 Unauthorized"))
		return
	}
	userInfo := UserInfo{
		Id:    storedUserData.Id,
		Email: storedUserData.Email,
	}
	data, marshallErr := json.Marshal(userInfo)
	if marshallErr != nil {
		fmt.Printf("There was an error marshalling userInfo to JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(data)
	return
}
