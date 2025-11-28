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
	ctx        context.Context
}

func NewServer(config *config.Config, queue *queue.Queue, scheduler *scheduler.Scheduler, ctx context.Context) *Server {
	return &Server{
		config:    config,
		manager:   queue,
		scheduler: scheduler,
		ctx:       ctx,
	}
}

func wrapMux(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		crrTime := time.Now()
		mux.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.RequestURI, time.Since(crrTime))
	})
}

func (s *Server) Listen(port string) error {
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
	case <-s.ctx.Done():
		return nil
	}
}

func (s *Server) Shutdown() error {
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(s.ctx)
}
