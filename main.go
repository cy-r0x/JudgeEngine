package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/judgenot0/judge-deamon/cmd"
	"github.com/judgenot0/judge-deamon/config"
	"github.com/judgenot0/judge-deamon/handlers"
	"github.com/judgenot0/judge-deamon/queue"
	"github.com/judgenot0/judge-deamon/scheduler"
)

func main() {
	config := config.GetConfig()

	queueManager := queue.NewQueue()
	err := queueManager.InitQueue(config.QueueName, config.WorkerCount, config.RabbitMQURL)
	if err != nil {
		log.Fatalf("Failed to initialize queue: %v", err)
	}

	handler := handlers.NewHandler(config)

	scheduler := scheduler.NewScheduler(handler)
	if err := scheduler.With(config.WorkerCount); err != nil {
		log.Fatalf("Failed to initialize scheduler: %v", err)
	}

	server := cmd.NewServer(config, queueManager, scheduler)
	server.RegisterMetrics()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("[*] Waiting for messages. To exit press CTRL+C")
		if err := queueManager.StartConsume(ctx, scheduler); err != nil {
			log.Printf("Queue consumer stopped: %v", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("[*] Server Running at %s", config.HttpPort)
		if err := server.Listen(ctx, config.HttpPort); err != nil {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	<-sigChan
	log.Println("\n[*] Shutting down gracefully...")

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	shutdownDone := make(chan struct{})
	go func() {
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Error shutting down server: %v", err)
		}
		close(shutdownDone)
	}()

	select {
	case <-shutdownDone:
		log.Println("[*] Server shut down successfully")
	case <-shutdownCtx.Done():
		log.Println("[*] Shutdown timeout exceeded, forcing exit")
	}

	if err := queueManager.Close(); err != nil {
		log.Printf("Error closing queue: %v", err)
	}

	wg.Wait()
	log.Println("[*] Shutdown complete")
}
