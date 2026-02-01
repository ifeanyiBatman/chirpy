package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"os"

	"github.com/google/uuid"
	"github.com/ifeanyibatman/chirpy/internal/auth"
	"github.com/ifeanyibatman/chirpy/internal/database"
	"github.com/joho/godotenv"

	_ "github.com/lib/pq"
)

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

type LoginResponse struct {
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	IsChirpyRed  bool      `json:"is_chirpy_red"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
}

type User struct {
	ID          uuid.UUID `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Email       string    `json:"email"`
	IsChirpyRed bool      `json:"is_chirpy_red"`
}

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
	jwt_secret     string
	polka_secret   string
}

func main() {

	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Println(err)
	}
	apiCfg := &apiConfig{}
	apiCfg.db = database.New(db)
	apiCfg.platform = os.Getenv("PLATFORM")
	apiCfg.jwt_secret = os.Getenv("JWT_SECRET")
	apiCfg.polka_secret = os.Getenv("POLKA_SECRET")
	serveMux := http.NewServeMux()
	srv := http.Server{
		Addr:    ":8080",
		Handler: serveMux,
	}

	serveMux.Handle("/app/", http.StripPrefix("/app/", apiCfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))
	serveMux.HandleFunc("GET /api/healthz", healthz)
	//Chirps
	serveMux.HandleFunc("GET /api/chirps", apiCfg.getChirps)
	serveMux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.getChirp)
	serveMux.HandleFunc("POST /api/chirps", apiCfg.createChirp)
	serveMux.HandleFunc("DELETE /api/chirps/{chirpID}", apiCfg.deleteChirp)
	//Users
	serveMux.HandleFunc("POST /api/users", apiCfg.createUser)
	serveMux.HandleFunc("POST /api/login", apiCfg.login)
	serveMux.HandleFunc("POST /api/refresh", apiCfg.refreshToken)
	serveMux.HandleFunc("POST /api/revoke", apiCfg.revokeToken)
	serveMux.HandleFunc("PUT /api/users", apiCfg.updateUser)
	serveMux.HandleFunc("POST /api/polka/webhooks", apiCfg.upgradeUserToChirpyRed)
	//Admin
	serveMux.HandleFunc("GET /admin/metrics", apiCfg.metrics)
	serveMux.HandleFunc("POST /admin/reset", apiCfg.resetMetrics)
	srv.ListenAndServe()
}

func healthz(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))

}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) metrics(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html ")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
	</html>`, cfg.fileserverHits.Load())))
}

func (cfg *apiConfig) resetMetrics(w http.ResponseWriter, req *http.Request) {
	cfg.fileserverHits.Store(0)
	if cfg.platform != "dev" {
		w.WriteHeader(http.StatusForbidden)
		return
	} else {
		w.WriteHeader(http.StatusOK)
	}

	err := cfg.db.DeleteUsers(req.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write([]byte("Users and Chirps deleted successfully"))

}

func (cfg *apiConfig) createChirp(w http.ResponseWriter, req *http.Request) {
	type chirp struct {
		Body   string    `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}

	type errorJson struct {
		Error string `json:"error"`
	}
	type validity struct {
		Valid bool `json:"valid"`
	}
	profane := []string{"kerfuffle", "sharbert", "fornax"}

	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	validatedID, err := auth.ValidateJWT(token, cfg.jwt_secret)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var reqChirp chirp
	decoder := json.NewDecoder(req.Body)
	err = decoder.Decode(&reqChirp)
	if err != nil {
		w.WriteHeader(500)
		wrong := errorJson{
			Error: fmt.Sprintf("Something went wrong: %s", err),
		}
		dat, err := json.Marshal(wrong)
		if err != nil {
			fmt.Println(err)
		}

		w.Write(dat)
		return
	}
	if reqChirp.UserID != validatedID {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if len(reqChirp.Body) > 140 {
		w.WriteHeader(http.StatusBadRequest)
		wrong := errorJson{
			Error: "Chirp is too long",
		}
		dat, err := json.Marshal(wrong)
		if err != nil {
			fmt.Println(err)
		}
		w.Write(dat)
		return
	}

	words := strings.Split(reqChirp.Body, " ")
	cleanedWords := []string{}
	for _, word := range words {
		if slices.Contains(profane, strings.ToLower(word)) {
			cleanedWords = append(cleanedWords, "****")
		} else {
			cleanedWords = append(cleanedWords, word)
		}
	}
	cleanedChirp := strings.Join(cleanedWords, " ")

	dbChirp, err := cfg.db.CreateChirp(req.Context(), database.CreateChirpParams{
		Body:   cleanedChirp,
		UserID: reqChirp.UserID,
	})
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	res := Chirp{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		UserID:    dbChirp.UserID,
	}

	dat, err := json.Marshal(res)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write(dat)

}

func (cfg *apiConfig) createUser(w http.ResponseWriter, req *http.Request) {
	type userEmail struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	params := userEmail{}
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&params)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	args := database.CreateUserParams{
		Email:          params.Email,
		HashedPassword: hashedPassword,
	}
	user, err := cfg.db.CreateUser(req.Context(), args)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	resUser := User{
		ID:          user.ID,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
		Email:       user.Email,
		IsChirpyRed: user.IsChirpyRed,
	}
	dat, err := json.Marshal(resUser)
	if err != nil {
		fmt.Println(err)
	}
	w.WriteHeader(http.StatusCreated)
	w.Write(dat)

}

func (cfg *apiConfig) getChirps(w http.ResponseWriter, req *http.Request) {
	var chirps []database.Chirp
	var err error

	authorID := req.URL.Query().Get("author_id")
	if authorID != "" {
		authorUUID, err := uuid.Parse(authorID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		chirps, err = cfg.db.GetChirpsByUserID(req.Context(), authorUUID)
	} else {
		chirps, err = cfg.db.GetChirps(req.Context())
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	sortDirection := req.URL.Query().Get("sort")
	if sortDirection == "desc" {
		sort.Slice(chirps, func(i, j int) bool {
			return chirps[i].CreatedAt.After(chirps[j].CreatedAt)
		})
	}

	resChirps := []Chirp{}
	for _, chirp := range chirps {
		resChirps = append(resChirps, Chirp{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		})
	}

	dat, err := json.Marshal(resChirps)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(dat)

}

func (cfg *apiConfig) getChirp(w http.ResponseWriter, req *http.Request) {
	chirpIDString := req.PathValue("chirpID")
	chirpID, err := uuid.Parse(chirpIDString)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	chirp, err := cfg.db.GetChirpByID(req.Context(), chirpID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	resChirp := Chirp{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
	}
	dat, err := json.Marshal(resChirp)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(dat)
}

func (cfg *apiConfig) login(w http.ResponseWriter, req *http.Request) {
	type credentials struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	var reqCred credentials
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&reqCred)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return

	}
	hour := 3600
	user, err := cfg.db.GetUserByEmail(req.Context(), reqCred.Email)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	match, err := auth.CheckPasswordHash(reqCred.Password, user.HashedPassword)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !match {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	token, err := auth.MakeJWT(user.ID, cfg.jwt_secret, time.Duration(hour)*time.Second)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	cfg.db.CreateRefreshToken(req.Context(), database.CreateRefreshTokenParams{
		Token:     refreshToken,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(60 * 24 * time.Hour),
	})

	resUser := LoginResponse{
		ID:           user.ID,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		Email:        user.Email,
		IsChirpyRed:  user.IsChirpyRed,
		Token:        token,
		RefreshToken: refreshToken,
	}
	dat, err := json.Marshal(resUser)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(dat)

}

func (cfg *apiConfig) refreshToken(w http.ResponseWriter, req *http.Request) {
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	refreshTokenDb, err := cfg.db.GetRefreshToken(req.Context(), token)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if refreshTokenDb.ExpiresAt.Before(time.Now()) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if refreshTokenDb.RevokedAt.Valid {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	userId := refreshTokenDb.UserID
	accessToken, err := auth.MakeJWT(userId, cfg.jwt_secret, time.Hour)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	type resToken struct {
		Token string `json:"token"`
	}
	dat, err := json.Marshal(resToken{
		Token: accessToken,
	})
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(dat)
}

func (cfg *apiConfig) revokeToken(w http.ResponseWriter, req *http.Request) {
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	err = cfg.db.RevokeRefreshToken(req.Context(), token)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (cfg *apiConfig) updateUser(w http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	userID, err := auth.ValidateJWT(token, cfg.jwt_secret)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	user, err := cfg.db.UpdateUser(req.Context(), database.UpdateUserParams{
		Email:          params.Email,
		HashedPassword: hashedPassword,
		ID:             userID,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resUser := User{
		ID:          user.ID,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
		Email:       user.Email,
		IsChirpyRed: user.IsChirpyRed,
	}
	dat, err := json.Marshal(resUser)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(dat)
}

func (cfg *apiConfig) deleteChirp(w http.ResponseWriter, req *http.Request) {
	chirpID, err := uuid.Parse(req.PathValue("chirpID"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	validatedUserID, err := auth.ValidateJWT(token, cfg.jwt_secret)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	chirp, err := cfg.db.GetChirpByID(req.Context(), chirpID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if chirp.UserID != validatedUserID {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	err = cfg.db.DeleteChirp(req.Context(), chirpID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (cfg *apiConfig) upgradeUserToChirpyRed(w http.ResponseWriter, req *http.Request) {

	type polkaWebhook struct {
		Event string `json:"event"`
		Data  struct {
			UserID string `json:"user_id"`
		} `json:"data"`
	}

	webhook := polkaWebhook{}
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&webhook)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	apiKey, err := auth.GetAPIKey(req.Header)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if apiKey != cfg.polka_secret {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if webhook.Event != "user.upgraded" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	userID, err := uuid.Parse(webhook.Data.UserID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	_, err = cfg.db.UpgradeUserToChirpyRed(req.Context(), userID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)

}
