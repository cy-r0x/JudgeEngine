package structs

type Verdict struct {
	Submission *Submission
	Result     string
	MaxTime    *float32
	MaxRSS     *float32
}
