package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"belt-presense/internal/config"
	"belt-presense/internal/database"
	"belt-presense/internal/handler"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"gopkg.in/natefinch/lumberjack.v2"
)

func main() {
	log.Println("Starting Belt_presense Service...")
	cfg := config.LoadConfig()
	setupLogging(cfg.LogToConsole)
	logConfiguration(cfg)

	apiEndpoint := cfg.PresenseAPIEndpoint
	apiKey := cfg.PresenseAPIKey
	if cfg.UseTestURL {
		log.Println("Using test URL")
		apiEndpoint = cfg.TestAPIEndpoint
		apiKey = cfg.TestAPIKey
	}

	if apiEndpoint == "" || apiKey == "" {
		log.Fatal("FATAL: API endpoint and key must be set in .env file")
	}

	repo, err := database.NewRepository(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer repo.Close()

	processor, err := handler.NewBeltProcessor(repo, apiEndpoint, apiKey, cfg.DataSource, cfg.WriteToFile)
	if err != nil {
		log.Fatalf("Failed to initialize processor: %v", err)
	}

	mqttClient, err := handler.InitializeMQTT(cfg, processor)
	if err != nil {
		log.Fatalf("Failed to initialize MQTT client: %v", err)
	}
	defer mqttClient.Disconnect(250)

	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutdown signal received, closing consumers...")
		cancel()
	}()

	var wg sync.WaitGroup
	wg.Add(3) // MQTT, Kafka Consumer, Housekeeping

	go func() {
		defer wg.Done()
		<-ctx.Done()
		log.Println("Shutting down MQTT client...")
	}()

	go func() {
		defer wg.Done()
		runConsumer(ctx, cfg, cfg.VitalsTopic, processor.RouteVitalsMessage)
	}()

	// Start the housekeeping goroutine
	go func() {
		defer wg.Done()
		processor.RunHousekeepingCycle(ctx)
	}()

	log.Println("🚀 Service started successfully. Waiting for messages...")
	wg.Wait()
	log.Println("All services closed. Exiting.")
}

func runConsumer(ctx context.Context, cfg *config.Config, topic string, handlerFunc func([]byte)) {
	kafkaConfig := &kafka.ConfigMap{
		"bootstrap.servers": cfg.KafkaBrokers,
		"group.id":          cfg.ConsumerGroup,
		"auto.offset.reset": "earliest",
	}

	consumer, err := kafka.NewConsumer(kafkaConfig)
	if err != nil {
		log.Fatalf("Failed to create consumer for topic %s: %v", topic, err)
	}
	defer consumer.Close()

	if err := consumer.Subscribe(topic, nil); err != nil {
		log.Fatalf("Failed to subscribe to topic %s: %v", topic, err)
	}

	log.Printf("Consumer started for topic '%s' with group ID '%s'", topic, cfg.ConsumerGroup)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Stopping consumer for topic: %s", topic)
			return
		default:
			ev := consumer.Poll(100)
			if ev == nil {
				continue
			}
			switch e := ev.(type) {
			case *kafka.Message:
				handlerFunc(e.Value)
			case kafka.Error:
				fmt.Fprintf(os.Stderr, "%% Kafka Error: %v\n", e)
			}
		}
	}
}

func setupLogging(logToConsole bool) {
	logFile := &lumberjack.Logger{
		Filename:   "./logs/presense.log", // Create log in the root directory
		MaxSize:    5,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   true,
	}
	if logToConsole {
		mw := io.MultiWriter(os.Stdout, logFile)
		log.SetOutput(mw)
	} else {
		log.SetOutput(logFile)
	}
}

func logConfiguration(cfg *config.Config) {
	log.Println("--- Service Configuration ---")
	log.Printf("Kafka Brokers: %s", cfg.KafkaBrokers)
	log.Printf("MQTT Broker URL: %s", cfg.MQTTBroker)
	log.Printf("Presense API Endpoint: %s", cfg.PresenseAPIEndpoint)
	log.Printf("Data Source: %s", cfg.DataSource)
	log.Printf("DB Path: %s", cfg.DBPath)

	if cfg.PresenseAPIKey != "" {
		log.Println("Presense API Key: [SET]")
	} else {
		log.Println("Presense API Key: [NOT SET]")
	}

	if cfg.MQTTPassword != "" {
		log.Println("MQTT Password: [SET]")
	} else {
		log.Println("MQTT Password: [NOT SET]")
	}
	log.Println("---------------------------")
}
