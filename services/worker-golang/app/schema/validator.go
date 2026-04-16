package schema

import (
	"context"
	"regexp"

	"go.uber.org/zap"
	"safezone.service.worker-golang/app/pkg/logger"
)

// CacheReader is the read-only interface CovidValidator needs for city/region lookups.
// *cache.Cache satisfies this interface.
type CacheReader interface {
	GetCityID(city string) int
	GetRegionID(cityID int, region string) int
}

type CovidValidator struct {
	Logger      *logger.ContextLogger
	datePattern *regexp.Regexp // YYYY-MM-DD 格式
	cache       CacheReader
}

func NewCovidValidator(logger *logger.ContextLogger, cache CacheReader) *CovidValidator {
	return &CovidValidator{
		Logger:      logger,
		datePattern: regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`),
		cache:       cache,
	}
}

func (v *CovidValidator) Validate(ctx context.Context, event CovidEvent) bool {
	// date is not in YYYY-MM-DD format
	if !v.datePattern.MatchString(event.Payload.Date) {
		v.Logger.Warn(ctx, "Invalid date format", zap.String("date", event.Payload.Date), zap.String("trace_id", event.TraceID))
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
	// cases must be a non-negative integer
	if event.Payload.Cases < 0 {
		v.Logger.Warn(ctx, "Invalid cases count", zap.Int("cases", event.Payload.Cases),
			zap.String("trace_id", event.TraceID),
			zap.String("date", event.Payload.Date),
			zap.String("city", event.Payload.City),
			zap.String("region", event.Payload.Region))
		return false
	}
	return true
}
