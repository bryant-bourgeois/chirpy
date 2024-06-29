package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/joho/godotenv"
)

const (
	dbFile     string = "database.json"
	userDbFile string = "users.json"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	config := new(apiConfig)

	mux := http.NewServeMux()
	server := new(http.Server)
	server.Handler = mux
	port := getPort()
	server.Addr = "localhost:" + port

	bootStrapChirpDb()
	bootStrapUserDb()

	mux.Handle("/app/*", config.middlewareMetricsIncr(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /api/healthz", healthEndpoint)
	mux.HandleFunc("GET /admin/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		page := fmt.Sprintf(`
			<html>
			        <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@picocss/pico@2/css/pico.min.css"/>
				<body>
					<main class="container">
					<h1>Welcome, Chirpy Admin</h1>
					<p>Chirpy has been visited %d times!</p>
					</main>
				</body>
			</html>
			`, config.fileserverHits)
		io.WriteString(w, page)

	})
	mux.HandleFunc("/api/reset", func(w http.ResponseWriter, r *http.Request) {
		config.middlewareMetricsReset()
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("Hits: %s", strconv.Itoa(config.fileserverHits))))
	})
	mux.HandleFunc("POST /api/chirps", newChirp)
	mux.HandleFunc("GET /api/chirps", getChirps)
	mux.HandleFunc("GET /api/chirps/{chirpId}", getChirpId)

	mux.HandleFunc("POST /api/users", newUser)
	mux.HandleFunc("PUT /api/users", updateUser)
	mux.HandleFunc("POST /api/login", authenticateUser)

	fmt.Printf("Starting server on %s\n", server.Addr)
	err = server.ListenAndServe()
	if err != nil {
		fmt.Printf("There was an error starting the server: %s", err.Error())
	}
}
