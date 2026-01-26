package main

import (
	"encoding/json"
	"fmt"
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

	serveMux := http.NewServeMux()
	srv := http.Server{
		Addr:    ":8080",
		Handler: serveMux,
	}

	serveMux.Handle("/app/", http.StripPrefix("/app/", apiCfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))
	serveMux.HandleFunc("GET /api/healthz", healthz)
	serveMux.HandleFunc("POST /api/validate_chirp", validate_chirp)
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

}

func validate_chirp(w http.ResponseWriter, req *http.Request) {
	type chirp struct {
		Body string `json:"body"`
	}
	type errorJson struct {
		Error string `json:"error"`
	}
	type validity struct {
		Valid bool `json:"valid"`
	}
	profane := []string{"kerfuffle", "sharbert", "fornax"}

	var reqChirp chirp
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&reqChirp)
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

	w.WriteHeader(http.StatusOK)

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
	resChirp := chirp{
		Body: cleanedChirp,
	}
	dat, err := json.Marshal(resChirp)
	if err != nil {
		fmt.Println(err)
	}
	w.Write(dat)

}
