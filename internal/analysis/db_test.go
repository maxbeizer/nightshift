package analysis

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/marcus/nightshift/internal/db"
)

func TestStoreAndLoadLatest(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dbPath := filepath.Join(home, "nightshift.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	now := time.Now().Truncate(time.Millisecond)
	result := &BusFactorResult{
		Component: "internal/auth",
		Timestamp: now,
		Metrics: &OwnershipMetrics{
			HerfindahlIndex:   0.5,
			GiniCoefficient:   0.6,
			Top1Percent:       60.0,
			Top3Percent:       85.0,
			Top5Percent:       95.0,
			RiskLevel:         "medium",
			TotalContributors: 3,
			BusFactor:         1,
		},
		Contributors: []CommitAuthor{
			{Name: "Alice", Email: "alice@example.com", Commits: 50},
			{Name: "Bob", Email: "bob@example.com", Commits: 30},
		},
		RiskLevel:  "medium",
		ReportPath: "/tmp/report.md",
	}

	if err := result.Store(database.SQL()); err != nil {
		t.Fatalf("Store() error = %v", err)
	}
	if result.ID == 0 {
		t.Error("Store() should set result ID")
	}

	loaded, err := LoadLatest(database.SQL(), "internal/auth")
	if err != nil {
		t.Fatalf("LoadLatest() error = %v", err)
	}
	if loaded == nil {
		t.Fatal("LoadLatest() returned nil")
	}

	if loaded.Component != "internal/auth" {
		t.Errorf("Component = %q, want internal/auth", loaded.Component)
	}
	if loaded.RiskLevel != "medium" {
		t.Errorf("RiskLevel = %q, want medium", loaded.RiskLevel)
	}
	if loaded.ReportPath != "/tmp/report.md" {
		t.Errorf("ReportPath = %q, want /tmp/report.md", loaded.ReportPath)
	}
	if loaded.Metrics == nil {
		t.Fatal("Metrics should not be nil")
	}
	if loaded.Metrics.BusFactor != 1 {
		t.Errorf("BusFactor = %d, want 1", loaded.Metrics.BusFactor)
	}
	if len(loaded.Contributors) != 2 {
		t.Fatalf("len(Contributors) = %d, want 2", len(loaded.Contributors))
	}
	if loaded.Contributors[0].Name != "Alice" {
		t.Errorf("Contributors[0].Name = %q, want Alice", loaded.Contributors[0].Name)
	}
}

func TestStoreNilDB(t *testing.T) {
	result := &BusFactorResult{Component: "test"}
	if err := result.Store(nil); err == nil {
		t.Fatal("Store(nil) should return error")
	}
}

func TestLoadLatestNilDB(t *testing.T) {
	if _, err := LoadLatest(nil, "test"); err == nil {
		t.Fatal("LoadLatest(nil) should return error")
	}
}

func TestLoadLatestNotFound(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dbPath := filepath.Join(home, "nightshift.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	result, err := LoadLatest(database.SQL(), "nonexistent")
	if err != nil {
		t.Fatalf("LoadLatest() error = %v", err)
	}
	if result != nil {
		t.Error("LoadLatest() should return nil for missing component")
	}
}

func TestLoadAll(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dbPath := filepath.Join(home, "nightshift.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	now := time.Now()

	for i := 0; i < 3; i++ {
		r := &BusFactorResult{
			Component: "pkg/core",
			Timestamp: now.Add(time.Duration(-i) * 24 * time.Hour),
			Metrics: &OwnershipMetrics{
				RiskLevel:         "low",
				TotalContributors: 5,
				BusFactor:         2,
			},
			Contributors: []CommitAuthor{{Name: "Dev", Commits: 10}},
			RiskLevel:    "low",
		}
		if err := r.Store(database.SQL()); err != nil {
			t.Fatalf("Store() %d error = %v", i, err)
		}
	}

	// Load all without date filter
	all, err := LoadAll(database.SQL(), "pkg/core", time.Time{})
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	if len(all) != 3 {
		t.Errorf("LoadAll() returned %d, want 3", len(all))
	}

	// Load with date filter (since yesterday)
	filtered, err := LoadAll(database.SQL(), "pkg/core", now.Add(-36*time.Hour))
	if err != nil {
		t.Fatalf("LoadAll(since) error = %v", err)
	}
	if len(filtered) != 2 {
		t.Errorf("LoadAll(since) returned %d, want 2", len(filtered))
	}

	// Load for non-existent component
	empty, err := LoadAll(database.SQL(), "nonexistent", time.Time{})
	if err != nil {
		t.Fatalf("LoadAll(nonexistent) error = %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("LoadAll(nonexistent) returned %d, want 0", len(empty))
	}
}

func TestLoadAllNilDB(t *testing.T) {
	if _, err := LoadAll(nil, "test", time.Time{}); err == nil {
		t.Fatal("LoadAll(nil) should return error")
	}
}

func TestStoreUpdatesID(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dbPath := filepath.Join(home, "nightshift.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	r1 := &BusFactorResult{
		Component:    "comp-a",
		Timestamp:    time.Now(),
		Metrics:      &OwnershipMetrics{RiskLevel: "low"},
		Contributors: []CommitAuthor{},
		RiskLevel:    "low",
	}
	r2 := &BusFactorResult{
		Component:    "comp-b",
		Timestamp:    time.Now(),
		Metrics:      &OwnershipMetrics{RiskLevel: "high"},
		Contributors: []CommitAuthor{},
		RiskLevel:    "high",
	}

	if err := r1.Store(database.SQL()); err != nil {
		t.Fatal(err)
	}
	if err := r2.Store(database.SQL()); err != nil {
		t.Fatal(err)
	}

	if r1.ID == r2.ID {
		t.Error("Store() should assign unique IDs")
	}
}
