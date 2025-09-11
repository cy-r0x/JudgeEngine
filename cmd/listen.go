package cmd

import (
	"io"
	"log"
	"net/http"

	"github.com/judgenot0/judge-deamon/queue"
)

var queue_manager *queue.Queue

func handleSubmit(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	log.Println(body)

	err = queue_manager.QueueMessage(body)
	if err != nil {
		return
	}
	w.WriteHeader(200)
}

func initRoute(mux *http.ServeMux) {
	mux.Handle("POST /submit", http.HandlerFunc(handleSubmit))
}

func Listen(port string, queue *queue.Queue) {
	queue_manager = queue
	mux := http.NewServeMux()
	initRoute(mux)

	http.ListenAndServe(port, mux)
}
