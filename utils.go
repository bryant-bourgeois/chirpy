package main

import (
	"os"
)

func getPort() string {
	port := os.Getenv("PORT")
	if len(port) < 1 {
		port = "8080"
	}
	return port
}
