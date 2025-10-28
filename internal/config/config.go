package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	KafkaBrokers string
	VitalsTopic  string
	// BPSPO2Topic      string // REMOVED
	ConsumerGroup       string
	PresenseAPIKey      string
	PresenseAPIEndpoint string
	TestAPIEndpoint     string
	TestAPIKey          string
	DataSource          string
	DBPath              string
	WriteToFile         bool
	LogToConsole        bool
	UseTestURL          bool
	MQTTBroker          string
	MQTTClientID        string
	MQTTUsername        string
	MQTTPassword        string
}

func LoadConfig() *Config {
	err := godotenv.Load() // Looks for ".env" in the current directory
	if err != nil {
		log.Println("No .env file found, using environment variables or default values")
	}

	return &Config{
		KafkaBrokers: getEnv("KAFKA_BROKERS", "localhost:9092"),
		VitalsTopic:  getEnv("VITALS_TOPIC", "patient-vitals-data-topic"),
		// BPSPO2Topic:      getEnv("BPSPO2_TOPIC", "patient-bpspo2-data-topic"), // REMOVED
		ConsumerGroup:       getEnv("CONSUMER_GROUP", "belt_presense"),
		PresenseAPIEndpoint: getEnv("PRESENSE_API_ENDPOINT", "https://vitals.presense.icu/data"),
		PresenseAPIKey:      getEnv("PRESENSE_API_KEY", "PT1YzV4wGK1OsZSmGDMBIaYC7muCIu6f3Dnl4qOO"),
		TestAPIEndpoint:     getEnv("TEST_API_ENDPOINT", "https://staging-vitals.presense.icu/data"),
		TestAPIKey:          getEnv("TEST_API_KEY", "814xqfGJWSVfKgfFOvQz24MLAuRsuDA3"),
		DataSource:          getEnv("DATA_SOURCE", "DefaultSource"),
		DBPath:              getEnv("DB_PATH", "presense.db"),
		WriteToFile:         strings.EqualFold(getEnv("WRITE_TO_FILE", "false"), "true"),
		LogToConsole:        strings.EqualFold(getEnv("LOG_TO_CONSOLE", "false"), "true"),
		UseTestURL:          strings.EqualFold(getEnv("USE_TEST_URL", "false"), "true"),
		MQTTBroker:          getEnv("MQTT_BROKER_URL", "tcp://localhost:1883"),
		MQTTClientID:        getEnv("MQTT_CLIENT_ID", "MqttCallService_local"),
		MQTTUsername:        getEnv("MQTT_USERNAME", ""),
		MQTTPassword:        getEnv("MQTT_PASSWORD", ""),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
