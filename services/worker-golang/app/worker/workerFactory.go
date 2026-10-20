package worker

import (
	"context"
	"fmt"
	"strconv"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"safezone.service.worker-golang/app/config"
	"safezone.service.worker-golang/app/event"
	"safezone.service.worker-golang/app/kafka"
	"safezone.service.worker-golang/app/logger"
	"safezone.service.worker-golang/app/postgres"
)

// NewWorker creates a production Worker wired to Kafka and PostgreSQL.
// For testing, construct Worker directly with mock Source/Sink.
func NewWorker(id int, cfg *config.Config, log *logger.ContextLogger) (*Worker, error) {
	db, err := sqlx.Connect("pgx", cfg.DBUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	cache, err := postgres.NewCache(db)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to load administrative-area cache: %w", err)
	}
	workerCtx := context.WithValue(context.Background(), log.WorkerIDKey, strconv.Itoa(id))
	source, err := kafka.NewSourceWithContext(workerCtx, log, cfg.KafkaBroker, cfg.KafkaGroupID, cfg.KafkaTopic)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to create kafka source: %w", err)
	}
	return &Worker{
		DB:        db,
		Cache:     cache,
		Source:    source,
		Validator: event.NewCovidValidator(log, cache),
		Sink:      postgres.NewDBSink(log, db, cache),
		Config:    cfg,
		Logger:    log,
		ID:        id,
	}, nil
}
