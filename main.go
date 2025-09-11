package main

import (
	"log"
	"sync"

	"github.com/judgenot0/judge-deamon/cmd"
	"github.com/judgenot0/judge-deamon/queue"
	"github.com/judgenot0/judge-deamon/worker"
)

const CPU_COUNT = 8
const QUEUE_NAME = "submissionQueue"
const PORT = ":8888"

func main() {
	manager := worker.NewManger()
	manager.With(QUEUE_NAME, CPU_COUNT)

	queue_manager := queue.NewQueue()
	err := queue_manager.InitQueue(QUEUE_NAME, CPU_COUNT)

	if err != nil {
		log.Println(err)
		return
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		cmd.Listen(PORT, queue_manager)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := queue_manager.StartConsume(manager)
		if err != nil {
			log.Println(err)
		}
	}()

	log.Println("Server Running at " + PORT)
	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")

	wg.Wait()
}
