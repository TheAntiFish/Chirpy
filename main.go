package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/TheAntiFish/Chirpy/internal/auth"
	"github.com/TheAntiFish/Chirpy/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db *database.Queries
	platform string
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

func main() {
	godotenv.Load()

	dbURL := os.Getenv("DB_URL")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Error connecting to database: %s", err)
	}
	defer db.Close()

	dbQueries := database.New(db)
	
	apiCfg := &apiConfig{
		db: dbQueries,
		platform: os.Getenv("PLATFORM"),
	}

	mux := http.NewServeMux()

	strippedFileServer := http.StripPrefix("/app/", http.FileServer(http.Dir("./")))

	mux.Handle("/app/", apiCfg.middlewareMetricsInc(strippedFileServer))

	mux.HandleFunc("GET /api/healthz", apiCfg.ReadinessEndpoint)

	mux.HandleFunc("POST /api/chirps", apiCfg.CreateChirp)
	mux.HandleFunc("GET /api/chirps", apiCfg.GetChirps)
	mux.HandleFunc("GET /api/chirps/{id}", apiCfg.GetChirpByID)

	mux.HandleFunc("POST /api/users", apiCfg.CreateUser)
	mux.HandleFunc("POST /api/login", apiCfg.LoginUser)

	mux.HandleFunc("GET /admin/metrics", apiCfg.PrintFileServerHits())
	mux.HandleFunc("POST /admin/reset", apiCfg.ResetFileServerHits())

	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	server.ListenAndServe()
}

func (cfg *apiConfig) ReadinessEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) CreateChirp(w http.ResponseWriter, r *http.Request) {
	type ChirpParams struct {
        Body string `json:"body"`
		UserID uuid.UUID `json:"user_id"`
    }

	decoder := json.NewDecoder(r.Body)
    params := ChirpParams{}
    err := decoder.Decode(&params)
    if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error decoding parameters: %s", err))
		return
    }

	if len(params.Body) > 140 {
		respondWithError(w, http.StatusBadRequest, "Chirp body is too long")
		return
	}

	cleanedBody := wordReplacement(params.Body)

	chirpParams := database.CreateChirpParams{
		Body: cleanedBody,
		UserID: params.UserID,
	}

	chirp, err := cfg.db.CreateChirp(r.Context(), chirpParams)
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error creating chirp: %s", err))
		return
	}

	returnChirp := Chirp{
		ID: chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body: chirp.Body,
		UserID: chirp.UserID,
	}

	respondWithJSON(w, http.StatusCreated, returnChirp)
}

func (cfg *apiConfig) GetChirps(w http.ResponseWriter, r *http.Request) {
	chirps, err := cfg.db.GetChirps(r.Context())
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error getting chirps: %s", err))
		return
	}

	var returnChirps []Chirp

	for _, chirp := range chirps {
		returnChirp := Chirp{
			ID: chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body: chirp.Body,
			UserID: chirp.UserID,
		}
		returnChirps = append(returnChirps, returnChirp)
	}

	respondWithJSON(w, http.StatusOK, returnChirps)
}

func (cfg *apiConfig) GetChirpByID(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.PathValue("id"), "/api/chirps/")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid chirp ID")
		return
	}

	chirp, err := cfg.db.GetChirpByID(r.Context(), id)
	if err != nil {
		respondWithError(w, 404, fmt.Sprintf("Error getting chirp: %s", err))
		return
	}

	returnChirp := Chirp{
		ID: chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body: chirp.Body,
		UserID: chirp.UserID,
	}

	respondWithJSON(w, http.StatusOK, returnChirp)
}

func (cfg *apiConfig) CreateUser(w http.ResponseWriter, r *http.Request) {
	type UserParams struct {
		Password string `json:"password"`
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	params := UserParams{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error decoding parameters: %s", err))
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error hashing password: %s", err))
		return
	}

	userParams := database.CreateUserParams{
		Email: params.Email,
		HashedPassword: hashedPassword,
	}

	user, err := cfg.db.CreateUser(r.Context(), userParams)
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error creating user: %s", err))
		return
	}

	returnUser := User{
		ID: user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email: user.Email,
	}

	respondWithJSON(w, http.StatusCreated, returnUser)
}

func (cfg *apiConfig) LoginUser(w http.ResponseWriter, r *http.Request) {
	type LoginParams struct {
		Password string `json:"password"`
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	params := LoginParams{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error decoding parameters: %s", err))
		return
	}

	user, err := cfg.db.GetUserByEmail(r.Context(), params.Email)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	match, err := auth.CheckPasswordHash(params.Password, user.HashedPassword)
	if err != nil || !match {
		respondWithError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	returnUser := User{
		ID: user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email: user.Email,
	}

	respondWithJSON(w, http.StatusOK, returnUser)
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
		if cfg.platform != "dev"{
			respondWithError(w, http.StatusForbidden, "Unauthorized")
			return
		}
		cfg.db.DeleteAllUsers(r.Context())
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