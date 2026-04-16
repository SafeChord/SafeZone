package schema_test

import (
	"context"
	"fmt"
	"testing"

	"safezone.service.worker-golang/app/pkg/logger"
	"safezone.service.worker-golang/app/schema"
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

func newValidator() *schema.CovidValidator {
	cache := &mockCache{
		cities:  map[string]int{"Taipei": 1},
		regions: map[string]int{"1|Zhongzheng": 10},
	}
	return schema.NewCovidValidator(
		logger.NewContextLogger("test", "0.0.0", "test"),
		cache,
	)
}

func validEvent() schema.CovidEvent {
	e := schema.CovidEvent{TraceID: "t1"}
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
