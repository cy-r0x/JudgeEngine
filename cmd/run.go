package cmd

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/judgenot0/judge-deamon/handlers"
	"github.com/judgenot0/judge-deamon/languages"
	"github.com/judgenot0/judge-deamon/structs"
	"github.com/judgenot0/judge-deamon/utils"
)

func run(boxId int, runReq *structs.Submission, handler *handlers.Handler) string {
	if runReq.Language == "" {
		return "ce"
	}
	if runReq.SourceCode == "" {
		return "ce"
	}
	var verdict structs.Verdict
	var err error
	var runner interface {
		Compile(boxId int, runReq *structs.Submission) (structs.Verdict, error)
		Run(boxId int, runReq *structs.Submission, handler *handlers.Handler) structs.Verdict
	}
	switch runReq.Language {
	case "c":
		runner = &languages.C{}
	case "cpp":
		runner = &languages.CPP{}
	case "python":
		runner = &languages.Python{}
	default:
		return "ce"
	}
	verdict, err = runner.Compile(boxId, runReq)
	if err != nil {
		return "ce"
	}
	verdict = runner.Run(boxId, runReq, handler)
	// more to go
	return verdict.Result
}

func (s *Server) handlerRun(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	var runReq structs.Submission
	if err := decoder.Decode(&runReq); err != nil {
		utils.SendResponse(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	select {
	case slave := <-s.scheduler.WorkChannel:
		defer func() {
			s.scheduler.WorkChannel <- slave
		}()

		var panicked bool
		var verdict string
		func() {
			defer func() {
				if r := recover(); r != nil {
					panicked = true
				}
			}()
			verdict = run(slave.Id, &runReq, s.scheduler.Handler)
		}()

		if panicked {
			utils.SendResponse(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		utils.SendResponse(w, http.StatusOK, map[string]string{
			"result": verdict,
		})
	case <-time.After(30 * time.Second):
		utils.SendResponse(w, http.StatusServiceUnavailable, "No workers available")
	}
}
