package snapshots

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/marcus/nightshift/internal/db"
)

func TestGetSinceWeekStart(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dbPath := filepath.Join(home, "nightshift.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	weekStartDay := time.Monday
	collector := NewCollector(database, fakeClaude{weekly: 500, daily: 100}, nil, nil, nil, weekStartDay)

	// Insert two snapshots for the current week directly
	now := time.Now()
	ws := startOfWeek(now, weekStartDay)

	for i := 0; i < 2; i++ {
		_, err := database.SQL().Exec(
			`INSERT INTO snapshots (provider, timestamp, week_start, local_tokens, local_daily, day_of_week, hour_of_day, week_number, year)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			"claude",
			now.Add(time.Duration(-i)*time.Hour),
			ws,
			int64(500+i*100),
			int64(100+i*10),
			int(now.Weekday()),
			now.Hour(),
			0, 0,
		)
		if err != nil {
			t.Fatalf("insert snapshot %d: %v", i, err)
		}
	}

	// Insert a snapshot from a different week (should not appear)
	_, err = database.SQL().Exec(
		`INSERT INTO snapshots (provider, timestamp, week_start, local_tokens, local_daily, day_of_week, hour_of_day, week_number, year)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"claude",
		now.AddDate(0, 0, -14),
		ws.AddDate(0, 0, -14),
		300, 50, 1, 10, 0, 0,
	)
	if err != nil {
		t.Fatalf("insert old snapshot: %v", err)
	}

	snaps, err := collector.GetSinceWeekStart("claude")
	if err != nil {
		t.Fatalf("GetSinceWeekStart() error = %v", err)
	}
	if len(snaps) != 2 {
		t.Errorf("GetSinceWeekStart() returned %d snapshots, want 2", len(snaps))
	}
	for _, snap := range snaps {
		if snap.Provider != "claude" {
			t.Errorf("snapshot provider = %q, want claude", snap.Provider)
		}
	}
}

func TestGetSinceWeekStartEmpty(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dbPath := filepath.Join(home, "nightshift.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	collector := NewCollector(database, fakeClaude{}, nil, nil, nil, time.Monday)

	snaps, err := collector.GetSinceWeekStart("claude")
	if err != nil {
		t.Fatalf("GetSinceWeekStart() error = %v", err)
	}
	if len(snaps) != 0 {
		t.Errorf("expected 0 snapshots, got %d", len(snaps))
	}
}

func TestGetHourlyAverages(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dbPath := filepath.Join(home, "nightshift.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	collector := NewCollector(database, fakeClaude{weekly: 1000, daily: 200}, nil, nil, nil, time.Monday)

	now := time.Now()
	// Insert snapshots across different hours
	for _, hour := range []int{8, 8, 12, 12, 16} {
		ts := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, now.Location())
		daily := int64(100 + hour*10) // varying daily values
		_, err := database.SQL().Exec(
			`INSERT INTO snapshots (provider, timestamp, week_start, local_tokens, local_daily, day_of_week, hour_of_day, week_number, year)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			"claude", ts, ts, 1000, daily, int(ts.Weekday()), hour, 0, 0,
		)
		if err != nil {
			t.Fatalf("insert snapshot hour %d: %v", hour, err)
		}
	}

	averages, err := collector.GetHourlyAverages("claude", 7)
	if err != nil {
		t.Fatalf("GetHourlyAverages() error = %v", err)
	}
	if len(averages) != 3 {
		t.Fatalf("GetHourlyAverages() returned %d entries, want 3 (hours 8, 12, 16)", len(averages))
	}

	// Verify hours are in order
	if averages[0].Hour != 8 || averages[1].Hour != 12 || averages[2].Hour != 16 {
		t.Errorf("hours = %d, %d, %d; want 8, 12, 16", averages[0].Hour, averages[1].Hour, averages[2].Hour)
	}

	// Hour 8 has two entries with daily=180 each → avg=180
	if averages[0].AvgDailyTokens != 180 {
		t.Errorf("hour 8 avg = %f, want 180", averages[0].AvgDailyTokens)
	}
}

func TestGetHourlyAveragesZeroLookback(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dbPath := filepath.Join(home, "nightshift.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	collector := NewCollector(database, fakeClaude{}, nil, nil, nil, time.Monday)
	averages, err := collector.GetHourlyAverages("claude", 0)
	if err != nil {
		t.Fatalf("GetHourlyAverages(0) error = %v", err)
	}
	if len(averages) != 0 {
		t.Errorf("expected 0 averages, got %d", len(averages))
	}
}

func TestGetSinceWeekStartViaSnapshot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dbPath := filepath.Join(home, "nightshift.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	collector := NewCollector(database, fakeClaude{weekly: 700, daily: 120}, nil, nil, nil, time.Monday)

	// Take a real snapshot
	_, err = collector.TakeSnapshot(context.Background(), "claude")
	if err != nil {
		t.Fatalf("TakeSnapshot() error = %v", err)
	}

	snaps, err := collector.GetSinceWeekStart("claude")
	if err != nil {
		t.Fatalf("GetSinceWeekStart() error = %v", err)
	}
	if len(snaps) != 1 {
		t.Errorf("expected 1 snapshot via TakeSnapshot, got %d", len(snaps))
	}
}
