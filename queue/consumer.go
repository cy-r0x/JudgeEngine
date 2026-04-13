package queue

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/judgenot0/judge-deamon/scheduler"
	"github.com/judgenot0/judge-deamon/structs"
	amqp "github.com/rabbitmq/amqp091-go"
)

func (q *Queue) StartConsume(ctx context.Context, scheduler *scheduler.Scheduler) error {
	q.ctx = ctx
	for {
		ch, conn := q.getChannel()
		if ch == nil || ch.IsClosed() || conn == nil || conn.IsClosed() {
			if err := q.reconnect(); err != nil {
				log.Printf("Failed to reconnect, retrying: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}
			ch, _ = q.getChannel()
		}

		var err error
		q.msgs, err = ch.Consume(q.queueName, "", false, false, false, false, nil)
		if err != nil {
			log.Printf("Failed to start consuming: %v, attempting reconnect", err)
			time.Sleep(5 * time.Second)
			if reconnectErr := q.reconnect(); reconnectErr != nil {
				log.Printf("Reconnection failed: %v", reconnectErr)
				time.Sleep(5 * time.Second)
			}
			continue
		}

		log.Println("[*] Started consuming messages from queue")

	messageLoop:
		for {
			select {
			case <-ctx.Done():
				log.Println("Context cancelled, stopping consumer loop")
				return nil
			case d, ok := <-q.msgs:
				if !ok {
					log.Println("Message channel closed")
					break messageLoop
				}

				select {
				case <-ctx.Done():
					log.Println("Context cancelled, nacking message to DLQ and stopping")
					d.Nack(false, false)
					return nil

				case worker := <-scheduler.WorkChannel:
					var submission structs.Submission
					err := json.Unmarshal(d.Body, &submission)
					if err != nil {
						log.Printf("Raw body: %s", string(d.Body))
						log.Printf("Invalid message body: %v", err)
						d.Nack(false, false)
						scheduler.WorkChannel <- worker
						continue
					}

					go func(delivery amqp.Delivery, w structs.Worker, sub *structs.Submission) {
						scheduler.Work(ctx, w, sub, delivery)
					}(d, worker, &submission)

				case <-time.After(5 * time.Minute):
					log.Println("Warning: No workers available for 5 minutes, message sent to DLQ")
					d.Nack(false, false)
				}
			}
		}

		log.Println("Message channel closed, attempting to reconnect...")
		time.Sleep(5 * time.Second)
	}
}
