package scheduler

import (
	"context"
	"fmt"
	"log"
	"os/exec"

	"github.com/judgenot0/judge-deamon/handlers"
	"github.com/judgenot0/judge-deamon/languages"
	"github.com/judgenot0/judge-deamon/structs"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Runner interface {
	Compile(ctx context.Context, boxId int, runReq *structs.Submission) (structs.Verdict, error)
	Run(ctx context.Context, boxId int, runReq *structs.Submission, handler *handlers.Handler) structs.Verdict
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
	case "js", "javascript", "node", "nodejs":
		return &languages.NodeJS{}
	default:
		return nil
	}
}

func (mngr *Scheduler) With(workerCount int) error {
	mngr.WorkChannel = make(chan structs.Worker, workerCount)
	mngr.WorkerCount = workerCount

	initialized := 0
	for i := 0; i < workerCount; i++ {
		cmd := exec.Command("isolate", fmt.Sprintf("--box-id=%d", i), "--cg", "--init")
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

func (mngr *Scheduler) Work(ctx context.Context, w structs.Worker, submission *structs.Submission, d amqp.Delivery) {
	// if true we need to ack the message queue
	ackStatus := true

	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic in scheduler.Work: %v", r)
			ackStatus = false
		}

		cmd := exec.Command("isolate", fmt.Sprintf("--box-id=%d", w.Id), "--cg", "--cleanup")
		if err := cmd.Run(); err != nil {
			log.Printf("Error cleaning up sandbox %d: %v", w.Id, err)
		}

		if !ackStatus {
			if err := d.Nack(false, true); err != nil {
				log.Printf("Error nacking message: %v", err)
			}
		} else {
			if err := d.Ack(false); err != nil {
				log.Printf("Error acknowledging message: %v", err)
			}
		}

		mngr.WorkChannel <- w
	}()

	mngr.processWork(ctx, w, submission, &ackStatus)
}

func (mngr *Scheduler) processWork(ctx context.Context, w structs.Worker, submission *structs.Submission, ackStatus *bool) {

	verdict := structs.Verdict{
		Submission: submission,
		Result:     "ac",
		MaxTime:    nil,
		MaxRSS:     nil,
	}

	defer func() {
		mngr.Handler.ProduceVerdict(&verdict, ackStatus)
	}()

	if submission.Language == "" {
		log.Printf("Missing language in submission")
		verdict.Result = "ce"
		return
	}

	if submission.SourceCode == "" {
		log.Printf("Missing source code in submission")
		verdict.Result = "ce"
		return
	}

	runner := GetRunner(submission.Language)
	if runner == nil {
		log.Printf("Unsupported language: %s", submission.Language)
		verdict.Result = "ce"
		return
	}

	var err error
	verdict, err = runner.Compile(ctx, w.Id, submission)
	if err != nil {
		log.Printf("Compilation error for submission %d: %v", getSubmissionID(submission), err)
		verdict.Result = "ce"
		return
	}

	verdict = runner.Run(ctx, w.Id, submission, mngr.Handler)
}

func getSubmissionID(submission *structs.Submission) int64 {
	if submission.SubmissionId != nil {
		return *submission.SubmissionId
	}
	return 0
}
