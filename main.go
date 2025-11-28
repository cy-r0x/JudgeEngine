package main

import (
	"log"
	"sync"

	"github.com/judgenot0/judge-deamon/cmd"
	"github.com/judgenot0/judge-deamon/config"
	"github.com/judgenot0/judge-deamon/handlers"
	"github.com/judgenot0/judge-deamon/queue"
	"github.com/judgenot0/judge-deamon/scheduler"
)

func main() {

	//load .env file
	config := config.GetConfig()

	//init new queue manager
	queueManager := queue.NewQueue()
	err := queueManager.InitQueue(config.QueueName, config.WorkerCount, config.RabbitMQURL)
	if err != nil {
		log.Fatalf("Failed to initialize queue: %v", err)
	}

	//init new handler
	handler := handlers.NewHandler(config)

	//init scheduler
	scheduler := scheduler.NewScheduler(handler)
	scheduler.With(config.WorkerCount)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("[*] Waiting for messages. To exit press CTRL+C")
		err := queueManager.StartConsume(scheduler)
		if err != nil {
			log.Println(err)
			return
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		server := cmd.NewServer(config, queueManager, scheduler)
		server.RegisterMetrics()
		log.Println("[*] Server Running at " + config.HttpPort)
		server.Listen(config.HttpPort)
	}()

	wg.Wait()
}
