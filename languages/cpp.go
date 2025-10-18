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

type CPP struct {
}

func (p *CPP) Compile(boxId int, submission *structs.Submission) (structs.Verdict, error) {
	code := submission.SourceCode

	boxPath := fmt.Sprintf("/var/local/lib/isolate/%d/box/", boxId)

	cppFilePath := filepath.Join(boxPath, "main.cpp")
	if err := os.WriteFile(cppFilePath, []byte(code), 0644); err != nil {
		log.Printf("Error writing code to file: %v", err)
		return structs.Verdict{}, err
	}

	outputBinary := filepath.Join(boxPath, "main")

	if _, err := exec.Command("g++", "-std=c++23", cppFilePath, "-o", outputBinary).CombinedOutput(); err != nil {
		log.Printf("Compilation error: %v", err)
		return structs.Verdict{
			Submission: submission,
			Result:     "ce",
			MaxTime:    nil,
			MaxRSS:     nil,
		}, errors.New("compilation error")
	}
	return structs.Verdict{}, nil
}

func (p *CPP) Run(boxId int, submission *structs.Submission, handler *handlers.Handler) structs.Verdict {
	boxPath := fmt.Sprintf("/var/local/lib/isolate/%d/box/", boxId)

	var maxTime float32
	var maxRSS float32
	finalResult := "ac"

	inputPath := filepath.Join(boxPath, "in.txt")
	expectedOutputPath := filepath.Join(boxPath, "expOut.txt")
	outputPath := filepath.Join(boxPath, "out.txt")
	metaPath := filepath.Join(boxPath, "meta.txt")

	for i, test := range submission.Testcases {
		input := test.Input
		output := test.ExpectedOutput

		os.WriteFile(inputPath, []byte(input), 0644)
		os.WriteFile(expectedOutputPath, []byte(output), 0644)
		os.WriteFile(outputPath, []byte(""), 0644)

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
		_ = isolateCmd.Run()

		handler.Compare(boxPath, &maxTime, &maxRSS, &finalResult, i)

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
