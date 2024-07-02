package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Id           int    `json:"id"`
	Email        string `json:"email"`
	PasswordHash []byte `json:"password"`
	IsChirpyRed  bool   `json:"is_chirpy_red"`
}

type UserAuth struct {
	Id           int    `json:"id"`
	Email        string `json:"email"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	IsChirpyRed  bool   `json:"is_chirpy_red"`
}

type UserInfo struct {
	Id          int    `json:"id"`
	Email       string `json:"email"`
	IsChirpyRed bool   `json:"is_chirpy_red"`
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
	if err := os.Truncate(file, 0); err != nil {
		fmt.Printf("Failed to truncate: %v", err)
	}
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
			IsChirpyRed:  false,
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

func produceJWT(expiry, uid int) (string, error) {
	secret := os.Getenv("JWT_SECRET")

	now := time.Now().UTC()
	claims := jwt.RegisteredClaims{
		Issuer:    "chirpy",
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(expiry) * time.Second)),
		Subject:   strconv.Itoa(uid),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		fmt.Printf("There was an error signing JWT: %s\n", err)
		return "", err
	}
	return tokenString, nil
}

func authenticateUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Expiry   int    `json:"expires_in_seconds"`
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
	token := ""
	if params.Expiry == 0 {
		token, err = produceJWT(86400, storedUserData.Id)
		if err != nil {
			fmt.Printf("Something is wrong with creating JWT for login request: %s\n", err)
		}
	} else {
		token, err = produceJWT(params.Expiry, storedUserData.Id)
		if err != nil {
			fmt.Printf("Something is wrong with creating JWT for login request: %s\n", err)
		}
	}
	refreshToken := newRefreshToken()
	tokens := readTokens(refreshTokenDbFile)
	tokens.Tokens[storedUserData.Id] = RefreshToken{
		UserId:        storedUserData.Id,
		Token:         refreshToken,
		ExirationDate: time.Now().UTC().AddDate(0, 0, 60),
	}
	saveTokens(refreshTokenDbFile, tokens)

	userInfo := UserAuth{
		Id:           storedUserData.Id,
		Email:        storedUserData.Email,
		IsChirpyRed:  storedUserData.IsChirpyRed,
		Token:        token,
		RefreshToken: refreshToken,
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

func updateUser(w http.ResponseWriter, r *http.Request) {
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
	header := r.Header.Get("authorization")
	if header == "" {
		out := "Request wasn't made with header 'Authorization: Bearer <my_auth_token>'"
		w.Write([]byte(out))
		w.WriteHeader(400)
		return
	}
	bearerToken := strings.Replace(header, "Bearer ", "", 1)
	secret := os.Getenv("JWT_SECRET")
	claims := &jwt.RegisteredClaims{}
	parsedToken, err := jwt.ParseWithClaims(bearerToken, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		fmt.Printf("Error parsing claims from received token: %s\n", err)
		w.WriteHeader(401)
		return
	}
	issuer, err := parsedToken.Claims.GetIssuer()
	if err != nil {
		fmt.Printf("There was an error reading issuer from parsed token claims: %s\n", err)
	}
	if issuer != "chirpy" {
		fmt.Println("Parsed a JWT that we did not issue")
		w.WriteHeader(401)
		return
	}
	uid, err := parsedToken.Claims.GetSubject()
	users := readUsers(userDbFile)
	if duplicateUserCheck(users, params.Email) {
		out := fmt.Sprintf("Email address %s is already in use\n", params.Email)
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
	uidInt, err := strconv.Atoi(uid)
	if err != nil {
		fmt.Printf("There was an error converting token parsed user id to a number: %s\n", err)
		w.WriteHeader(401)
		return
	}
	updatedUser, ok := users.Users[uidInt]
	if !ok {
		fmt.Println("Targeted user no longer exists!")
		w.WriteHeader(401)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(params.Password), 2)
	if err != nil {
		fmt.Printf("There was an error generating a password hash: %s", err)
		w.WriteHeader(401)
		return
	}
	updatedUser.Email = params.Email
	updatedUser.PasswordHash = hash
	users.Users[uidInt] = updatedUser
	saveUsers(userDbFile, users)
	user := UserInfo{
		Id:    uidInt,
		Email: updatedUser.Email,
	}
	data, err := json.Marshal(&user)
	if err != nil {
		fmt.Printf("There was an error marshalling updated user to JSON: %s\n", err)
		w.WriteHeader(500)
		return
	}

	w.Write(data)
	w.WriteHeader(200)
	return
}
