package languages

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/judgenot0/judge-deamon/handlers"
	"github.com/judgenot0/judge-deamon/structs"
)

type C struct {
}

func (p *C) Compile(boxId int, submission *structs.Submission) (structs.Verdict, error) {
	code := submission.SourceCode

	boxPath := fmt.Sprintf("/var/local/lib/isolate/%d/box/", boxId)

	cFilePath := filepath.Join(boxPath, "main.c")
	if err := os.WriteFile(cFilePath, []byte(code), 0644); err != nil {
		log.Printf("Error writing code to file: %v", err)
		return structs.Verdict{}, err
	}

	outputBinary := filepath.Join(boxPath, "main")

	output, err := exec.Command(
		"gcc",
		"-std=gnu11",
		"-O2",
		"-pipe",
		"-s",
		cFilePath,
		"-o", outputBinary,
	).CombinedOutput()

	if err != nil {
		log.Printf("Compilation error: %v, output: %s", err, string(output))
		return structs.Verdict{
			Submission: submission,
			Result:     "ce",
			MaxTime:    nil,
			MaxRSS:     nil,
		}, errors.New("compilation error")
	}

	if _, err := os.Stat(outputBinary); os.IsNotExist(err) {
		log.Printf("Compilation succeeded but binary not found: %s", outputBinary)
		return structs.Verdict{
			Submission: submission,
			Result:     "ce",
			MaxTime:    nil,
			MaxRSS:     nil,
		}, errors.New("binary not created")
	}

	return structs.Verdict{}, nil
}

func (p *C) Run(boxId int, submission *structs.Submission, handler *handlers.Handler) structs.Verdict {
	boxPath := fmt.Sprintf("/var/local/lib/isolate/%d/box/", boxId)

	var maxTime float32
	var maxRSS float32
	finalResult := "ac"

	inputPath := filepath.Join(boxPath, "in.txt")
	expectedOutputPath := filepath.Join(boxPath, "expOut.txt")
	outputPath := filepath.Join(boxPath, "out.txt")
	metaPath := filepath.Join(boxPath, "meta.txt")

	for _, test := range submission.Testcases {
		input := test.Input
		output := test.ExpectedOutput

		if err := os.WriteFile(inputPath, []byte(input), 0644); err != nil {
			log.Printf("Error writing input file: %v", err)
			finalResult = "ie"
			break
		}

		if err := os.WriteFile(expectedOutputPath, []byte(output), 0644); err != nil {
			log.Printf("Error writing expected output file: %v", err)
			finalResult = "ie"
			break
		}

		if err := os.WriteFile(outputPath, []byte(""), 0644); err != nil {
			log.Printf("Error writing output file: %v", err)
			finalResult = "ie"
			break
		}

		memLimit := submission.MemoryLimit * 1024
		isolateCmd := exec.Command("isolate",
			fmt.Sprintf("--box-id=%d", boxId),
			"--stdin=in.txt",
			"--stdout=out.txt",
			fmt.Sprintf("--time=%.3f", submission.Timelimit),
			fmt.Sprintf("--wall-time=%.3f", (submission.Timelimit)*1.5),
			"--fsize=10240",
			fmt.Sprintf("--mem=%d", int(memLimit)),
			fmt.Sprintf("--meta=%s", metaPath),
			"--run",
			"--",
			"./main",
		)

		if err := isolateCmd.Run(); err != nil {
			log.Printf("Error running isolate command: %v", err)
		}
		switch submission.CheckerType {
		case "float":
			handler.CompareFloat(boxPath, &maxTime, &maxRSS, &finalResult, submission.CheckerStrictSpace, submission.CheckerPrecision)
		default:
			handler.Compare(boxPath, &maxTime, &maxRSS, &finalResult, submission.CheckerStrictSpace)
		}

		if finalResult != "ac" {
			break
		}
	}
	return structs.Verdict{
		Submission: submission,
		Result:     finalResult,
		MaxTime:    &maxTime,
		MaxRSS:     &maxRSS,
	}
}
