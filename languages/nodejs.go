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

type NodeJS struct {
}

func resolveNodeBinary() (string, error) {
	nodeBinary, err := exec.LookPath("node")
	if err != nil {
		return "", errors.New("node executable not found")
	}

	return nodeBinary, nil
}

func (p *NodeJS) Compile(boxId int, submission *structs.Submission) (structs.Verdict, error) {
	code := submission.SourceCode
	boxPath := fmt.Sprintf("/var/local/lib/isolate/%d/box/", boxId)
	nodeBinary, err := resolveNodeBinary()
	if err != nil {
		log.Printf("Node.js executable not found: %v", err)
		return structs.Verdict{
			Submission: submission,
			Result:     "ce",
			MaxTime:    nil,
			MaxRSS:     nil,
		}, err
	}

	jsFilePath := filepath.Join(boxPath, "main.js")
	if err := os.WriteFile(jsFilePath, []byte(code), 0644); err != nil {
		log.Printf("Error writing code to file: %v", err)
		return structs.Verdict{}, err
	}

	output, err := exec.Command(nodeBinary, "--check", jsFilePath).CombinedOutput()
	if err != nil {
		log.Printf("Node.js syntax error: %v, output: %s", err, string(output))
		return structs.Verdict{
			Submission: submission,
			Result:     "ce",
			MaxTime:    nil,
			MaxRSS:     nil,
		}, errors.New("compilation error")
	}

	return structs.Verdict{}, nil
}

func (p *NodeJS) Run(boxId int, submission *structs.Submission, handler *handlers.Handler) structs.Verdict {
	boxPath := fmt.Sprintf("/var/local/lib/isolate/%d/box/", boxId)
	nodeBinary, err := resolveNodeBinary()
	if err != nil {
		log.Printf("Node.js executable not found: %v", err)
		result := "ie"
		return structs.Verdict{
			Submission: submission,
			Result:     result,
			MaxTime:    nil,
			MaxRSS:     nil,
		}
	}

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
			"--cg",
			"--stdin=in.txt",
			"--stdout=out.txt",
			fmt.Sprintf("--time=%.3f", submission.TimeLimit),
			fmt.Sprintf("--wall-time=%.3f", (submission.TimeLimit)*1.5),
			"--fsize=10240",
			fmt.Sprintf("--cg-mem=%d", int(memLimit)),
			fmt.Sprintf("--meta=%s", metaPath),
			"--run",
			"--",
			nodeBinary,
			"main.js",
		)

		_ = isolateCmd.Run()

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
