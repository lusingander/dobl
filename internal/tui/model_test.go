package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lusingander/dobl"
)

func TestFilterSteps(t *testing.T) {
	steps := sampleSteps()

	tests := []struct {
		name   string
		filter FilterMode
		want   []string
	}{
		{name: "all", filter: FilterAll, want: []string{"#1", "#2", "#3", "#4"}},
		{name: "problems", filter: FilterProblems, want: []string{"#3", "#4"}},
		{name: "warnings", filter: FilterWarnings, want: []string{"#3"}},
		{name: "failed", filter: FilterFailed, want: []string{"#4"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterSteps(steps, tt.filter, "")
			assertStepIDs(t, got, tt.want)
		})
	}
}

func TestSearchMatchesStepFieldsAndOutputTail(t *testing.T) {
	steps := sampleSteps()

	got := filterSteps(steps, FilterAll, "missing dependency")
	assertStepIDs(t, got, []string{"#4"})

	got = filterSteps(steps, FilterAll, "metadata")
	assertStepIDs(t, got, []string{"#1"})

	got = filterSteps(steps, FilterAll, "copy")
	assertStepIDs(t, got, []string{"#3"})
}

func TestUpdateNavigationClampsSelection(t *testing.T) {
	model := NewModel(sampleSteps(), "test.log")

	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyDown})
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyDown})
	if model.selected != 2 {
		t.Fatalf("selected = %d, want 2", model.selected)
	}

	for range 10 {
		model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyDown})
	}
	if model.selected != len(model.visible)-1 {
		t.Fatalf("selected = %d, want last", model.selected)
	}

	for range 10 {
		model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyUp})
	}
	if model.selected != 0 {
		t.Fatalf("selected = %d, want 0", model.selected)
	}
}

func TestUpdateFilterCyclePreservesValidSelection(t *testing.T) {
	model := NewModel(sampleSteps(), "test.log")
	model.selected = len(model.visible) - 1

	model = updateModel(t, model, keyRunes("f"))
	if model.filter != FilterProblems {
		t.Fatalf("filter = %s, want problems", model.filter)
	}
	assertStepIDs(t, model.visible, []string{"#3", "#4"})
	if model.selected != 1 {
		t.Fatalf("selected = %d, want 1", model.selected)
	}

	model = updateModel(t, model, keyRunes("f"))
	if model.filter != FilterWarnings {
		t.Fatalf("filter = %s, want warnings", model.filter)
	}
	assertStepIDs(t, model.visible, []string{"#3"})
	if model.selected != 0 {
		t.Fatalf("selected = %d, want 0", model.selected)
	}
}

func TestUpdateSearchModeFiltersAndEscClears(t *testing.T) {
	model := NewModel(sampleSteps(), "test.log")

	model = updateModel(t, model, keyRunes("/"))
	if !model.searching {
		t.Fatalf("searching = false, want true")
	}
	for _, r := range "run" {
		model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	assertStepIDs(t, model.visible, []string{"#4"})

	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	if model.searching {
		t.Fatalf("searching = true, want false")
	}

	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyEsc})
	if model.search != "" {
		t.Fatalf("search = %q, want empty", model.search)
	}
	assertStepIDs(t, model.visible, []string{"#1", "#2", "#3", "#4"})
}

func TestViewHandlesEmptyAndNarrowScreens(t *testing.T) {
	model := NewModel(nil, "empty.log")
	model.width = 40
	model.height = 12

	view := model.View()
	for _, want := range []string{"Dobl TUI", "Steps", "(none)", "Details"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view %q does not contain %q", view, want)
		}
	}
}

func keyRunes(value string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value)}
}

func updateModel(t *testing.T, model Model, msg tea.Msg) Model {
	t.Helper()
	updated, _ := model.Update(msg)
	next, ok := updated.(Model)
	if !ok {
		t.Fatalf("updated model type = %T, want Model", updated)
	}
	return next
}

func assertStepIDs(t *testing.T, steps []dobl.Step, want []string) {
	t.Helper()
	if len(steps) != len(want) {
		t.Fatalf("step count = %d, want %d: %+v", len(steps), len(want), steps)
	}
	for i, step := range steps {
		if step.ID != want[i] {
			t.Fatalf("step[%d] = %s, want %s", i, step.ID, want[i])
		}
	}
}

func sampleSteps() []dobl.Step {
	return []dobl.Step{
		{
			ID:          "#1",
			Order:       1,
			DisplayName: "load metadata",
			Category:    dobl.StepCategoryInternal,
			Status:      dobl.EventStatusDone,
			Duration:    "0.1s",
			StartLine:   1,
			EndLine:     2,
		},
		{
			ID:          "#2",
			Order:       2,
			DisplayName: "FROM alpine",
			Category:    dobl.StepCategoryDockerfile,
			Status:      dobl.EventStatusCached,
			Instruction: "FROM",
			StartLine:   3,
			EndLine:     4,
		},
		{
			ID:            "#3",
			Order:         3,
			DisplayName:   "COPY . .",
			Category:      dobl.StepCategoryDockerfile,
			Status:        dobl.EventStatusWarning,
			Instruction:   "COPY",
			WarningCount:  1,
			WarningDetail: "copy produced a warning",
			StartLine:     5,
			EndLine:       7,
		},
		{
			ID:          "#4",
			Order:       4,
			DisplayName: "RUN make build",
			Category:    dobl.StepCategoryDockerfile,
			Status:      dobl.EventStatusError,
			Instruction: "RUN",
			OutputCount: 2,
			OutputTail:  []string{"compiling", "missing dependency"},
			ErrorDetail: "process did not complete successfully",
			StartLine:   8,
			EndLine:     12,
		},
	}
}
