package handlers

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type Meta struct {
	Status    string
	Message   string
	Killed    int
	Time      float32
	Time_Wall float32
	Max_RSS   float32
}

func (h *Handler) Compare(boxPath string, maxTime *float32, maxRSS *float32, finalResult *string, testCase int) {
	metaPath := filepath.Join(boxPath, "meta.txt")
	outputPath := filepath.Join(boxPath, "out.txt")
	expectedOutputPath := filepath.Join(boxPath, "expOut.txt")

	metaContent, err := os.ReadFile(metaPath)
	if err != nil {
		log.Printf("Error reading meta file: %v", err)
		*finalResult = "ie"
		return
	}

	var meta Meta
	for _, line := range strings.Split(string(metaContent), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		switch key {
		case "status":
			meta.Status = value
		case "message":
			meta.Message = value
		case "killed":
			if v, err := strconv.Atoi(value); err == nil {
				meta.Killed = v
			}
		case "time":
			if v, err := strconv.ParseFloat(value, 32); err == nil {
				meta.Time = float32(v)
			}
		case "time-wall":
			if v, err := strconv.ParseFloat(value, 32); err == nil {
				meta.Time_Wall = float32(v)
			}
		case "max-rss":
			if v, err := strconv.ParseFloat(value, 32); err == nil {
				meta.Max_RSS = float32(v)
			}
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
			*finalResult = "re"
		case "SG":
			*finalResult = "re"
		case "TO":
			*finalResult = "tle"
		case "XX":
			*finalResult = "ie"
		}
		return
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		log.Printf("Output file does not exist: %s", outputPath)
		*finalResult = "ie"
		return
	}

	if _, err := os.Stat(expectedOutputPath); os.IsNotExist(err) {
		log.Printf("Expected output file does not exist: %s", expectedOutputPath)
		*finalResult = "ie"
		return
	}

	diffCmd := exec.Command("diff", "-Z", "-B", outputPath, expectedOutputPath)
	if _, err := diffCmd.CombinedOutput(); err != nil {
		*finalResult = "wa"
	} else {
		*finalResult = "ac"
	}
}
