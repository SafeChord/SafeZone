package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Environment string
	LogLevel    string
	// service specific configuration
	ServiceName    string
	ServiceVersion string
	// batch processing configuration
	BatchSize     int
	FlushInterval time.Duration
	// parallel processing configuration
	WorkerCount int // number of workers
	ParallelN   int // number of parallel workers at a time
	// Kafka configuration
	KafkaBroker  string
	KafkaGroupID string
	KafkaTopic   string
	// database configuration
	DBUrl string
}

func Load() (*Config, error) {
	err := godotenv.Load()
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	// basic configuration
	environment := getEnv("ENVIRONMENT", "TEST")
	logLevel := getEnv("LOG_LEVEL", "info")
	serviceName := getEnv("SERVICE_NAME", "worker-golang")
	serviceVersion := getEnv("SERVICE_VERSION", "0.1.0")
	// batch processing configuration
	batchSize := getNumEnv("BATCH_SIZE", 5)
	flushInterval := getDurationEnv("FLUSH_INTERVAL", 1*time.Second)
	// parallel processing configuration
	workerCount := getNumEnv("WORKER_COUNT", 1)
	parallelN := getNumEnv("PARALLEL_N", 1)
	// Kafka configuration
	kafkaBroker := getEnv("KAFKA_BROKER", "localhost:9092")
	kafkaGroupID := getEnv("KAFKA_GROUP_ID", "worker-group")
	kafkaTopic := getEnv("KAFKA_TOPIC", "covid-events")
	// database configuration
	dbUrl := getEnv("DB_URL", "postgres://postgres:password@localhost:5432/safezone")

	return &Config{
		Environment:    environment,
		LogLevel:       logLevel,
		ServiceName:    serviceName,
		ServiceVersion: serviceVersion,
		BatchSize:      batchSize,
		FlushInterval:  flushInterval,
		WorkerCount:    workerCount,
		ParallelN:      parallelN,
		KafkaBroker:    kafkaBroker,
		KafkaGroupID:   kafkaGroupID,
		KafkaTopic:     kafkaTopic,
		DBUrl:          dbUrl,
	}, nil
}

func getEnv(key string, defaultVal string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		fmt.Fprintf(os.Stderr, "Environment variable %s not set, using default value: %s\n", key, defaultVal)
		return defaultVal
	}
	return val
}

func getNumEnv(key string, defaultVal int) int {
	valStr := getEnv(key, strconv.Itoa(defaultVal))
	val, err := strconv.Atoi(valStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse environment variable %s: %v, using default value %d\n", key, err, defaultVal)
		return defaultVal
	}
	return val
}

func getDurationEnv(key string, defaultVal time.Duration) time.Duration {
	valStr := getEnv(key, defaultVal.String())
	val, err := time.ParseDuration(valStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse duration environment variable %s: %v, using default value %s\n", key, err, defaultVal)
		return defaultVal
	}
	return val
}
