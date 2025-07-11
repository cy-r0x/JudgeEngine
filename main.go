package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Testcase struct {
	Input  string `json:"input"`
	Output string `json:"output"`
}

type Meta struct {
	Status    string
	Message   string
	Killed    int
	Time      float64
	Time_Wall float64
	Max_RSS   int
}

type Submission struct {
	Id        string     `json:"id"`
	Language  string     `json:"language"`
	Time      float32    `json:"timeLimit"`
	Memory    int        `json:"memoryLimit"`
	Code      string     `json:"code"`
	Testcases []Testcase `json:"testCases"`
	UserId    int        `json:"userId"`
}

type Worker struct {
	Id     int
	Status bool
}

var workChannel chan Worker

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func doWork(w *Worker, submission Submission, d *amqp.Delivery) {
	defer func() {
		d.Ack(false)
		workChannel <- *w
	}()

	switch submission.Language {
	case "cpp":
		var cpp CPP
		cpp.Compile(w.Id, submission)
		cpp.Run(w.Id, submission)
	case "py":
		var py Python
		py.Compile(w.Id, submission)
		py.Run(w.Id, submission)
	default:
		log.Printf("Unsupported!")
	}

}

func main() {
	workChannel = make(chan Worker, 4)
	for i := 0; i < 4; i++ {
		workChannel <- Worker{Id: i, Status: true}
		if err := exec.Command("isolate", fmt.Sprintf("--box-id=%d", i), "--init").Run(); err != nil {
			log.Printf("Isolate init error: %v", err)
			return
		}
	}

	conn, err := amqp.Dial("amqp://guest:guest@127.0.0.1:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	ch.Qos(4, 0, false)
	args := amqp.Table{
		"x-queue-type": "quorum",
	}

	queueName := "msgQueue"
	q, err := ch.QueueDeclare(queueName, true, false, false, false, args)
	failOnError(err, "Failed to declare a queue")

	msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
	failOnError(err, "Failed to register a consumer")

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		
		for d := range msgs {
			worker := <-workChannel
			var submission Submission
			if err := json.Unmarshal(d.Body, &submission); err != nil {
				log.Printf("Raw body: %s", string(d.Body))
				log.Printf("Invalid message body: %v", err)
				continue
			}
			dCopy := d
			go doWork(&worker, submission, &dCopy)
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	wg.Wait()
}
