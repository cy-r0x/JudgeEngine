package cmd

import (
	"encoding/json"
	"net/http"

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
	decoder := json.NewDecoder(r.Body)
	var runReq structs.Submission
	err := decoder.Decode(&runReq)
	if err != nil {
		utils.SendResponse(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	slave := <-s.scheduler.WorkChannel

	defer func() {
		s.scheduler.WorkChannel <- slave
	}()
	verdict := run(slave.Id, &runReq, s.scheduler.Handler)

	utils.SendResponse(w, http.StatusOK, struct {
		Result string `json:"result"`
	}{
		Result: verdict,
	})

}
