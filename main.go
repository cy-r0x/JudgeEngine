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

func main() {
	manager := worker.NewManger()
	manager.With(QUEUE_NAME, CPU_COUNT)

	var wg sync.WaitGroup
	wg.Add(1)

	//dui tai blocking :"_) dk what to dooooooooooooooooooo
	go cmd.Listen()

	queue.InitQueue(manager)
	go queue.StartConsume(manager)
	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")

	wg.Wait()
}
