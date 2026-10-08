package strategy

import (
	"context"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"safezone.service.worker-golang/app/pkg/cache"
	"safezone.service.worker-golang/app/pkg/logger"
	"safezone.service.worker-golang/app/schema"
)

type DBSink struct {
	DB     *sqlx.DB
	Logger *logger.ContextLogger
	cache  *cache.Cache
}

func NewDBSink(logger *logger.ContextLogger, db *sqlx.DB, cache *cache.Cache) *DBSink {
	return &DBSink{
		Logger: logger,
		DB:     db,
		cache:  cache,
	}
}

func (d *DBSink) Flush(ctx context.Context, buffer *[]schema.CovidEvent) error {
	if buffer == nil || len(*buffer) == 0 {
		return nil
	}

	// Check if the context is done before proceeding
	tx, txErr := d.DB.BeginTxx(ctx, nil)
	if txErr != nil {
		d.Logger.Error(ctx, "Failed to begin transaction", zap.Error(txErr))
		return txErr
	}

	// DEF-2: in-batch dedupe keeps the last occurrence (last-wins)
	type eventRow struct {
		date     string
		cityID   int
		regionID int
		cases    int
	}

	seen := make(map[string]bool)
	var deduped []eventRow

	// Iterate backwards so the last event for each key is selected
	for i := len(*buffer) - 1; i >= 0; i-- {
		event := (*buffer)[i]
		cityID := d.cache.GetCityID(event.Payload.City)
		regionID := d.cache.GetRegionID(cityID, event.Payload.Region)
		date := event.Payload.Date

		collisionKey := fmt.Sprintf("%s:%d:%d", date, cityID, regionID)
		if seen[collisionKey] {
			d.Logger.Warn(ctx, "Duplicate event found in buffer, keeping later occurrence",
				zap.String("date", date),
				zap.Int("city_id", cityID),
				zap.Int("region_id", regionID))
			continue
		}
		seen[collisionKey] = true
		deduped = append(deduped, eventRow{
			date:     date,
			cityID:   cityID,
			regionID: regionID,
			cases:    event.Payload.Cases,
		})
	}

	if len(deduped) == 0 {
		_ = tx.Rollback()
		*buffer = (*buffer)[:0]
		return nil
	}

	// Restore original relative order
	for i, j := 0, len(deduped)-1; i < j; i, j = i+1, j-1 {
		deduped[i], deduped[j] = deduped[j], deduped[i]
	}

	sql := "INSERT INTO covid_cases (date, city_id, region_id, cases) VALUES "
	args := make([]any, 0, len(deduped)*4)
	for i, row := range deduped {
		if i > 0 {
			sql += ","
		}
		sql += fmt.Sprintf("($%d, $%d, $%d, $%d)", i*4+1, i*4+2, i*4+3, i*4+4)
		args = append(args, row.date, row.cityID, row.regionID, row.cases)
	}
	sql += " ON CONFLICT (date, city_id, region_id) DO UPDATE SET cases=EXCLUDED.cases"

	d.Logger.Debug(ctx, "Executing buffer insert", zap.String("sql", sql), zap.Any("args", args))

	_, execErr := tx.ExecContext(ctx, sql, args...)
	if execErr != nil {
		d.Logger.Error(ctx, "Failed to execute buffer insert", zap.Error(execErr))
		_ = tx.Rollback()
		return execErr
	}

	// WG-3: clear buffer only after commit succeeds
	if commitErr := tx.Commit(); commitErr != nil {
		d.Logger.Error(ctx, "Failed to commit transaction", zap.Error(commitErr))
		return commitErr
	}

	d.Logger.Info(ctx, "DBSink flushing events",
		zap.Int("buffer_size", len(*buffer)),
		zap.Int("inserted_rows", len(deduped)),
		zap.String("event", "Events flushed"))

	*buffer = (*buffer)[:0]
	return nil
}

func (d *DBSink) Close(ctx context.Context) error {
	if d.Logger != nil {
		d.Logger.Info(ctx, "DBSink closed")
	}
	return nil
}
