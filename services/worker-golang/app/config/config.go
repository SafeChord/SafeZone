package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	Environment string
	LogLevel    zapcore.Level
	// service specific configuration
	ServiceName    string
	ServiceVersion string
	// batch processing configuration
	BatchSize         int
	FlushInterval     time.Duration
	LivenessWindow    time.Duration
	DependencyTimeout time.Duration
	ServerIP          string
	ServerPort        int
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
	logLevel := getLogLevelEnv("LOG_LEVEL", zapcore.InfoLevel)
	serviceName := getEnv("SERVICE_NAME", "worker-golang")
	serviceVersion := getEnv("SERVICE_VERSION", "0.1.0")
	// batch processing configuration
	batchSize := getNumEnv("BATCH_SIZE", 5)
	flushInterval := getDurationEnv("FLUSH_INTERVAL", 1*time.Second)
	livenessWindow := getDurationEnv("LIVENESS_WINDOW", 30*time.Second)
	dependencyTimeout := getDurationEnv("DEPENDENCY_TIMEOUT", 10*time.Second)
	if dependencyTimeout <= 0 {
		fmt.Fprintln(os.Stderr, "Invalid duration environment variable DEPENDENCY_TIMEOUT: must be positive, using default value 10s")
		dependencyTimeout = 10 * time.Second
	}
	if dependencyTimeout >= livenessWindow {
		return nil, fmt.Errorf("DEPENDENCY_TIMEOUT (%s) must be less than LIVENESS_WINDOW (%s)", dependencyTimeout, livenessWindow)
	}
	serverIP := getEnv("SERVER_IP", "0.0.0.0")
	serverPort := getNumEnv("SERVER_PORT", 8080)
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
		Environment:       environment,
		LogLevel:          logLevel,
		ServiceName:       serviceName,
		ServiceVersion:    serviceVersion,
		BatchSize:         batchSize,
		FlushInterval:     flushInterval,
		LivenessWindow:    livenessWindow,
		DependencyTimeout: dependencyTimeout,
		ServerIP:          serverIP,
		ServerPort:        serverPort,
		WorkerCount:       workerCount,
		ParallelN:         parallelN,
		KafkaBroker:       kafkaBroker,
		KafkaGroupID:      kafkaGroupID,
		KafkaTopic:        kafkaTopic,
		DBUrl:             dbUrl,
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

func getLogLevelEnv(key string, defaultVal zapcore.Level) zapcore.Level {
	valStr := getEnv(key, defaultVal.String())
	// ParseLevel ignores case, and also accepts dpanic, panic and fatal; the worker accepts debug through error only.
	val, err := zapcore.ParseLevel(valStr)
	if err == nil && val <= zapcore.ErrorLevel {
		return val
	}
	fmt.Fprintf(os.Stderr, "Unrecognized log level environment variable %s: %q, using default value %s\n", key, valStr, defaultVal)
	return defaultVal
}
