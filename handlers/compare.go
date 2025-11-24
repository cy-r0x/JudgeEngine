package handlers

import (
	"os/exec"
)

type Meta struct {
	Status    string
	Message   string
	Killed    int
	Time      float32
	Time_Wall float32
	Max_RSS   float32
}

func (h *Handler) Compare(boxPath string, maxTime *float32, maxRSS *float32, finalResult *string, strictSpace bool) {
	outputPath, expectedOutputPath, shouldReturn := h.parseMeta(boxPath, maxTime, maxRSS, finalResult)
	if shouldReturn {
		return
	}
	var diffCmd *exec.Cmd
	if strictSpace {
		diffCmd = exec.Command("diff", outputPath, expectedOutputPath)
	} else {
		diffCmd = exec.Command("diff", "-Z", "-B", outputPath, expectedOutputPath)
	}
	if _, err := diffCmd.CombinedOutput(); err != nil {
		*finalResult = "wa"
	} else {
		*finalResult = "ac"
	}
}
