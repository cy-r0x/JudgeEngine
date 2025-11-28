package config

import (
	"log"
	"os"
	"strconv"
	"sync"

	env "github.com/joho/godotenv"
)

type Config struct {
	WorkerCount    int
	QueueName      string
	RabbitMQURL    string
	HttpPort       string
	EngineKey      string
	ServerEndpoint string
}

var (
	instance *Config
	once     sync.Once
)

func loadConfig() Config {
	err := env.Load()
	if err != nil {
		log.Fatalln(".env not found")
	}

	var config Config

	workerCountStr := os.Getenv("WORKER_COUNT")
	workerCount, err := strconv.Atoi(workerCountStr)
	if err != nil {
		log.Println("Invalid WORKER_COUNT, using default value 1")
		workerCount = 1
	}
	config.WorkerCount = workerCount

	config.QueueName = os.Getenv("QUEUE_NAME")
	if config.QueueName == "" {
		config.QueueName = "judge_queue"
		log.Println("QUEUE_NAME not set, using default: judge_queue")
	}

	config.RabbitMQURL = os.Getenv("RABBITMQ_URL")
	if config.RabbitMQURL == "" {
		config.RabbitMQURL = "amqp://guest:guest@localhost:5672/"
		log.Println("RABBITMQ_URL not set, using default: amqp://guest:guest@localhost:5672/")
	}

	config.HttpPort = os.Getenv("HTTP_PORT")
	if config.HttpPort == "" {
		config.HttpPort = "8080"
		log.Println("HTTP_PORT not set, using default: 8080")
	}

	config.EngineKey = os.Getenv("ENGINE_KEY")
	if config.EngineKey == "" {
		log.Fatalln("ENGINE_KEY not set") // same as log.println and then exit
	}

	config.ServerEndpoint = os.Getenv("SERVER_ENDPOINT")
	if config.ServerEndpoint == "" {
		log.Fatalln("SERVER_ENDPOINT not set")
	}

	return config
}

func GetConfig() *Config {
	once.Do(func() {
		config := loadConfig()
		instance = &config
	})
	return instance
}
