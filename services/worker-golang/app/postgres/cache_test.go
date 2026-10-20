package postgres

import (
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestNewCacheReturnsQueryScanAndIterationErrors(t *testing.T) {
	queryErr := errors.New("query failed")
	iterationErr := errors.New("iteration failed")
	tests := []struct {
		name      string
		rows      *sqlmock.Rows
		queryErr  error
		wantCause error
		wantError string
	}{
		{
			name:      "city query",
			queryErr:  queryErr,
			wantCause: queryErr,
		},
		{
			name:      "city scan",
			rows:      sqlmock.NewRows([]string{"id", "name"}).AddRow("not-an-id", "Taipei"),
			wantError: "Scan error",
		},
		{
			name:      "city iteration",
			rows:      sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "Taipei").RowError(0, iterationErr),
			wantCause: iterationErr,
		},
		{
			name:      "region query",
			queryErr:  queryErr,
			wantCause: queryErr,
		},
		{
			name:      "region scan",
			rows:      sqlmock.NewRows([]string{"id", "name", "city_id"}).AddRow("not-an-id", "Central", 1),
			wantError: "Scan error",
		},
		{
			name:      "region iteration",
			rows:      sqlmock.NewRows([]string{"id", "name", "city_id"}).AddRow(10, "Central", 1).RowError(0, iterationErr),
			wantCause: iterationErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			if err != nil {
				t.Fatalf("creating sqlmock: %v", err)
			}
			defer raw.Close()
			db := sqlx.NewDb(raw, "sqlmock")

			if strings.HasPrefix(tt.name, "region") {
				mock.ExpectQuery("SELECT id, name FROM cities").
					RowsWillBeClosed().
					WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "Taipei"))
			}
			cityQuery := "SELECT id, name FROM cities"
			if strings.HasPrefix(tt.name, "region") {
				cityQuery = "SELECT id, name, city_id FROM regions"
			}
			expected := mock.ExpectQuery(cityQuery)
			if tt.queryErr != nil {
				expected.WillReturnError(tt.queryErr)
			} else {
				expected.RowsWillBeClosed()
				expected.WillReturnRows(tt.rows)
			}

			loaded, err := NewCache(db)
			if err == nil {
				t.Fatal("expected cache loading to fail")
			}
			if loaded != nil {
				t.Fatalf("expected no partial cache, got %#v", loaded)
			}
			if tt.queryErr != nil && !errors.Is(err, tt.wantCause) {
				t.Fatalf("expected query cause %v to be wrapped, got %v", tt.wantCause, err)
			}
			if tt.wantError != "" && !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("expected error to include %q, got %v", tt.wantError, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("SQL expectations: %v", err)
			}
		})
	}
}

func TestNewCacheClosesRowsAfterSuccessfulLoad(t *testing.T) {
	raw, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("creating sqlmock: %v", err)
	}
	defer raw.Close()
	db := sqlx.NewDb(raw, "sqlmock")

	mock.ExpectQuery("SELECT id, name FROM cities").
		RowsWillBeClosed().
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "Taipei"))
	mock.ExpectQuery("SELECT id, name, city_id FROM regions").
		RowsWillBeClosed().
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "city_id"}).AddRow(10, "Central", 1))

	loaded, err := NewCache(db)
	if err != nil {
		t.Fatalf("loading area tables: %v", err)
	}
	if loaded.GetCityID("Taipei") != 1 || loaded.GetRegionID(1, "Central") != 10 {
		t.Fatalf("loaded cache has wrong contents: city=%d region=%d", loaded.GetCityID("Taipei"), loaded.GetRegionID(1, "Central"))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("SQL expectations: %v", err)
	}
}
