package scheduler

import (
	"fmt"
	"log"
	"os/exec"

	"github.com/judgenot0/judge-deamon/handlers"
	"github.com/judgenot0/judge-deamon/languages"
	"github.com/judgenot0/judge-deamon/structs"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Scheduler struct {
	WorkChannel chan structs.Worker
	CPU_COUNT   int
	Handler     *handlers.Handler
}

func NewScheduler(handler *handlers.Handler) *Scheduler {
	return &Scheduler{
		Handler: handler,
	}
}

func (mngr *Scheduler) With(workerCount int) {
	mngr.WorkChannel = make(chan structs.Worker, workerCount)
	mngr.CPU_COUNT = workerCount

	for i := 0; i < workerCount; i++ {
		cmd := exec.Command("isolate", fmt.Sprintf("--box-id=%d", i), "--init")
		if err := cmd.Run(); err != nil {
			log.Printf("Error initializing sandbox for worker %d: %v", i, err)
			continue
		}

		mngr.WorkChannel <- structs.Worker{Id: i}
		log.Printf("Worker %d initialized and added to pool", i)
	}
}

func (mngr *Scheduler) Work(w structs.Worker, submission structs.Submission, d amqp.Delivery) {
	var shouldAck bool
	var shouldNack bool

	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic in scheduler.Work: %v", r)
			shouldNack = true
		}

		cmd := exec.Command("isolate", fmt.Sprintf("--box-id=%d", w.Id), "--init")
		if err := cmd.Run(); err != nil {
			log.Printf("Error cleaning up sandbox %d: %v", w.Id, err)
		}

		if shouldNack {
			d.Nack(false, true)
		} else if shouldAck {
			if err := d.Ack(false); err != nil {
				log.Printf("Error acknowledging message: %v", err)
			}
		} else {
			d.Nack(false, true)
		}

		mngr.WorkChannel <- w
	}()

	var verdict structs.Verdict
	var err error
	var runner interface {
		Compile(boxId int, runReq *structs.Submission) (structs.Verdict, error)
		Run(boxId int, runReq *structs.Submission, handler *handlers.Handler) structs.Verdict
	}

	switch submission.Language {
	case "c":
		runner = &languages.C{}
	case "cpp":
		runner = &languages.CPP{}
	case "py", "python":
		runner = &languages.Python{}
	default:
		log.Printf("Unsupported language: %s", submission.Language)
		verdict = structs.Verdict{
			Submission: &submission,
			Result:     "ce",
		}
		mngr.Handler.ProduceVerdict(&verdict)
		shouldAck = true
		return
	}

	verdict, err = runner.Compile(w.Id, &submission)
	if err != nil {
		if verdict.Result == "ce" {
			mngr.Handler.ProduceVerdict(&verdict)
			shouldAck = true
		} else {
			log.Printf("Compilation error for submission %v: %v", submission, err)
			verdict = structs.Verdict{
				Submission: &submission,
				Result:     "ce",
			}
			mngr.Handler.ProduceVerdict(&verdict)
			shouldAck = true
		}
		return
	}

	verdict = runner.Run(w.Id, &submission, mngr.Handler)
	mngr.Handler.ProduceVerdict(&verdict)
	shouldAck = true
}
