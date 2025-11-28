package queue

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/judgenot0/judge-deamon/config"
	"github.com/judgenot0/judge-deamon/scheduler"
	"github.com/judgenot0/judge-deamon/structs"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Queue struct {
	msgs        <-chan amqp.Delivery
	conn        *amqp.Connection
	ch          *amqp.Channel
	queueName   string
	rabbitmqURL string
	workerCount int
}

func NewQueue() *Queue {
	return &Queue{}
}

func (q *Queue) InitQueue(config *config.Config) error {
	q.queueName = config.QueueName
	q.rabbitmqURL = config.RabbitMQURL
	q.workerCount = config.WorkerCount

	return q.connect()
}

func (q *Queue) connect() error {
	var err error
	q.conn, err = amqp.Dial(q.rabbitmqURL)
	if err != nil {
		log.Printf("Failed to connect to RabbitMQ: %v", err)
		return err
	}

	q.ch, err = q.conn.Channel()
	if err != nil {
		log.Printf("Failed to open channel: %v", err)
		if q.conn != nil {
			q.conn.Close()
		}
		return err
	}

	err = q.ch.Qos(q.workerCount, 0, false)
	if err != nil {
		log.Printf("Failed to set QoS: %v", err)
		q.ch.Close()
		q.conn.Close()
		return err
	}

	args := amqp.Table{
		"x-queue-type": "quorum",
	}
	_, err = q.ch.QueueDeclare(q.queueName, true, false, false, false, args)
	if err != nil {
		log.Printf("Failed to declare queue: %v", err)
		q.ch.Close()
		q.conn.Close()
		return err
	}

	return nil
}

func (q *Queue) reconnect() error {
	log.Println("Attempting to reconnect to RabbitMQ...")

	if q.ch != nil {
		q.ch.Close()
	}
	if q.conn != nil {
		q.conn.Close()
	}

	backoff := time.Second
	maxBackoff := 30 * time.Second

	for {
		err := q.connect()
		if err == nil {
			log.Println("Successfully reconnected to RabbitMQ")
			return nil
		}

		log.Printf("Reconnection failed, retrying in %v: %v", backoff, err)
		time.Sleep(backoff)

		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

func (q *Queue) QueueMessage(submission []byte) error {
	if q.ch == nil || q.ch.IsClosed() {
		if err := q.reconnect(); err != nil {
			return err
		}
	}

	err := q.ch.Publish(
		"",
		q.queueName,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        submission,
		},
	)

	if err != nil {
		log.Printf("Failed to publish message, attempting reconnect: %v", err)
		if reconnectErr := q.reconnect(); reconnectErr != nil {
			return reconnectErr
		}
		err = q.ch.Publish(
			"",
			q.queueName,
			false,
			false,
			amqp.Publishing{
				ContentType: "application/json",
				Body:        submission,
			},
		)
	}

	return err
}

func (q *Queue) StartConsume(ctx context.Context, scheduler *scheduler.Scheduler) error {
	for {
		select {
		case <-ctx.Done():
			log.Println("Context cancelled, stopping consumer")
			return nil
		default:
		}

		if q.ch == nil || q.ch.IsClosed() || q.conn == nil || q.conn.IsClosed() {
			if err := q.reconnect(); err != nil {
				log.Printf("Failed to reconnect, retrying: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}
		}

		var err error
		q.msgs, err = q.ch.Consume(q.queueName, "", false, false, false, false, nil)
		if err != nil {
			log.Printf("Failed to start consuming: %v, attempting reconnect", err)
			time.Sleep(5 * time.Second)
			if reconnectErr := q.reconnect(); reconnectErr != nil {
				log.Printf("Reconnection failed: %v", reconnectErr)
				time.Sleep(5 * time.Second)
			}
			continue
		}

		log.Println("Started consuming messages from queue")

		for d := range q.msgs {
			select {
			case <-ctx.Done():
				log.Println("Context cancelled, stopping consumer loop")
				return nil
			case worker := <-scheduler.WorkChannel:
				var submission structs.Submission
				err := json.Unmarshal(d.Body, &submission)
				if err != nil {
					log.Printf("Raw body: %s", string(d.Body))
					log.Printf("Invalid message body: %v", err)
					d.Nack(false, false)
					continue
				}

				go func(delivery amqp.Delivery, w structs.Worker, sub structs.Submission) {
					defer func() {
						if r := recover(); r != nil {
							log.Printf("Panic in scheduler.Work: %v", r)
							delivery.Nack(false, true)
							scheduler.WorkChannel <- w
						}
					}()
					scheduler.Work(w, sub, delivery)
				}(d, worker, submission)

			case <-time.After(5 * time.Minute):
				log.Println("Warning: No workers available for 5 minutes, message will be redelivered")
				d.Nack(false, true)
			}
		}

		log.Println("Message channel closed, attempting to reconnect...")
		time.Sleep(5 * time.Second)
	}
}

func (q *Queue) Close() error {
	var errs []error
	if q.ch != nil {
		if err := q.ch.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if q.conn != nil {
		if err := q.conn.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}
