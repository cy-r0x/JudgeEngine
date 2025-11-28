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

type Runner interface {
	Compile(boxId int, runReq *structs.Submission) (structs.Verdict, error)
	Run(boxId int, runReq *structs.Submission, handler *handlers.Handler) structs.Verdict
}

type Scheduler struct {
	WorkChannel chan structs.Worker
	WorkerCount int
	Handler     *handlers.Handler
}

func NewScheduler(handler *handlers.Handler) *Scheduler {
	return &Scheduler{
		Handler: handler,
	}
}

func GetRunner(language string) Runner {
	switch language {
	case "c":
		return &languages.C{}
	case "cpp":
		return &languages.CPP{}
	case "py":
		return &languages.Python{}
	default:
		return nil
	}
}

func (mngr *Scheduler) With(workerCount int) error {
	mngr.WorkChannel = make(chan structs.Worker, workerCount)
	mngr.WorkerCount = workerCount

	initialized := 0
	for i := 0; i < workerCount; i++ {
		cmd := exec.Command("isolate", fmt.Sprintf("--box-id=%d", i), "--init")
		if err := cmd.Run(); err != nil {
			log.Printf("Error initializing sandbox for worker %d: %v", i, err)
			continue
		}

		mngr.WorkChannel <- structs.Worker{Id: i}
		initialized++
		log.Printf("Worker %d initialized and added to pool", i)
	}

	if initialized == 0 {
		return fmt.Errorf("failed to initialize any workers")
	}

	if initialized < workerCount {
		log.Printf("Warning: Only %d out of %d workers initialized", initialized, workerCount)
	}

	return nil
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
			if err := d.Nack(false, true); err != nil {
				log.Printf("Error nacking message: %v", err)
			}
		} else if shouldAck {
			if err := d.Ack(false); err != nil {
				log.Printf("Error acknowledging message: %v", err)
			}
		} else {
			log.Printf("Warning: Neither ack nor nack set, defaulting to nack")
			if err := d.Nack(false, true); err != nil {
				log.Printf("Error nacking message: %v", err)
			}
		}

		mngr.WorkChannel <- w
	}()

	if submission.Language == "" {
		log.Printf("Missing language in submission")
		verdict := structs.Verdict{
			Submission: &submission,
			Result:     "ce",
		}
		mngr.Handler.ProduceVerdict(&verdict)
		shouldAck = true
		return
	}

	if submission.SourceCode == "" {
		log.Printf("Missing source code in submission")
		verdict := structs.Verdict{
			Submission: &submission,
			Result:     "ce",
		}
		mngr.Handler.ProduceVerdict(&verdict)
		shouldAck = true
		return
	}

	runner := GetRunner(submission.Language)
	if runner == nil {
		log.Printf("Unsupported language: %s", submission.Language)
		verdict := structs.Verdict{
			Submission: &submission,
			Result:     "ce",
		}
		mngr.Handler.ProduceVerdict(&verdict)
		shouldAck = true
		return
	}

	verdict, err := runner.Compile(w.Id, &submission)
	if err != nil {
		if verdict.Result == "ce" {
			mngr.Handler.ProduceVerdict(&verdict)
			shouldAck = true
		} else {
			log.Printf("Compilation error for submission %d: %v", getSubmissionID(submission), err)
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

func getSubmissionID(submission structs.Submission) int64 {
	if submission.SubmissionId != nil {
		return *submission.SubmissionId
	}
	return 0
}
