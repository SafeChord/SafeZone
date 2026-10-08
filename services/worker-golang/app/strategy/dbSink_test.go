package strategy_test

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"safezone.service.worker-golang/app/pkg/cache"
	"safezone.service.worker-golang/app/pkg/logger"
	"safezone.service.worker-golang/app/schema"
	"safezone.service.worker-golang/app/strategy"
)

func testLogger() *logger.ContextLogger {
	l := logger.NewContextLogger("test", "0.0.0", "test")
	l.Logger = zap.NewNop()
	return l
}

func makeEvent(date, city, region string, cases int) schema.CovidEvent {
	var e schema.CovidEvent
	e.EventType = "covid.case.reported"
	e.TraceID = "trace-test"
	e.Version = "1.0"
	e.Payload.Date = date
	e.Payload.City = city
	e.Payload.Region = region
	e.Payload.Cases = cases
	return e
}

// TestDBSink_CommitFailureKeepsBuffer verifies that when tx.Commit() fails,
// the buffer is NOT cleared (WG-3).
func TestDBSink_CommitFailureKeepsBuffer(t *testing.T) {
	raw, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer raw.Close()
	db := sqlx.NewDb(raw, "pgx")

	mock.ExpectQuery("FROM cities").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "Taipei"))
	mock.ExpectQuery("FROM regions").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "city_id"}).AddRow(10, "Zhongzheng", 1))

	sink := strategy.NewDBSink(testLogger(), db, cache.NewCache(db))

	mock.ExpectBegin()
	mock.ExpectExec("covid_cases").
		WillReturnResult(sqlmock.NewResult(0, 1))
	commitErr := errors.New("disk full on commit")
	mock.ExpectCommit().WillReturnError(commitErr)

	buffer := []schema.CovidEvent{
		makeEvent("2024-01-01", "Taipei", "Zhongzheng", 10),
	}

	err = sink.Flush(context.Background(), &buffer)
	if err == nil {
		t.Fatal("expected error on commit failure, got nil")
	}
	if !errors.Is(err, commitErr) {
		t.Fatalf("expected commit error %v, got %v", commitErr, err)
	}

	if len(buffer) != 1 {
		t.Fatalf("buffer should not be cleared on commit failure; got len %d, want 1", len(buffer))
	}
}

// TestDBSink_ExecFailureKeepsBuffer verifies that when tx.ExecContext fails,
// transaction rolls back and the buffer is NOT cleared.
func TestDBSink_ExecFailureKeepsBuffer(t *testing.T) {
	raw, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer raw.Close()
	db := sqlx.NewDb(raw, "pgx")

	mock.ExpectQuery("FROM cities").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "Taipei"))
	mock.ExpectQuery("FROM regions").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "city_id"}).AddRow(10, "Zhongzheng", 1))

	sink := strategy.NewDBSink(testLogger(), db, cache.NewCache(db))

	execErr := errors.New("syntax error or connection lost")
	mock.ExpectBegin()
	mock.ExpectExec("covid_cases").WillReturnError(execErr)
	mock.ExpectRollback()

	buffer := []schema.CovidEvent{
		makeEvent("2024-01-01", "Taipei", "Zhongzheng", 10),
	}

	err = sink.Flush(context.Background(), &buffer)
	if err == nil {
		t.Fatal("expected error on exec failure, got nil")
	}
	if !errors.Is(err, execErr) {
		t.Fatalf("expected exec error %v, got %v", execErr, err)
	}

	if len(buffer) != 1 {
		t.Fatalf("buffer should not be cleared on exec failure; got len %d, want 1", len(buffer))
	}
}

// TestDBSink_EmptyBufferNoop verifies that calling Flush on an empty buffer does nothing.
func TestDBSink_EmptyBufferNoop(t *testing.T) {
	raw, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer raw.Close()
	db := sqlx.NewDb(raw, "pgx")

	mock.ExpectQuery("FROM cities").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "Taipei"))
	mock.ExpectQuery("FROM regions").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "city_id"}).AddRow(10, "Zhongzheng", 1))

	sink := strategy.NewDBSink(testLogger(), db, cache.NewCache(db))

	buffer := []schema.CovidEvent{}
	if err := sink.Flush(context.Background(), &buffer); err != nil {
		t.Fatalf("expected nil on empty buffer, got %v", err)
	}
}

type anyMatcher struct{}

func (anyMatcher) Match(v driver.Value) bool { return true }

// TestDBSink_LastWinsDedupe verifies DEF-2: the later event in the buffer overwrites the earlier.
func TestDBSink_LastWinsDedupe(t *testing.T) {
	raw, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer raw.Close()
	db := sqlx.NewDb(raw, "pgx")

	mock.ExpectQuery("FROM cities").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "Taipei"))
	mock.ExpectQuery("FROM regions").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "city_id"}).AddRow(10, "Zhongzheng", 1))

	sink := strategy.NewDBSink(testLogger(), db, cache.NewCache(db))

	mock.ExpectBegin()
	// Only ONE row should be inserted, with cases = 20 (the later event)
	mock.ExpectExec("covid_cases").
		WithArgs("2024-01-01", 1, 10, 20).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	buffer := []schema.CovidEvent{
		makeEvent("2024-01-01", "Taipei", "Zhongzheng", 10),
		makeEvent("2024-01-01", "Taipei", "Zhongzheng", 20),
	}

	if err := sink.Flush(context.Background(), &buffer); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if len(buffer) != 0 {
		t.Fatalf("buffer should be cleared on success; got len %d", len(buffer))
	}
}
