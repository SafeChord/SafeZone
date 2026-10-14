package requirements_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"safezone.service.worker-golang/app/schema"
)

// exampleAreas knows exactly the cities and regions of the example events.
type exampleAreas struct{ cities, regions map[string]bool }

func (a exampleAreas) GetCityID(city string) int {
	if a.cities[city] {
		return 1
	}
	return -1
}

func (a exampleAreas) GetRegionID(cityID int, region string) int {
	if cityID == 1 && a.regions[region] {
		return 1
	}
	return -1
}

// WK-R10: The worker honours the contracts it depends on.
// Scenario: event the contract allows. The examples under contract/examples/ are the
// contract-valid events; the contract tests in utils/ check that each satisfies the schema.
func TestWK_R10_AcceptsEveryContractExample(t *testing.T) {
	dir := os.Getenv("CONTRACT_DIR")
	if dir == "" {
		t.Fatal("CONTRACT_DIR is not set; it must point at the repository's contract/ directory")
	}
	paths, err := filepath.Glob(filepath.Join(dir, "examples", "covid_event", "*.json"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("no example events under %s/examples/covid_event (err: %v)", dir, err)
	}

	events := make(map[string]schema.CovidEvent, len(paths))
	areas := exampleAreas{cities: map[string]bool{}, regions: map[string]bool{}}
	for _, p := range paths {
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read %s: %v", p, err)
		}
		// The same decoding the Kafka source applies to a record's value.
		var e schema.CovidEvent
		if err := json.Unmarshal(raw, &e); err != nil {
			t.Errorf("%s: worker cannot parse it: %v", filepath.Base(p), err)
			continue
		}
		events[filepath.Base(p)] = e
		areas.cities[e.Payload.City] = true
		areas.regions[e.Payload.Region] = true
	}

	v := schema.NewCovidValidator(quietLogger(), areas)
	for name, e := range events {
		if !v.Validate(context.Background(), e) {
			t.Errorf("%s: worker rejects a contract-valid event", name)
		}
	}
}
