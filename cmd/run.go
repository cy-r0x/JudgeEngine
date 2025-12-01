package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"time"

	"github.com/judgenot0/judge-deamon/handlers"
	"github.com/judgenot0/judge-deamon/scheduler"
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

	runner := scheduler.GetRunner(runReq.Language)
	if runner == nil {
		return "ce"
	}

	verdict, err := runner.Compile(boxId, runReq)
	if err != nil {
		return "ce"
	}

	verdict = runner.Run(boxId, runReq, handler)
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
	case <-s.ctx.Done():
		utils.SendResponse(w, http.StatusServiceUnavailable, "Server shutting down")
	case worker := <-s.scheduler.WorkChannel:
		defer func() {
			cmd := exec.Command("isolate", fmt.Sprintf("--box-id=%d", worker.Id), "--cg", "--init")
			if err := cmd.Run(); err != nil {
				log.Printf("Error resetting sandbox %d: %v", worker.Id, err)
			}
			s.scheduler.WorkChannel <- worker
		}()

		var panicked bool
		var verdict string
		func() {
			defer func() {
				if r := recover(); r != nil {
					panicked = true
				}
			}()
			verdict = run(worker.Id, &runReq, s.scheduler.Handler)
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
