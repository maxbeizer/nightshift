package setup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/marcus/nightshift/internal/tasks"
)

func TestPresetTasksSafe(t *testing.T) {
	defs := tasks.AllDefinitions()
	signals := RepoSignals{HasRelease: true, HasADR: true}
	selected := PresetTasks(PresetSafe, defs, signals)

	for _, def := range defs {
		if selected[def.Type] {
			if def.RiskLevel != tasks.RiskLow {
				t.Errorf("safe preset selected high-risk task %s (risk=%v)", def.Type, def.RiskLevel)
			}
			if def.CostTier > tasks.CostMedium {
				t.Errorf("safe preset selected expensive task %s (cost=%v)", def.Type, def.CostTier)
			}
		}
	}
}

func TestPresetTasksBalanced(t *testing.T) {
	defs := tasks.AllDefinitions()
	signals := RepoSignals{HasRelease: true, HasADR: true}
	selected := PresetTasks(PresetBalanced, defs, signals)

	if len(selected) == 0 {
		t.Fatal("balanced preset selected no tasks")
	}

	for _, def := range defs {
		if selected[def.Type] && def.RiskLevel > tasks.RiskMedium {
			t.Errorf("balanced preset selected high-risk task %s", def.Type)
		}
	}
}

func TestPresetTasksAggressive(t *testing.T) {
	defs := tasks.AllDefinitions()
	signals := RepoSignals{HasRelease: true, HasADR: true}
	selected := PresetTasks(PresetAggressive, defs, signals)

	safeSelected := PresetTasks(PresetSafe, defs, signals)
	if len(selected) < len(safeSelected) {
		t.Errorf("aggressive (%d) selected fewer tasks than safe (%d)", len(selected), len(safeSelected))
	}
}

func TestPresetTasksReleaseSignal(t *testing.T) {
	defs := tasks.AllDefinitions()

	withRelease := PresetTasks(PresetBalanced, defs, RepoSignals{HasRelease: true})
	withoutRelease := PresetTasks(PresetBalanced, defs, RepoSignals{HasRelease: false})

	if withRelease[tasks.TaskChangelogSynth] && withoutRelease[tasks.TaskChangelogSynth] {
		t.Error("changelog-synth should be excluded without release signal")
	}
	if withRelease[tasks.TaskReleaseNotes] && withoutRelease[tasks.TaskReleaseNotes] {
		t.Error("release-notes should be excluded without release signal")
	}
}

func TestPresetTasksADRSignal(t *testing.T) {
	defs := tasks.AllDefinitions()

	withADR := PresetTasks(PresetBalanced, defs, RepoSignals{HasADR: true})
	withoutADR := PresetTasks(PresetBalanced, defs, RepoSignals{HasADR: false})

	if withADR[tasks.TaskADRDraft] && withoutADR[tasks.TaskADRDraft] {
		t.Error("adr-draft should be excluded without ADR signal")
	}
}

func TestDetectRepoSignals(t *testing.T) {
	root := t.TempDir()

	// No signals initially
	signals := DetectRepoSignals([]string{root})
	if signals.HasRelease {
		t.Error("HasRelease should be false for empty project")
	}
	if signals.HasADR {
		t.Error("HasADR should be false for empty project")
	}

	// Add CHANGELOG.md
	if err := os.WriteFile(filepath.Join(root, "CHANGELOG.md"), []byte("# Changelog"), 0644); err != nil {
		t.Fatal(err)
	}
	signals = DetectRepoSignals([]string{root})
	if !signals.HasRelease {
		t.Error("HasRelease should be true with CHANGELOG.md")
	}

	// Add ADR directory
	if err := os.MkdirAll(filepath.Join(root, "docs", "adr"), 0755); err != nil {
		t.Fatal(err)
	}
	signals = DetectRepoSignals([]string{root})
	if !signals.HasADR {
		t.Error("HasADR should be true with docs/adr")
	}
}

func TestDetectRepoSignalsReleaseWorkflow(t *testing.T) {
	root := t.TempDir()

	workflowDir := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workflowDir, "release.yml"), []byte("name: release"), 0644); err != nil {
		t.Fatal(err)
	}

	signals := DetectRepoSignals([]string{root})
	if !signals.HasRelease {
		t.Error("HasRelease should be true with release.yml workflow")
	}
}

func TestDetectRepoSignalsEmptyProject(t *testing.T) {
	signals := DetectRepoSignals([]string{""})
	if signals.HasRelease || signals.HasADR {
		t.Error("empty project path should yield no signals")
	}
}

func TestIsReleaseTask(t *testing.T) {
	if !isReleaseTask(tasks.TaskChangelogSynth) {
		t.Error("changelog-synth should be a release task")
	}
	if !isReleaseTask(tasks.TaskReleaseNotes) {
		t.Error("release-notes should be a release task")
	}
	if isReleaseTask(tasks.TaskLintFix) {
		t.Error("lint-fix should not be a release task")
	}
}

func TestHasAny(t *testing.T) {
	root := t.TempDir()

	if hasAny(root, []string{"nonexistent.md"}) {
		t.Error("hasAny should return false for nonexistent files")
	}

	if err := os.WriteFile(filepath.Join(root, "CHANGELOG.md"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if !hasAny(root, []string{"CHANGELOG.md"}) {
		t.Error("hasAny should return true for existing file")
	}
}
