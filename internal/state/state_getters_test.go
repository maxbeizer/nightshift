package state

import (
	"testing"
	"time"
)

func TestGetProjectState(t *testing.T) {
	s := newTestState(t)
	project := "/path/to/project"

	// Not tracked yet
	ps := s.GetProjectState(project)
	if ps != nil {
		t.Fatal("GetProjectState() should return nil for untracked project")
	}

	// Record a project run and task run
	s.RecordProjectRun(project)
	s.RecordTaskRun(project, "lint-fix")
	s.RecordTaskRun(project, "test-gap")

	ps = s.GetProjectState(project)
	if ps == nil {
		t.Fatal("GetProjectState() returned nil for tracked project")
	}

	if ps.RunCount != 1 {
		t.Errorf("RunCount = %d, want 1", ps.RunCount)
	}
	if ps.LastRun.IsZero() {
		t.Error("LastRun should not be zero")
	}
	if len(ps.TaskHistory) != 2 {
		t.Errorf("len(TaskHistory) = %d, want 2", len(ps.TaskHistory))
	}
	if _, ok := ps.TaskHistory["lint-fix"]; !ok {
		t.Error("TaskHistory missing lint-fix")
	}
	if _, ok := ps.TaskHistory["test-gap"]; !ok {
		t.Error("TaskHistory missing test-gap")
	}

	// Second run increments count
	s.RecordProjectRun(project)
	ps = s.GetProjectState(project)
	if ps.RunCount != 2 {
		t.Errorf("RunCount after second run = %d, want 2", ps.RunCount)
	}
}

func TestGetTodayRuns(t *testing.T) {
	s := newTestState(t)

	// No runs yet
	runs := s.GetTodayRuns()
	if len(runs) != 0 {
		t.Fatalf("GetTodayRuns() = %d runs, want 0", len(runs))
	}

	// Add a run for today
	now := time.Now()
	s.AddRunRecord(RunRecord{
		ID:         "run-1",
		StartTime:  now.Add(-time.Hour),
		EndTime:    now,
		Project:    "/proj/a",
		Tasks:      []string{"lint-fix"},
		TokensUsed: 5000,
		Status:     "success",
	})

	runs = s.GetTodayRuns()
	if len(runs) != 1 {
		t.Fatalf("GetTodayRuns() = %d runs after insert, want 1", len(runs))
	}
	if runs[0].ID != "run-1" {
		t.Errorf("run ID = %q, want run-1", runs[0].ID)
	}
	if runs[0].Status != "success" {
		t.Errorf("run status = %q, want success", runs[0].Status)
	}
}

func TestGetTodayRunsMultiple(t *testing.T) {
	s := newTestState(t)
	now := time.Now()

	s.AddRunRecord(RunRecord{
		ID:         "run-a",
		StartTime:  now.Add(-2 * time.Hour),
		EndTime:    now.Add(-time.Hour),
		Project:    "/proj/a",
		Tasks:      []string{"lint-fix"},
		TokensUsed: 3000,
		Status:     "success",
	})
	s.AddRunRecord(RunRecord{
		ID:         "run-b",
		StartTime:  now.Add(-30 * time.Minute),
		EndTime:    now,
		Project:    "/proj/b",
		Tasks:      []string{"test-gap", "dead-code"},
		TokensUsed: 7000,
		Status:     "failed",
	})

	runs := s.GetTodayRuns()
	if len(runs) != 2 {
		t.Fatalf("GetTodayRuns() = %d runs, want 2", len(runs))
	}
}

func TestGetTodaySummary(t *testing.T) {
	s := newTestState(t)
	now := time.Now()

	s.AddRunRecord(RunRecord{
		ID:         "run-1",
		StartTime:  now.Add(-2 * time.Hour),
		EndTime:    now.Add(-time.Hour),
		Project:    "/proj/a",
		Tasks:      []string{"lint-fix", "dead-code"},
		TokensUsed: 5000,
		Status:     "success",
	})
	s.AddRunRecord(RunRecord{
		ID:         "run-2",
		StartTime:  now.Add(-30 * time.Minute),
		EndTime:    now,
		Project:    "/proj/b",
		Tasks:      []string{"test-gap"},
		TokensUsed: 3000,
		Status:     "failed",
	})

	summary := s.GetTodaySummary()

	if summary.TotalRuns != 2 {
		t.Errorf("TotalRuns = %d, want 2", summary.TotalRuns)
	}
	if summary.SuccessfulRuns != 1 {
		t.Errorf("SuccessfulRuns = %d, want 1", summary.SuccessfulRuns)
	}
	if summary.FailedRuns != 1 {
		t.Errorf("FailedRuns = %d, want 1", summary.FailedRuns)
	}
	if summary.TotalTokens != 8000 {
		t.Errorf("TotalTokens = %d, want 8000", summary.TotalTokens)
	}
	if len(summary.Projects) != 2 {
		t.Errorf("len(Projects) = %d, want 2", len(summary.Projects))
	}
	if summary.TaskCounts["lint-fix"] != 1 {
		t.Errorf("TaskCounts[lint-fix] = %d, want 1", summary.TaskCounts["lint-fix"])
	}
	if summary.TaskCounts["test-gap"] != 1 {
		t.Errorf("TaskCounts[test-gap] = %d, want 1", summary.TaskCounts["test-gap"])
	}
}

func TestGetTodaySummaryEmpty(t *testing.T) {
	s := newTestState(t)

	summary := s.GetTodaySummary()
	if summary.TotalRuns != 0 {
		t.Errorf("TotalRuns = %d, want 0", summary.TotalRuns)
	}
	if summary.TaskCounts == nil {
		t.Error("TaskCounts should not be nil")
	}
}
