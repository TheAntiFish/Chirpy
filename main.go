package main

import "net/http"

func main() {
	mux := http.NewServeMux()

	strippedFileServer := http.StripPrefix("/app/", http.FileServer(http.Dir("./")))

	mux.Handle("/app/", strippedFileServer)
	mux.HandleFunc("/healthz", ReadinessEndpoint)

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