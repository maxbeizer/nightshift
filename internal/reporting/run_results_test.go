package reporting

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSaveAndLoadRunResults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "results.json")

	start := time.Date(2024, 6, 15, 2, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Hour)

	original := &RunResults{
		Date:            start,
		StartBudget:     100000,
		UsedBudget:      40000,
		RemainingBudget: 60000,
		StartTime:       start,
		EndTime:         end,
		Tasks: []TaskResult{
			{Project: "/proj/a", TaskType: "lint-fix", Title: "Fixed lint", Status: "completed", TokensUsed: 20000},
			{Project: "/proj/b", TaskType: "test-gap", Title: "Test gap", Status: "skipped", SkipReason: "budget"},
		},
	}

	if err := SaveRunResults(original, path); err != nil {
		t.Fatalf("SaveRunResults() error = %v", err)
	}

	loaded, err := LoadRunResults(path)
	if err != nil {
		t.Fatalf("LoadRunResults() error = %v", err)
	}

	if loaded.StartBudget != original.StartBudget {
		t.Errorf("StartBudget = %d, want %d", loaded.StartBudget, original.StartBudget)
	}
	if loaded.UsedBudget != original.UsedBudget {
		t.Errorf("UsedBudget = %d, want %d", loaded.UsedBudget, original.UsedBudget)
	}
	if len(loaded.Tasks) != 2 {
		t.Fatalf("len(Tasks) = %d, want 2", len(loaded.Tasks))
	}
	if loaded.Tasks[0].TaskType != "lint-fix" {
		t.Errorf("Tasks[0].TaskType = %q, want lint-fix", loaded.Tasks[0].TaskType)
	}
	if loaded.Tasks[1].SkipReason != "budget" {
		t.Errorf("Tasks[1].SkipReason = %q, want budget", loaded.Tasks[1].SkipReason)
	}
}

func TestSaveRunResultsNil(t *testing.T) {
	dir := t.TempDir()
	if err := SaveRunResults(nil, filepath.Join(dir, "results.json")); err == nil {
		t.Fatal("SaveRunResults(nil) should return error")
	}
}

func TestLoadRunResultsNotFound(t *testing.T) {
	if _, err := LoadRunResults("/nonexistent/path.json"); err == nil {
		t.Fatal("LoadRunResults(nonexistent) should return error")
	}
}

func TestSaveRunResultsCreatesDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "deep", "results.json")

	results := &RunResults{StartTime: time.Now(), EndTime: time.Now()}
	if err := SaveRunResults(results, path); err != nil {
		t.Fatalf("SaveRunResults() error = %v", err)
	}

	if _, err := LoadRunResults(path); err != nil {
		t.Fatalf("LoadRunResults() after save error = %v", err)
	}
}

func TestDefaultRunResultsPath(t *testing.T) {
	ts := time.Date(2024, 6, 15, 14, 30, 45, 0, time.UTC)
	path := DefaultRunResultsPath(ts)

	if !strings.HasSuffix(path, "run-2024-06-15-143045.json") {
		t.Errorf("path %q has unexpected suffix", path)
	}
	if !strings.Contains(path, "nightshift") {
		t.Errorf("path %q should contain 'nightshift'", path)
	}
}

func TestDefaultReportsDir(t *testing.T) {
	dir := DefaultReportsDir()
	if !strings.Contains(dir, "nightshift") {
		t.Errorf("dir %q should contain 'nightshift'", dir)
	}
	if !strings.Contains(dir, "reports") {
		t.Errorf("dir %q should contain 'reports'", dir)
	}
}
