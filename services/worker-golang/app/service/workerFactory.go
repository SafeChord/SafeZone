package service

import (
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
func NewWorker(id int, cfg *config.Config, log *logger.ContextLogger) *Worker {
	db := sqlx.MustConnect("pgx", cfg.DBUrl)
	cache := cachepkg.NewCache(db)
	return &Worker{
		DB:        db,
		Cache:     cache,
		Source:    adapter.NewKafkaSource(log, cfg.KafkaBroker, cfg.KafkaGroupID, cfg.KafkaTopic),
		Validator: schema.NewCovidValidator(log, cache),
		Sink:      strategy.NewDBSink(log, db, cache),
		Config:    cfg,
		Logger:    log,
		ID:        id,
	}
}
