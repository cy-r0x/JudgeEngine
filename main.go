package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"sync"

	"github.com/judgenot0/judge-deamon/handlers"
	"github.com/judgenot0/judge-deamon/structs"
	"github.com/judgenot0/judge-deamon/worker"
	amqp "github.com/rabbitmq/amqp091-go"
)

var msgs <-chan amqp.Delivery
var conn *amqp.Connection
var ch *amqp.Channel

func init() {
	worker.WorkChannel = make(chan structs.Worker, 4)
	for i := 0; i < 4; i++ {
		worker.WorkChannel <- structs.Worker{Id: i}
		exec.Command("isolate", fmt.Sprintf("--box-id=%d", i), "--init").Run()
	}

	var err error
	conn, err = amqp.Dial("amqp://guest:guest@127.0.0.1:5672/")
	handlers.FailOnError(err, "Failed to connect to RabbitMQ")

	ch, err = conn.Channel()
	handlers.FailOnError(err, "Failed to open a channel")

	err = ch.Qos(4, 0, false)
	handlers.FailOnError(err, "Failed to set QoS")

	args := amqp.Table{
		"x-queue-type": "quorum",
	}

	queueName := "msgQueue"
	q, err := ch.QueueDeclare(queueName, true, false, false, false, args)
	handlers.FailOnError(err, "Failed to declare a queue")

	msgs, err = ch.Consume(q.Name, "", false, false, false, false, nil)
	handlers.FailOnError(err, "Failed to register a consumer")
}

func main() {
	defer func() {
		if ch != nil {
			ch.Close()
		}
		if conn != nil {
			conn.Close()
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		for d := range msgs {
			slave := <-worker.WorkChannel
			var submission structs.Submission
			if err := json.Unmarshal(d.Body, &submission); err != nil {
				log.Printf("Raw body: %s", string(d.Body))
				log.Printf("Invalid message body: %v", err)
				continue
			}
			dCopy := d
			go worker.DoWork(&slave, submission, &dCopy)
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	wg.Wait()
}
