package handlers

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	structs "github.com/judgenot0/judge-deamon/structs"
)

func Compare(boxPath string, maxTime *float64, maxRSS *int, finalResult *string, testCase int) {

	metaPath := fmt.Sprintf("%s/meta.txt", boxPath)
	outputPath := fmt.Sprintf("%s/out.txt", boxPath)
	expectedOutputPath := fmt.Sprintf("%s/expOut.txt", boxPath)

	metaContent, err := os.ReadFile(metaPath)
	if err != nil {
		log.Printf("Error reading meta file: %v", err)
		return
	}

	var meta structs.Meta
	for line := range strings.SplitSeq(string(metaContent), "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		switch parts[0] {
		case "status":
			meta.Status = parts[1]
		case "message":
			meta.Message = parts[1]
		case "killed":
			meta.Killed, _ = strconv.Atoi(parts[1])
		case "time":
			meta.Time, _ = strconv.ParseFloat(parts[1], 64)
		case "time-wall":
			meta.Time_Wall, _ = strconv.ParseFloat(parts[1], 64)
		case "max-rss":
			meta.Max_RSS, _ = strconv.Atoi(parts[1])
		}
	}

	if meta.Time > *maxTime {
		*maxTime = meta.Time
	}
	if meta.Max_RSS > *maxRSS {
		*maxRSS = meta.Max_RSS
	}

	if meta.Status != "" {
		switch meta.Status {
		case "RE":
			*finalResult = "Runtime Error"
		case "SG":
			*finalResult = "Runtime Error (Signal)"
		case "TO":
			*finalResult = "Time Limit Exceeded"
		case "XX":
			*finalResult = "Internal Error"
		}
		*finalResult = fmt.Sprintf("%s on testcase %d", *finalResult, testCase)
		return
	}

	diffCmd := exec.Command("diff", "-Z", "-B", outputPath, expectedOutputPath)
	if _, err := diffCmd.CombinedOutput(); err != nil {
		*finalResult = "Wrong Answer"
	}

}
