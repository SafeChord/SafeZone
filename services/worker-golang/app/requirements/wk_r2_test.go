package requirements_test

import (
	"context"
	"database/sql/driver"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"

	"safezone.service.worker-golang/app/pkg/cache"
	"safezone.service.worker-golang/app/schema"
	"safezone.service.worker-golang/app/strategy"
)

// written records every value a statement sends to the database.
type written struct{ values *[]driver.Value }

func (w written) Match(v driver.Value) bool {
	*w.values = append(*w.values, v)
	return true
}

// WK-R2: The latest event for a key wins.
// Scenario: two events for one key arrive together.
//
// There is no case table in the unit-test container, so this checks what the sink sends:
// one row for the key, carrying the later count. PostgreSQL rejects an upsert that touches
// one row twice, so a write carrying both events would fail in production.
func TestWK_R2_LaterEventWinsWithinOneWrite(t *testing.T) {
	raw, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer raw.Close()
	db := sqlx.NewDb(raw, "pgx")

	mock.ExpectQuery("FROM cities").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "Taipei"))
	mock.ExpectQuery("FROM regions").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "city_id"}).AddRow(7, "region-0", 1))
	sink := strategy.NewDBSink(quietLogger(), db, cache.NewCache(db))

	var sent []driver.Value
	one := written{&sent}
	mock.ExpectBegin()
	// Four values are one row: date, city, region, cases.
	mock.ExpectExec("covid_cases").WithArgs(one, one, one, one).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	const earlier, later = 111, 222
	batch := []schema.CovidEvent{event(earlier, "region-0"), event(later, "region-0")}
	if err := sink.Flush(context.Background(), &batch); err != nil {
		t.Fatalf("writing two events for one key failed: %v", err)
	}

	var sawLater bool
	for _, v := range sent {
		switch v {
		case int64(earlier):
			t.Fatalf("the write carries the earlier count %d; the later event must win", earlier)
		case int64(later):
			sawLater = true
		}
	}
	if !sawLater {
		t.Fatalf("the write does not carry the later count %d (sent: %v)", later, sent)
	}
}
