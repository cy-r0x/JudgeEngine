package languages

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/judgenot0/judge-deamon/handlers"
	"github.com/judgenot0/judge-deamon/structs"
)

type Python struct {
}

func (p *Python) Compile(boxId int, submission *structs.Submission) (structs.Verdict, error) {
	code := submission.SourceCode
	boxPath := fmt.Sprintf("/var/local/lib/isolate/%d/box/", boxId)

	pyFilePath := filepath.Join(boxPath, "main.py")
	if err := os.WriteFile(pyFilePath, []byte(code), 0644); err != nil {
		log.Printf("Error writing code to file: %v", err)
		return structs.Verdict{}, err
	}
	return structs.Verdict{}, nil
}

func (p *Python) Run(boxId int, submission *structs.Submission, handler *handlers.Handler) structs.Verdict {
	boxPath := fmt.Sprintf("/var/local/lib/isolate/%d/box/", boxId)

	var maxTime float32
	var maxRSS float32
	finalResult := "ac"

	inputPath := filepath.Join(boxPath, "in.txt")
	expectedOutputPath := filepath.Join(boxPath, "expOut.txt")
	outputPath := filepath.Join(boxPath, "out.txt")
	metaPath := filepath.Join(boxPath, "meta.txt")

	memLimit := submission.MemoryLimit * 1024

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
			"/usr/bin/python3",
			"main.py",
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
			data, err := os.ReadFile(outputPath)
			if err != nil {
				log.Printf("Error reading output file for debugging: %v", err)
			} else {
				log.Printf("Output for debugging:\n%s", string(data))
			}
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
