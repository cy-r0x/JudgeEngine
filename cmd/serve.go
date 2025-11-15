package cmd

import (
	"log"
	"net/http"
	"time"

	"github.com/judgenot0/judge-deamon/config"
	"github.com/judgenot0/judge-deamon/queue"
	"github.com/judgenot0/judge-deamon/scheduler"
)

type Server struct {
	config    *config.Config
	manager   *queue.Queue
	scheduler *scheduler.Scheduler
}

func NewServer(config *config.Config, queue *queue.Queue, scheduler *scheduler.Scheduler) *Server {
	return &Server{
		config:    config,
		manager:   queue,
		scheduler: scheduler,
	}
}

func wrapMux(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		crrTime := time.Now()
		mux.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.RequestURI, time.Since(crrTime))
	})
}

func (s *Server) Listen(port string) {
	mux := http.NewServeMux()
	s.registerRoutes(mux)
	wrapedMux := wrapMux(mux)
	http.ListenAndServe(port, wrapedMux)
}
