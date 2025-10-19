package cmd

import (
	"net/http"

	"github.com/judgenot0/judge-deamon/queue"
	"github.com/judgenot0/judge-deamon/scheduler"
)

type Server struct {
	manager   *queue.Queue
	scheduler *scheduler.Scheduler
}

func NewServer(queue *queue.Queue, scheduler *scheduler.Scheduler) *Server {
	return &Server{
		manager:   queue,
		scheduler: scheduler,
	}
}

func (s *Server) Listen(port string) {
	mux := http.NewServeMux()
	s.registerRoutes(mux)
	http.ListenAndServe(port, mux)
}
