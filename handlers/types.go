package handlers

type Meta struct {
	Status        string
	Message       string
	Killed        int // Present when sandbox terminated the program (time/memory limit)
	Time          float32
	Time_Wall     float32
	Max_RSS       float32
	CG_Mem        float32
	CG_OOM_Killed int
	ExitCode      int // Added - for normal exits
	ExitSig       int // Added - for signal deaths
	CSW_Voluntary int // Added - optional, for debugging
	CSW_Forced    int // Added - optional, for debugging
}
