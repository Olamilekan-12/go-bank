package main

import (
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})
	log.Println("Go Bank API listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
