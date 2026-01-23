package main

import "net/http"

func main(){

	serveMux := http.NewServeMux()
	srv := http.Server {
		Addr : ":8080" ,
		Handler : serveMux }
		
	serveMux.Handle("/app/",http.StripPrefix("/app/", http.FileServer(http.Dir("."))))
	serveMux.HandleFunc("/healthz",healthz)
	srv.ListenAndServe()
}

func healthz (w http.ResponseWriter,req *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	
	}