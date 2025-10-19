package cmd

import (
	"io"
	"net/http"

	"github.com/judgenot0/judge-deamon/utils"
)

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
