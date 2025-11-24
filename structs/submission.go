package structs

type Testcase struct {
	Input          string `json:"input" db:"input"`
	ExpectedOutput string `json:"expected_output" db:"expected_output"`
}

type Submission struct {
	SubmissionId       *int64     `json:"submission_id"`
	Language           string     `json:"language"`
	SourceCode         string     `json:"source_code"`
	Testcases          []Testcase `json:"testcases"`
	Timelimit          float32    `json:"time_limit"`
	MemoryLimit        float32    `json:"memory_limit"`
	CheckerType        string     `json:"checker_type"`
	CheckerStrictSpace bool       `json:"checker_strict_space"`
	CheckerPrecision   *string    `json:"checker_precision"`
}
