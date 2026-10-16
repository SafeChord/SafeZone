package event_test

import (
	"context"
	"fmt"
	"math"
	"testing"

	"go.uber.org/zap/zapcore"

	"safezone.service.worker-golang/app/event"
	"safezone.service.worker-golang/app/logger"
)

type mockCache struct {
	cities  map[string]int
	regions map[string]int // key: "cityID|regionName"
}

func (m *mockCache) GetCityID(city string) int {
	if id, ok := m.cities[city]; ok {
		return id
	}
	return -1
}

func (m *mockCache) GetRegionID(cityID int, region string) int {
	key := fmt.Sprintf("%d|%s", cityID, region)
	if id, ok := m.regions[key]; ok {
		return id
	}
	return -1
}

func newValidator() *event.CovidValidator {
	cache := &mockCache{
		cities:  map[string]int{"Taipei": 1},
		regions: map[string]int{"1|Zhongzheng": 10},
	}
	return event.NewCovidValidator(
		logger.NewContextLogger("test", "0.0.0", "test", zapcore.DebugLevel),
		cache,
	)
}

func validEvent() event.CovidEvent {
	e := event.CovidEvent{TraceID: "t1"}
	e.Payload.Date = "2024-01-01"
	e.Payload.City = "Taipei"
	e.Payload.Region = "Zhongzheng"
	e.Payload.Cases = 10
	return e
}

func TestValidator_ValidEvent(t *testing.T) {
	v := newValidator()
	if !v.Validate(context.Background(), validEvent()) {
		t.Fatal("expected valid event to pass")
	}
}

func TestValidator_InvalidDateFormat(t *testing.T) {
	v := newValidator()
	e := validEvent()
	e.Payload.Date = "01/01/2024" // wrong format
	if v.Validate(context.Background(), e) {
		t.Fatal("expected invalid date to fail")
	}
}

func TestValidator_InvalidCalendarDate(t *testing.T) {
	for _, date := range []string{"2026-13-45", "0000-01-01"} {
		t.Run(date, func(t *testing.T) {
			v := newValidator()
			e := validEvent()
			e.Payload.Date = date
			if v.Validate(context.Background(), e) {
				t.Fatalf("expected unstoreable date %q to fail", date)
			}
		})
	}
}

func TestValidator_UnknownCityOrRegion(t *testing.T) {
	v := newValidator()

	unknown := validEvent()
	unknown.Payload.City = "UnknownCity"
	if v.Validate(context.Background(), unknown) {
		t.Fatal("expected unknown city to fail")
	}

	wrongRegion := validEvent()
	wrongRegion.Payload.Region = "UnknownRegion"
	if v.Validate(context.Background(), wrongRegion) {
		t.Fatal("expected unknown region to fail")
	}
}

func TestValidator_NegativeCases(t *testing.T) {
	v := newValidator()
	e := validEvent()
	e.Payload.Cases = -1
	if v.Validate(context.Background(), e) {
		t.Fatal("expected negative cases to fail")
	}
}

func TestValidator_CaseTableIntegerUpperBound(t *testing.T) {
	v := newValidator()
	e := validEvent()
	e.Payload.Cases = math.MaxInt32
	if !v.Validate(context.Background(), e) {
		t.Fatal("expected the largest case-table integer to pass")
	}

	e.Payload.Cases = math.MaxInt32 + 1
	if v.Validate(context.Background(), e) {
		t.Fatal("expected a case count above the case-table integer range to fail")
	}
}
