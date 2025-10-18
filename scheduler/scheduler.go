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
	defer func() {
		exec.Command("isolate", fmt.Sprintf("--box-id=%d", w.Id), "--init").Run()
		d.Ack(false)
		mngr.WorkChannel <- w
	}()

	var verdict structs.Verdict
	var err error

	switch submission.Language {
	case "cpp":
		var cpp languages.CPP
		verdict, err = cpp.Compile(w.Id, &submission)
		if err != nil {
			if verdict.Result == "ce" {
				mngr.Handler.ProduceVerdict(&verdict)
			} else {
				log.Println(err)
				return
			}
		} else {
			verdict = cpp.Run(w.Id, &submission, mngr.Handler)
			mngr.Handler.ProduceVerdict(&verdict)
		}
	case "python":
		var py languages.Python
		verdict, err = py.Compile(w.Id, &submission)
		if err == nil {
			verdict = py.Run(w.Id, &submission, mngr.Handler)
			mngr.Handler.ProduceVerdict(&verdict)
		}
	default:
		log.Printf("Unsupported!")
	}
}
