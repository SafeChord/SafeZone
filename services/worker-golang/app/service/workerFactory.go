package service

import (
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"safezone.service.worker-golang/app/adapter"
	"safezone.service.worker-golang/app/config"
	cachepkg "safezone.service.worker-golang/app/pkg/cache"
	"safezone.service.worker-golang/app/pkg/logger"
	"safezone.service.worker-golang/app/schema"
	"safezone.service.worker-golang/app/strategy"
)

// NewWorker creates a production Worker wired to Kafka and PostgreSQL.
// For testing, construct Worker directly with mock Source/Sink.
func NewWorker(id int, cfg *config.Config, log *logger.ContextLogger) (*Worker, error) {
	db, err := sqlx.Connect("pgx", cfg.DBUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	cache := cachepkg.NewCache(db)
	source, err := adapter.NewKafkaSource(log, cfg.KafkaBroker, cfg.KafkaGroupID, cfg.KafkaTopic)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to create kafka source: %w", err)
	}
	return &Worker{
		DB:        db,
		Cache:     cache,
		Source:    source,
		Validator: schema.NewCovidValidator(log, cache),
		Sink:      strategy.NewDBSink(log, db, cache),
		Config:    cfg,
		Logger:    log,
		ID:        id,
	}, nil
}
