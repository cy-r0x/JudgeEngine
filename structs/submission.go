package structs

type Submission struct {
	Id        string     `json:"id"`
	Language  string     `json:"language"`
	Time      float32    `json:"timeLimit"`
	Memory    int        `json:"memoryLimit"`
	Code      string     `json:"code"`
	Testcases []Testcase `json:"testCases"`
	UserId    int        `json:"userId"`
}
