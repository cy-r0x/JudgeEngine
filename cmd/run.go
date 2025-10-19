package cmd

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/judgenot0/judge-deamon/handlers"
	"github.com/judgenot0/judge-deamon/languages"
	"github.com/judgenot0/judge-deamon/structs"
	"github.com/judgenot0/judge-deamon/utils"
)

func run(boxId int, runReq *structs.Submission, handler *handlers.Handler) string {
	var verdict structs.Verdict
	var err error
	switch runReq.Language {
	case "cpp":
		var cpp languages.CPP

		verdict, err = cpp.Compile(boxId, runReq)
		if err != nil {
			if verdict.Result == "ce" {
				return verdict.Result
			} else {
				log.Println(err)
			}
		} else {
			verdict = cpp.Run(boxId, runReq, handler)
		}
	case "python":
		var py languages.Python
		verdict, err = py.Compile(boxId, runReq)
		if err != nil {
			if verdict.Result == "ce" {
				return verdict.Result
			} else {
				log.Println(err)
			}
		} else {
			verdict = py.Run(boxId, runReq, handler)
		}
	default:
		log.Printf("Unsupported!")
	}
	// more to go
	return verdict.Result
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
		slave := <-s.scheduler.WorkChannel

		defer func() {
			s.scheduler.WorkChannel <- slave
			wg.Done()
		}()
		verdict := run(slave.Id, &runReq, s.scheduler.Handler)

		utils.SendResponse(w, http.StatusOK, struct {
			Result string `json:"result"`
		}{
			Result: verdict,
		})

	}()

	wg.Wait()
}
