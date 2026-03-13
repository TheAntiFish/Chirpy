package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strings"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func main() {
	apiCfg := &apiConfig{}

	mux := http.NewServeMux()

	strippedFileServer := http.StripPrefix("/app/", http.FileServer(http.Dir("./")))

	mux.Handle("/app/", apiCfg.middlewareMetricsInc(strippedFileServer))
	mux.HandleFunc("GET /api/healthz", ReadinessEndpoint)
	mux.HandleFunc("POST /api/validate_chirp", ValidateEndpoint)
	mux.HandleFunc("GET /admin/metrics", apiCfg.PrintFileServerHits())
	mux.HandleFunc("POST /admin/reset", apiCfg.ResetFileServerHits())

	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	server.ListenAndServe()
}

func ReadinessEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func ValidateEndpoint(w http.ResponseWriter, r *http.Request) {
	type ChirpParams struct {
        Body string `json:"body"`
    }

	decoder := json.NewDecoder(r.Body)
    params := ChirpParams{}
    err := decoder.Decode(&params)
    if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		w.WriteHeader(500)
		return
    }

	if len(params.Body) > 140 {
		respondWithError(w, http.StatusBadRequest, "Chirp body is too long")
		return
	}

	cleanedBody := wordReplacement(params.Body)
	respondWithJSON(w, http.StatusOK, map[string]interface{}{"cleaned_body": cleanedBody})
}

func (cfg *apiConfig) PrintFileServerHits() http.HandlerFunc{
	return func(w http.ResponseWriter, r *http.Request) {
		hits := cfg.fileserverHits.Load()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(200)
		html := fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", hits)
		w.Write([]byte(html))
	}
}

func (cfg *apiConfig) ResetFileServerHits() http.HandlerFunc{
	return func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Store(0)

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
	}
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func wordReplacement(input string) string {
	wordsToReplace := []string{
		"kerfuffle",
		"sharbert",
		"fornax",
	}

	words := strings.Split(input, " ")

	fixedString := ""

	for _, word := range words {
		if slices.Contains(wordsToReplace, strings.ToLower(word)) {
			fixedString += "**** "
		} else {
			fixedString += word + " "
		}
	}

	return strings.TrimSpace(fixedString)
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	type returnVals struct {
        Error string `json:"error"`
    }

    respBody := returnVals{
        Error: msg,
    }
    dat, err := json.Marshal(respBody)
	if err != nil {
			log.Printf("Error marshalling JSON: %s", err)
			w.WriteHeader(500)
			return
	}
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    w.Write(dat)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
    dat, err := json.Marshal(payload)
	if err != nil {
			log.Printf("Error marshalling JSON: %s", err)
			w.WriteHeader(500)
			return
	}
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    w.Write(dat)
}