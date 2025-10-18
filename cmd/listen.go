package cmd

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/judgenot0/judge-deamon/handlers"
	"github.com/judgenot0/judge-deamon/languages"
	"github.com/judgenot0/judge-deamon/queue"
	"github.com/judgenot0/judge-deamon/scheduler"
	"github.com/judgenot0/judge-deamon/structs"
	"github.com/judgenot0/judge-deamon/utils"
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

func (s *Server) handleSubmit(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	submission, err := io.ReadAll(r.Body)
	if err != nil {
		utils.SendResponse(w, http.StatusBadRequest, "Failed to read request body")
		return
	}

	err = s.manager.QueueMessage(submission)
	if err != nil {
		utils.SendResponse(w, http.StatusBadRequest, "Failed to Queue submission")
		return
	}
	utils.SendResponse(w, http.StatusOK, "")
}
func run(boxId int, runReq *structs.Submission, handler *handlers.Handler) structs.Verdict {
	var Verdict structs.Verdict
	var err error
	switch runReq.Language {
	case "cpp":
		var cpp languages.CPP
		Verdict, err = cpp.Compile(boxId, runReq)
		if err != nil {
			if Verdict.Result == "ce" {
				return Verdict
			} else {
				log.Println(err)
			}
		} else {
			Verdict = cpp.Run(boxId, runReq, handler)
		}
	case "python":
	default:
		log.Printf("Unsupported!")
	}
	// more to go
	return Verdict
}

func (s *Server) handlerRun(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	decoder := json.NewDecoder(r.Body)
	var runReq structs.Submission
	err := decoder.Decode(&runReq)
	if err != nil {
		utils.SendResponse(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer func() {
			wg.Done()
		}()
		slave := <-s.scheduler.WorkChannel
		defer func() {
			s.scheduler.WorkChannel <- slave
		}()
		verdict := run(slave.Id, &runReq, s.scheduler.Handler)
		utils.SendResponse(w, http.StatusOK, verdict)
	}()
	wg.Wait()
}

func (s *Server) hudai(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		utils.SendResponse(w, http.StatusNoContent, "")
		return
	}
}

func (s *Server) initRoute(mux *http.ServeMux) {
	mux.Handle("POST /submit", http.HandlerFunc(s.handleSubmit))
	mux.Handle("POST /run", http.HandlerFunc(s.handlerRun))
	mux.Handle("OPTIONS /run", http.HandlerFunc(s.hudai))
}

func (s *Server) Listen(port string) {
	mux := http.NewServeMux()
	s.initRoute(mux)
	http.ListenAndServe(port, mux)
}
