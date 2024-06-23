package main

import (
	"fmt"
	"os"
)

func getPort() string {
	port := os.Getenv("PORT")
	if len(port) < 1 {
		port = "8080"
	}
	return port
}

func bootStrapChirpDb() {
	db, err := os.OpenFile(dbFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Printf("Could not open db: %s", err)
		os.Exit(1)
	}
	dbInfo, _ := db.Stat()
	if dbInfo.Size() <= 0 {
		db.Close()
		chirps := ChirpData{Chirps: make(map[int]Chirp)}
		saveChirps(dbFile, chirps)
	}
}

func bootStrapUserDb() {
	db, err := os.OpenFile(userDbFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Printf("Could not open db: %s", err)
		os.Exit(1)
	}
	dbInfo, _ := db.Stat()
	if dbInfo.Size() <= 0 {
		db.Close()
		users := UserData{Users: make(map[int]User)}
		saveUsers(userDbFile, users)
	}
}
