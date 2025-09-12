package cmd

import (
	"io"
	"log"
	"net/http"

	"github.com/judgenot0/judge-deamon/queue"
)

type Server struct {
	manager *queue.Queue
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) handleSubmit(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	log.Println(body)

	err = s.manager.QueueMessage(body)
	if err != nil {
		return
	}
	w.WriteHeader(200)
}

func (s *Server) initRoute(mux *http.ServeMux) {
	mux.Handle("POST /submit", http.HandlerFunc(s.handleSubmit))
}

func (s *Server) Listen(port string, queue *queue.Queue) {
	s.manager = queue
	mux := http.NewServeMux()
	s.initRoute(mux)

	http.ListenAndServe(port, mux)
}
