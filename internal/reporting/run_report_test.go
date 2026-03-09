package reporting

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRenderRunReport(t *testing.T) {
	start := time.Date(2024, 6, 15, 2, 0, 0, 0, time.UTC)
	end := start.Add(3 * time.Hour)

	results := &RunResults{
		StartTime:       start,
		EndTime:         end,
		StartBudget:     100000,
		UsedBudget:      40000,
		RemainingBudget: 60000,
		Tasks: []TaskResult{
			{Project: "/proj/a", TaskType: "lint-fix", Title: "Fixed lint", Status: "completed", TokensUsed: 20000, Duration: 30 * time.Second},
			{Project: "/proj/a", TaskType: "dead-code", Title: "Dead code scan", Status: "failed", TokensUsed: 15000, Duration: 20 * time.Second},
			{Project: "/proj/b", TaskType: "test-gap", Title: "Test gap", Status: "skipped", SkipReason: "insufficient budget"},
		},
	}

	content, err := RenderRunReport(results, "/tmp/nightshift.log")
	if err != nil {
		t.Fatalf("RenderRunReport() error = %v", err)
	}

	for _, want := range []string{
		"# Nightshift Run",
		"2024-06-15",
		"1 completed",
		"1 failed",
		"1 skipped",
		"/tmp/nightshift.log",
		"Tasks Completed",
		"lint-fix",
		"Tasks Failed",
		"dead-code",
		"Tasks Skipped",
		"insufficient budget",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("RenderRunReport() missing %q", want)
		}
	}
}

func TestRenderRunReportNilResults(t *testing.T) {
	_, err := RenderRunReport(nil, "")
	if err == nil {
		t.Fatal("RenderRunReport(nil) should return error")
	}
}

func TestRenderRunReportNoLogPath(t *testing.T) {
	results := &RunResults{
		StartTime: time.Now(),
		EndTime:   time.Now(),
	}
	content, err := RenderRunReport(results, "")
	if err != nil {
		t.Fatalf("RenderRunReport() error = %v", err)
	}
	if strings.Contains(content, "Logs:") {
		t.Error("expected no Logs line when logPath is empty")
	}
}

func TestDefaultRunReportPath(t *testing.T) {
	ts := time.Date(2024, 6, 15, 14, 30, 45, 0, time.UTC)
	path := DefaultRunReportPath(ts)

	if !strings.Contains(path, "nightshift") {
		t.Errorf("path %q should contain 'nightshift'", path)
	}
	if !strings.HasSuffix(path, "run-2024-06-15-143045.md") {
		t.Errorf("path %q has unexpected suffix", path)
	}
}

func TestSaveRunReport(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "report.md")

	results := &RunResults{
		StartTime: time.Now().Add(-time.Hour),
		EndTime:   time.Now(),
		Tasks: []TaskResult{
			{Project: "/proj", TaskType: "lint", Title: "Lint fix", Status: "completed", TokensUsed: 5000},
		},
	}

	if err := SaveRunReport(results, path, ""); err != nil {
		t.Fatalf("SaveRunReport() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading saved report: %v", err)
	}
	if !strings.Contains(string(data), "Nightshift Run") {
		t.Error("saved report missing expected header")
	}
}

func TestSaveRunReportNilResults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.md")
	if err := SaveRunReport(nil, path, ""); err == nil {
		t.Fatal("SaveRunReport(nil) should return error")
	}
}
