package worker

import (
	"fmt"
	"log"
	"os/exec"

	"github.com/judgenot0/judge-deamon/languages"
	"github.com/judgenot0/judge-deamon/structs"
	amqp "github.com/rabbitmq/amqp091-go"
)

var WorkChannel chan structs.Worker

func DoWork(w *structs.Worker, submission structs.Submission, d *amqp.Delivery) {
	defer func() {
		exec.Command("isolate", fmt.Sprintf("--box-id=%d", w.Id), "--init").Run()
		d.Ack(false)
		WorkChannel <- *w
	}()

	switch submission.Language {
	case "cpp":
		var cpp languages.CPP
		cpp.Compile(w.Id, submission)
		cpp.Run(w.Id, submission)
	case "py":
		var py languages.Python
		py.Compile(w.Id, submission)
		py.Run(w.Id, submission)
	default:
		log.Printf("Unsupported!")
	}
}
