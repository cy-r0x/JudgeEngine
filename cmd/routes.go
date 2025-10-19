package cmd

import "net/http"

func (s *Server) registerRoutes(mux *http.ServeMux) {
	mux.Handle("POST /submit", http.HandlerFunc(s.handleSubmit))
	mux.Handle("POST /run", http.HandlerFunc(s.handlerRun))
}
