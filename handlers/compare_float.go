package handlers

import (
	"bufio"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
)

func (h *Handler) CompareFloat(boxPath string, maxTime *float32, maxRSS *float32, finalResult *string, strictSpace bool, precision *string) {
	outputPath, expectedOutputPath, shouldReturn := h.parseMeta(boxPath, maxTime, maxRSS, finalResult)
	if shouldReturn {
		return
	}

	// Compare floating point outputs with precision tolerance
	outputFile, err := os.Open(outputPath)
	if err != nil {
		log.Printf("Error opening output file: %v", err)
		*finalResult = "ie"
		return
	}
	defer outputFile.Close()

	expectedFile, err := os.Open(expectedOutputPath)
	if err != nil {
		log.Printf("Error opening expected output file: %v", err)
		*finalResult = "ie"
		return
	}
	defer expectedFile.Close()

	outputScanner := bufio.NewScanner(outputFile)
	expectedScanner := bufio.NewScanner(expectedFile)

	epsilon, err := strconv.ParseFloat(*precision, 64)
	if err != nil || epsilon <= 0 {
		epsilon = 1e-6 // default precision
	}

	for {
		hasOutput := outputScanner.Scan()
		hasExpected := expectedScanner.Scan()

		if !hasOutput && !hasExpected {
			// Both files ended at the same time - success
			break
		}

		if hasOutput != hasExpected {
			// Files have different number of lines
			*finalResult = "wa"
			return
		}

		outputLine := strings.TrimSpace(outputScanner.Text())
		expectedLine := strings.TrimSpace(expectedScanner.Text())

		if !strictSpace {
			outputLine = strings.Join(strings.Fields(outputLine), " ")
			expectedLine = strings.Join(strings.Fields(expectedLine), " ")
		}

		// Split lines into tokens
		outputTokens := strings.Fields(outputLine)
		expectedTokens := strings.Fields(expectedLine)

		if len(outputTokens) != len(expectedTokens) {
			*finalResult = "wa"
			return
		}

		// Compare each token
		for i := 0; i < len(outputTokens); i++ {
			outputVal, outputErr := strconv.ParseFloat(outputTokens[i], 64)
			expectedVal, expectedErr := strconv.ParseFloat(expectedTokens[i], 64)

			if outputErr != nil && expectedErr != nil {
				// Both are not numbers, compare as strings
				if outputTokens[i] != expectedTokens[i] {
					*finalResult = "wa"
					return
				}
			} else if outputErr != nil || expectedErr != nil {
				// One is a number, the other is not
				*finalResult = "wa"
				return
			} else {
				// Both are numbers, compare with epsilon

				// Use relative error if values are large, absolute error otherwise
				diff := math.Abs(outputVal - expectedVal)
				maxVal := math.Max(math.Abs(outputVal), math.Abs(expectedVal))

				// Compute tolerance
				tolerance := epsilon * (1 + maxVal)
				// If difference exceeds tolerance -> WA
				if diff > tolerance {
					*finalResult = "wa"
					return
				}
			}
		}
	}

	if err := outputScanner.Err(); err != nil {
		log.Printf("Error reading output file: %v", err)
		*finalResult = "ie"
		return
	}

	if err := expectedScanner.Err(); err != nil {
		log.Printf("Error reading expected output file: %v", err)
		*finalResult = "ie"
		return
	}

	*finalResult = "ac"
}
