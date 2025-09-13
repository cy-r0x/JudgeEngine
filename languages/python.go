package languages

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/judgenot0/judge-deamon/handlers"
	"github.com/judgenot0/judge-deamon/structs"
)

type Python struct {
}

func (p *Python) Compile(boxId int, submission *structs.Submission) {
	code := submission.Code
	boxPath := fmt.Sprintf("/var/local/lib/isolate/%d/box/", boxId)

	pyFilePath := boxPath + "main.py"
	if err := os.WriteFile(pyFilePath, []byte(code), 0644); err != nil {
		log.Printf("Error writing code to file: %v", err)
		return
	}
}

func (p *Python) Run(boxId int, submission *structs.Submission) {

	boxPath := fmt.Sprintf("/var/local/lib/isolate/%d/box/", boxId)

	var maxTime float64
	var maxRSS int
	finalResult := "Accepted"

	inputPath := boxPath + "in.txt"
	expectedOutputPath := boxPath + "expOut.txt"
	outputPath := boxPath + "out.txt"
	metaPath := boxPath + "meta.txt"

	for i, test := range submission.Testcases {
		input := test.Input
		output := test.Output

		os.WriteFile(inputPath, []byte(input), 0644)
		os.WriteFile(expectedOutputPath, []byte(output), 0644)
		os.WriteFile(outputPath, []byte(""), 0644)

		memLimit := submission.Memory * 1024
		isolateCmd := exec.Command("isolate",
			fmt.Sprintf("--box-id=%d", boxId),
			"--stdin=in.txt",
			"--stdout=out.txt",
			fmt.Sprintf("--time=%.3f", submission.Time/1000),
			fmt.Sprintf("--wall-time=%.3f", (submission.Time/1000)*1.5),
			"--fsize=10240",
			fmt.Sprintf("--mem=%d", memLimit),
			"--meta="+metaPath,
			"--run",
			"--",
			"/usr/bin/python3",
			"main.py",
		)
		_ = isolateCmd.Run()

		handlers.Compare(boxPath, &maxTime, &maxRSS, &finalResult, i)

		if finalResult != "Accepted" {
			break
		}
	}

	handlers.ProduceVerdict(submission, finalResult, maxTime, maxRSS)

}
