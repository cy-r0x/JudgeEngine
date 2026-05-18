package handlers

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

// getCheckerPath returns the absolute path to a built-in testlib checker binary.
// Binaries are expected to live in ../checker/ relative to the engine binary.
func getCheckerPath(name string) string {
	// Look for checker binary relative to working directory or use env override
	exePath := os.Getenv("CHECKER_DIR")
	if exePath == "" {
		// Default: checker/ directory next to the engine binary location
		// In production, set CHECKER_DIR env var for reliability
		exePath = "checker"
	}
	return filepath.Join(exePath, name)
}

// Compare runs the old diff-based comparison (kept as fallback).
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

// CompareWithTestlib invokes a built-in testlib checker binary.
// checkerName should match a binary in the checker/ folder (e.g. "exact_token_checker").
func (h *Handler) CompareWithTestlib(boxPath string, maxTime *float32, maxRSS *float32, finalResult *string, checkerName string) {
	outputPath, expectedOutputPath, shouldReturn := h.parseMeta(boxPath, maxTime, maxRSS, finalResult)
	if shouldReturn {
		return
	}

	inputPath := filepath.Join(boxPath, "in.txt")
	checkerBin := getCheckerPath(checkerName)

	if _, err := os.Stat(checkerBin); os.IsNotExist(err) {
		log.Printf("Checker binary not found: %s", checkerBin)
		*finalResult = "ie"
		return
	}

	cmd := exec.Command(checkerBin, inputPath, outputPath, expectedOutputPath)
	out, err := cmd.CombinedOutput()

	// testlib exit codes:
	// 0 = OK (AC)
	// 1 = WA
	// 2 = PE
	// 3 = FAIL (judge/internal error)
	// 7 = Partial points (not used for built-in checkers)
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			switch exitErr.ExitCode() {
			case 1:
				*finalResult = "wa"
			case 2:
				*finalResult = "pe"
			case 3:
				log.Printf("Checker FAIL: %s", string(out))
				*finalResult = "ie"
			case 7:
				*finalResult = "ac" // built-in checkers don't emit partials
			default:
				log.Printf("Checker exited with code %d: %s", exitErr.ExitCode(), string(out))
				*finalResult = "ie"
			}
			return
		}
		log.Printf("Checker execution error: %v, output: %s", err, string(out))
		*finalResult = "ie"
		return
	}

	*finalResult = "ac"
}

// CompareFloatWithTestlib invokes the testlib float checker with a precision argument.
func (h *Handler) CompareFloatWithTestlib(boxPath string, maxTime *float32, maxRSS *float32, finalResult *string, precision *string) {
	outputPath, expectedOutputPath, shouldReturn := h.parseMeta(boxPath, maxTime, maxRSS, finalResult)
	if shouldReturn {
		return
	}

	inputPath := filepath.Join(boxPath, "in.txt")
	checkerBin := getCheckerPath("float_checker")

	if _, err := os.Stat(checkerBin); os.IsNotExist(err) {
		log.Printf("Checker binary not found: %s", checkerBin)
		*finalResult = "ie"
		return
	}

	eps := "1e-9"
	if precision != nil && *precision != "" {
		eps = *precision
	}

	cmd := exec.Command(checkerBin, inputPath, outputPath, expectedOutputPath, eps)
	out, err := cmd.CombinedOutput()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			switch exitErr.ExitCode() {
			case 1:
				*finalResult = "wa"
			case 2:
				*finalResult = "pe"
			case 3:
				log.Printf("Checker FAIL: %s", string(out))
				*finalResult = "ie"
			default:
				log.Printf("Checker exited with code %d: %s", exitErr.ExitCode(), string(out))
				*finalResult = "ie"
			}
			return
		}
		log.Printf("Checker execution error: %v, output: %s", err, string(out))
		*finalResult = "ie"
		return
	}

	*finalResult = "ac"
}
