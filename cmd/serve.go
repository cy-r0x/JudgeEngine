package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/judgenot0/judge-deamon/config"
	"github.com/judgenot0/judge-deamon/queue"
	"github.com/judgenot0/judge-deamon/scheduler"
)

type Server struct {
	config     *config.Config
	manager    *queue.Queue
	scheduler  *scheduler.Scheduler
	httpServer *http.Server
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

func (s *Server) Listen(ctx context.Context, port string) error {
	mux := http.NewServeMux()
	s.registerRoutes(mux)
	wrapedMux := wrapMux(mux)

	addr := port
	if addr[0] != ':' {
		addr = ":" + addr
	}

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      wrapedMux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	errChan := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		return nil
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}
