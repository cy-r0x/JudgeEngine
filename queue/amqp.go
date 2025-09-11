package queue

import (
	"encoding/json"
	"log"

	"github.com/judgenot0/judge-deamon/handlers"
	structs "github.com/judgenot0/judge-deamon/structs"
	"github.com/judgenot0/judge-deamon/worker"
	amqp "github.com/rabbitmq/amqp091-go"
)

var msgs <-chan amqp.Delivery
var conn *amqp.Connection
var ch *amqp.Channel

func InitQueue(manager *worker.Manager) {
	var err error
	conn, err = amqp.Dial("amqp://guest:guest@127.0.0.1:5672/")
	handlers.FailOnError(err, "Failed to connect to RabbitMQ")

	ch, err = conn.Channel()
	handlers.FailOnError(err, "Failed to open a channel")

	err = ch.Qos(manager.CPU_COUNT, 0, false)
	handlers.FailOnError(err, "Failed to set QoS")

	args := amqp.Table{
		"x-queue-type": "quorum",
	}

	_, err = ch.QueueDeclare(manager.QueueName, true, false, false, false, args)
	handlers.FailOnError(err, "Failed to declare a queue")
}

func QueueMessage(submission []byte) {

}

func StartConsume(manager *worker.Manager) {
	defer func() {
		if ch != nil {
			ch.Close()
		}
		if conn != nil {
			conn.Close()
		}
	}()

	msgs, err := ch.Consume(manager.QueueName, "", false, false, false, false, nil)
	handlers.FailOnError(err, "Failed to register a consumer")

	for d := range msgs {
		slave := <-manager.WorkChannel
		var submission structs.Submission
		err := json.Unmarshal(d.Body, &submission)

		if err != nil {
			log.Printf("Raw body: %s", string(d.Body))
			log.Printf("Invalid message body: %v", err)
			continue
		}
		dCopy := d

		go manager.Work(&slave, submission, &dCopy)
	}

}
