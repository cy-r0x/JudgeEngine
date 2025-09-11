package cmd

import (
	"log"
	"net/http"
)

func handleSubmit(w http.ResponseWriter, r *http.Request) {
	log.Println(r.Body)
	w.WriteHeader(200)
}

func initRoute(mux *http.ServeMux) {
	mux.Handle("POST /submit", http.HandlerFunc(handleSubmit))
}

func Listen(port string) {
	mux := http.NewServeMux()
	initRoute(mux)

	http.ListenAndServe(port, mux)
}
