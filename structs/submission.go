package structs

type Testcase struct {
	Input          string `json:"input" db:"input"`
	ExpectedOutput string `json:"expectedOutput" db:"expected_output"`
}

type Submission struct {
	SubmissionId       *int64     `json:"submissionId"`
	Language           string     `json:"language"`
	SourceCode         string     `json:"sourceCode"`
	Testcases          []Testcase `json:"testcases"`
	TimeLimit          float32    `json:"timeLimit"`
	MemoryLimit        float32    `json:"memoryLimit"`
	CheckerType        string     `json:"checkerType"`
	CheckerStrictSpace bool       `json:"checkerStrictSpace"`
	CheckerPrecision   *string    `json:"checkerPrecision"`
}
