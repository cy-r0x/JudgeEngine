package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func Compare(boxPath string, maxTime *float64, maxRSS *int, finalResult *string, testCase int, id int) {

	metaPath := fmt.Sprintf("%s/meta.txt", boxPath)
	outputPath := fmt.Sprintf("%s/out.txt", boxPath)
	expectedOutputPath := fmt.Sprintf("%s/expOut.txt", boxPath)

	metaContent, err := os.ReadFile(metaPath)
	if err != nil {
		log.Printf("Error reading meta file: %v", err)
		return
	}

	var meta Meta
	for _, line := range strings.Split(string(metaContent), "\n") {
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
		log.Printf("Test case %d failed: %s for submission %d", testCase, *finalResult, id)
		return
	}

	diffCmd := exec.Command("diff", "-Z", "-B", outputPath, expectedOutputPath)
	if _, err := diffCmd.CombinedOutput(); err != nil {
		*finalResult = "Wrong Answer"
		return
	}

}

// CPP

type CPP struct {
}

func (p *CPP) Compile(boxId int, submission Submission) {

	code := submission.Code

	boxPath := fmt.Sprintf("/var/local/lib/isolate/%d/box/", boxId)

	cppFilePath := boxPath + "main.cpp"
	if err := os.WriteFile(cppFilePath, []byte(code), 0644); err != nil {
		log.Printf("Error writing code to file: %v", err)
		return
	}

	outputBinary := boxPath + "main"
	if _, err := exec.Command("g++", "-std=c++23", cppFilePath, "-o", outputBinary).CombinedOutput(); err != nil {
		log.Printf("Compilation error: %v", err)
		return
	}

}

func (p *CPP) Run(boxId int, submission Submission) {

	boxPath := fmt.Sprintf("/var/local/lib/isolate/%d/box/", boxId)

	var maxTime float64
	var maxRSS int
	finalResult := "Accepted"

	for i, test := range submission.Testcases {
		input := test.Input
		output := test.Output

		inputPath := boxPath + "in.txt"
		expectedOutputPath := boxPath + "expOut.txt"
		outputPath := boxPath + "out.txt"
		metaPath := boxPath + "meta.txt"

		os.WriteFile(inputPath, []byte(input), 0644)
		os.WriteFile(expectedOutputPath, []byte(output), 0644)
		os.WriteFile(outputPath, []byte(""), 0644)

		memLimit := submission.Memory * 1024
		isolateCmd := exec.Command("isolate",
			fmt.Sprintf("--box-id=%d", boxId),
			"--stdin=in.txt",
			"--stdout=out.txt",
			fmt.Sprintf("--time=%.3f", submission.Time),
			fmt.Sprintf("--wall-time=%.3f", submission.Time*1.5),
			"--fsize=1024",
			fmt.Sprintf("--mem=%d", memLimit),
			"--meta="+metaPath,
			"--run",
			"--",
			"./main",
		)
		_ = isolateCmd.Run()

		Compare(boxPath, &maxTime, &maxRSS, &finalResult, i, submission.UserId)
	}

	log.Printf("Submission %s (User %d): %s (Time: %.3fs, Memory: %dKB)",
		submission.Id, submission.UserId, finalResult, maxTime, maxRSS)

}

// Python

type Python struct {
}

func (p *Python) Compile(boxId int, submission Submission) {
	code := submission.Code
	boxPath := fmt.Sprintf("/var/local/lib/isolate/%d/box/", boxId)

	pyFilePath := boxPath + "main.py"
	if err := os.WriteFile(pyFilePath, []byte(code), 0644); err != nil {
		log.Printf("Error writing code to file: %v", err)
		return
	}
}

func (p *Python) Run(boxId int, submission Submission) {

	boxPath := fmt.Sprintf("/var/local/lib/isolate/%d/box/", boxId)

	var maxTime float64
	var maxRSS int
	finalResult := "Accepted"

	for i, test := range submission.Testcases {
		input := test.Input
		output := test.Output

		inputPath := boxPath + "in.txt"
		expectedOutputPath := boxPath + "expOut.txt"
		outputPath := boxPath + "out.txt"
		metaPath := boxPath + "meta.txt"

		os.WriteFile(inputPath, []byte(input), 0644)
		os.WriteFile(expectedOutputPath, []byte(output), 0644)
		os.WriteFile(outputPath, []byte(""), 0644)

		memLimit := submission.Memory * 1024
		isolateCmd := exec.Command("isolate",
			fmt.Sprintf("--box-id=%d", boxId),
			"--stdin=in.txt",
			"--stdout=out.txt",
			fmt.Sprintf("--time=%.3f", submission.Time),
			fmt.Sprintf("--wall-time=%.3f", submission.Time*1.5),
			"--fsize=1024",
			fmt.Sprintf("--mem=%d", memLimit),
			"--meta="+metaPath,
			"--run",
			"--",
			"/usr/bin/python3",
			"main.py",
		)
		_ = isolateCmd.Run()

		Compare(boxPath, &maxTime, &maxRSS, &finalResult, i, submission.UserId)
	}

	log.Printf("Submission %s (User %d): %s (Time: %.3fs, Memory: %dKB)",
		submission.Id, submission.UserId, finalResult, maxTime, maxRSS)

}
