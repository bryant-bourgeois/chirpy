package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
)

type Chirp struct {
	Id   int    `json:"id"`
	Body string `json:"body"`
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

func newChirp(w http.ResponseWriter, r *http.Request) {

	type parameters struct {
		Body string `json:"body"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		w.WriteHeader(500)
		return
	}
	if len(params.Body) <= 140 {
		chirps := readChirps(dbFile)
		chirp := Chirp{
			Id:   (len(chirps.Chirps) + 1),
			Body: cleanProfanity(params.Body),
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
