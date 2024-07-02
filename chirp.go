package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

type Chirp struct {
	Id       int    `json:"id"`
	Body     string `json:"body"`
	AuthorId int    `json:"author_id"`
}

type ChirpData struct {
	Chirps map[int]Chirp `json:"chirps"`
}

func readChirps(file string) ChirpData {
	db, err := os.Open(file)
	if err != nil {
		fmt.Printf("There was an error opening DB for reading: %s\n", err)
	}
	defer db.Close()

	scanner := bufio.NewScanner(db)
	chirps := ChirpData{}

	for scanner.Scan() {
		data := scanner.Text()
		err := json.Unmarshal([]byte(data), &chirps)
		if err != nil {
			fmt.Printf("Error unmarshalling JSON from DB: %s\n", err)
		}
	}
	return chirps
}

func saveChirps(file string, chirps ChirpData) {
	if err := os.Truncate(file, 0); err != nil {
		fmt.Printf("Failed to truncate: %v", err)
	}
	db, err := os.OpenFile(file, os.O_WRONLY, 0666)
	if err != nil {
		fmt.Printf("There was an error opening DB for reading: %s\n", err)
	}
	defer db.Close()

	data, marshallError := json.Marshal(&chirps)
	if marshallError != nil {
		fmt.Printf("Error marshalling chirpdata to JSON: %s\n", marshallError)
	}
	_, err = db.Write(data)
	if err != nil {
		fmt.Printf("Problem writing chirps to db: %s\n", err)
	}
}

func cleanProfanity(str string) string {
	badWords := []string{"kerfuffle", "sharbert", "fornax"}
	lower := strings.ToLower(str)
	for _, val := range badWords {
		idx := strings.Index(lower, val)
		if idx == -1 {
			continue
		} else {
			str = str[:idx] + "****" + str[idx+len(val):]
			lower = lower[:idx] + "****" + lower[idx+len(val):]
		}
	}
	return str
}

func newChirp(w http.ResponseWriter, r *http.Request) {
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
	uidInt, err := strconv.Atoi(uid)
	if err != nil {
		fmt.Printf("Failed to convert user id string to int: %s\n", err)
	}

	type parameters struct {
		Body string `json:"body"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		w.WriteHeader(500)
		return
	}
	if len(params.Body) <= 140 {
		chirps := readChirps(dbFile)
		chirp := Chirp{
			Id:       (getHighestChirpId(chirps) + 1),
			Body:     cleanProfanity(params.Body),
			AuthorId: uidInt,
		}
		chirps.Chirps[chirp.Id] = chirp
		saveChirps(dbFile, chirps)
		data, err := json.Marshal(chirp)
		if err != nil {
			log.Printf("Error marshalling JSON: %s", err)
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		w.Write(data)
		return
	}
	if len(params.Body) > 140 {
		type badResponse struct {
			Error string `json:"error"`
		}
		respBody := badResponse{
			Error: "Chirp is too long",
		}
		data, err := json.Marshal(respBody)
		if err != nil {
			log.Printf("Error marshalling JSON: %s", err)
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		w.Write(data)
	}
}

func getChirps(w http.ResponseWriter, r *http.Request) {
	chirps := readChirps(dbFile)
	outSlice := []Chirp{}
	for _, val := range chirps.Chirps {
		outSlice = append(outSlice, val)
	}
	sort.Slice(outSlice, func(i, j int) bool {
		return outSlice[i].Id < outSlice[j].Id
	})
	data, err := json.Marshal(&outSlice)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(data)
}

func getChirpId(w http.ResponseWriter, r *http.Request) {
	chirps := readChirps(dbFile)
	idString := r.PathValue("chirpId")
	id, convErr := strconv.Atoi(idString)
	if convErr != nil {
		log.Printf("Bad value passed to path: %s", convErr)
		w.WriteHeader(500)
		return
	}
	chirp, ok := chirps.Chirps[id]
	if !ok {
		w.WriteHeader(404)
		return
	}

	data, err := json.Marshal(&chirp)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(data)
}

func getHighestChirpId(c ChirpData) int {
	highest := 0
	for key, _ := range c.Chirps {
		if key > highest {
			highest = key
		}
	}
	return highest
}

func deleteChirp(w http.ResponseWriter, r *http.Request) {
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
	uidInt, err := strconv.Atoi(uid)
	if err != nil {
		fmt.Printf("Failed to convert user id string to int: %s\n", err)
	}
	idString := r.PathValue("chirpId")
	id, convErr := strconv.Atoi(idString)
	if convErr != nil {
		log.Printf("Bad value passed to path: %s", convErr)
		w.WriteHeader(500)
		return
	}
	chirps := readChirps(dbFile)

	chirp, ok := chirps.Chirps[id]
	if !ok {
		w.WriteHeader(404)
		return
	}

	if chirp.AuthorId != uidInt {
		w.WriteHeader(403)
		return
	}
	delete(chirps.Chirps, chirp.Id)
	saveChirps(dbFile, chirps)
	w.WriteHeader(204)
	return
}
