package event

import (
	"context"
	"math"
	"time"

	"go.uber.org/zap"
	"safezone.service.worker-golang/app/logger"
)

// CacheReader is the read-only interface CovidValidator needs for city/region lookups.
// *postgres.Cache satisfies this interface.
type CacheReader interface {
	GetCityID(city string) int
	GetRegionID(cityID int, region string) int
}

type CovidValidator struct {
	Logger *logger.ContextLogger
	cache  CacheReader
}

func NewCovidValidator(logger *logger.ContextLogger, cache CacheReader) *CovidValidator {
	return &CovidValidator{
		Logger: logger,
		cache:  cache,
	}
}

func (v *CovidValidator) Validate(ctx context.Context, event CovidEvent) bool {
	// The case table stores DATE. Parse the calendar date as well as its shape; Go accepts
	// year 0000, but PostgreSQL DATE does not (and the event contract requires a real date).
	date, err := time.Parse(time.DateOnly, event.Payload.Date)
	if err != nil || date.Year() == 0 {
		v.Logger.Warn(ctx, "Invalid date", zap.String("date", event.Payload.Date), zap.String("trace_id", event.TraceID))
		return false
	}
	// city and region must exist in cache

	cityID := v.cache.GetCityID(event.Payload.City)
	regionID := v.cache.GetRegionID(cityID, event.Payload.Region)

	if cityID < 0 || regionID < 0 {
		v.Logger.Warn(ctx, "City or region not found in cache",
			zap.String("city", event.Payload.City),
			zap.String("region", event.Payload.Region),
			zap.String("trace_id", event.TraceID))
		return false
	}
	// Cases must fit the case table's INTEGER column. Zero remains accepted by the consumer,
	// although the event contract's producer-side minimum is one.
	if event.Payload.Cases < 0 || event.Payload.Cases > math.MaxInt32 {
		v.Logger.Warn(ctx, "Invalid cases count", zap.Int("cases", event.Payload.Cases),
			zap.String("trace_id", event.TraceID),
			zap.String("date", event.Payload.Date),
			zap.String("city", event.Payload.City),
			zap.String("region", event.Payload.Region))
		return false
	}
	return true
}
