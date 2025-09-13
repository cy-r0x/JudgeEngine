package main

import (
	"log"
	"sync"

	"github.com/judgenot0/judge-deamon/cmd"
	"github.com/judgenot0/judge-deamon/queue"
	"github.com/judgenot0/judge-deamon/scheduler"
)

const CPU_COUNT = 8
const QUEUE_NAME = "submissionQueue"
const PORT = ":8888"

func main() {
	manager := queue.NewQueue()
	err := manager.InitQueue(QUEUE_NAME, CPU_COUNT)

	if err != nil {
		log.Println(err)
		return
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		scheduler := scheduler.NewScheduler()
		scheduler.With(CPU_COUNT)
		log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
		err := manager.StartConsume(scheduler)
		if err != nil {
			log.Println(err)
			return
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		server := cmd.NewServer(manager)
		log.Println("Server Running at " + PORT)
		server.Listen(PORT)
	}()

	wg.Wait()
}
