package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
)

func main() {
	config := new(apiConfig)
	mux := http.NewServeMux()
	server := new(http.Server)
	server.Handler = mux
	port := getPort()
	server.Addr = "localhost:" + port

	mux.Handle("/app/*", config.middlewareMetricsIncr(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("GET /admin/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		// w.Write([]byte(fmt.Sprintf("Hits: %s", strconv.Itoa(config.fileserverHits))))
		page := fmt.Sprintf(`
			<html>

				<body>
					<h1>Welcome, Chirpy Admin</h1>
					<p>Chirpy has been visited %d times!</p>
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
	mux.HandleFunc("/api/validate_chirp", func(w http.ResponseWriter, r *http.Request) {
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
			type returnVals struct {
				Valid       bool   `json:"valid"`
				CleanedBody string `json:"cleaned_body"`
			}
			respBody := returnVals{
				Valid:       true,
				CleanedBody: cleanProfanity(params.Body),
			}
			dat, err := json.Marshal(respBody)
			if err != nil {
				log.Printf("Error marshalling JSON: %s", err)
				w.WriteHeader(500)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			w.Write(dat)
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
	})

	fmt.Printf("Starting server on %s\n", server.Addr)
	err := server.ListenAndServe()
	if err != nil {
		fmt.Printf("There was an error starting the server: %s", err.Error())
	}
}
