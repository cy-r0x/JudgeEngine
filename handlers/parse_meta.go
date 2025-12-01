package handlers

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func (h *Handler) parseMeta(boxPath string, maxTime *float32, maxRSS *float32, finalResult *string) (outputPath, expectedOutputPath string, shouldReturn bool) {
	metaPath := filepath.Join(boxPath, "meta.txt")
	outputPath = filepath.Join(boxPath, "out.txt")
	expectedOutputPath = filepath.Join(boxPath, "expOut. txt")

	metaContent, err := os.ReadFile(metaPath)
	if err != nil {
		log.Printf("Error reading meta file: %v", err)
		*finalResult = "ie"
		return "", "", true
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
		case "exitcode":
			if v, err := strconv.Atoi(value); err == nil {
				meta.ExitCode = v
			}
		case "exitsig":
			if v, err := strconv.Atoi(value); err == nil {
				meta.ExitSig = v
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
		case "cg-mem":
			if v, err := strconv.ParseFloat(value, 32); err == nil {
				meta.CG_Mem = float32(v)
			}
		case "cg-oom-killed":
			if v, err := strconv.Atoi(value); err == nil {
				meta.CG_OOM_Killed = v
			}
		case "csw-voluntary":
			if v, err := strconv.Atoi(value); err == nil {
				meta.CSW_Voluntary = v
			}
		case "csw-forced":
			if v, err := strconv.Atoi(value); err == nil {
				meta.CSW_Forced = v
			}
		}
	}

	if meta.Time > *maxTime {
		*maxTime = meta.Time
	}
	if meta.Max_RSS > *maxRSS {
		*maxRSS = meta.Max_RSS
	}

	// Priority 1: Check for OOM kill (Memory Limit Exceeded)
	if meta.CG_OOM_Killed == 1 {
		*finalResult = "mle"
		return "", "", true
	}

	// Priority 2: Check if killed by sandbox (time/memory limit)
	if meta.Killed == 1 {
		// If killed is present, check the status to determine why
		if meta.Status == "TO" {
			*finalResult = "tle"
			return "", "", true
		}
		// Other kill reasons would fall through to status check
	}

	// Priority 3: Check status codes
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
		return "", "", true
	}

	// Priority 4: Check for non-zero exit code (runtime error without status)
	// This handles cases where program exits with error but no status is set
	if meta.ExitCode != 0 {
		*finalResult = "re"
		return "", "", true
	}

	// All checks passed - program executed successfully, proceed to output comparison
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		log.Printf("Output file does not exist: %s", outputPath)
		*finalResult = "ie"
		return "", "", true
	}

	if _, err := os.Stat(expectedOutputPath); os.IsNotExist(err) {
		log.Printf("Expected output file does not exist: %s", expectedOutputPath)
		*finalResult = "ie"
		return "", "", true
	}

	return outputPath, expectedOutputPath, false
}
