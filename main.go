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
	scheduler := scheduler.NewScheduler()
	scheduler.With(CPU_COUNT)

	manager := queue.NewQueue()
	err := manager.InitQueue(QUEUE_NAME, CPU_COUNT)

	server := cmd.NewServer()

	if err != nil {
		log.Println(err)
		return
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		server.Listen(PORT, manager)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := manager.StartConsume(scheduler)
		if err != nil {
			log.Println(err)
			return
		}
	}()

	log.Println("Server Running at " + PORT)
	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")

	wg.Wait()
}
