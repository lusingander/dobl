package tui

import (
	"bytes"
	"os"
	"strconv"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

func TestParseFilterMode(t *testing.T) {
	tests := []struct {
		value string
		want  FilterMode
	}{
		{value: "", want: FilterAll},
		{value: "all", want: FilterAll},
		{value: "problems", want: FilterProblems},
		{value: "warnings", want: FilterWarnings},
		{value: "failed", want: FilterFailed},
		{value: "FAILED", want: FilterFailed},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got, err := ParseFilterMode(tt.value)
			if err != nil {
				t.Fatalf("parse filter mode returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("filter = %s, want %s", got, tt.want)
			}
		})
	}

	if _, err := ParseFilterMode("slow"); err == nil {
		t.Fatalf("parse filter mode returned nil error for unknown filter")
	}
}

func TestNextProblemIndexWrapsThroughVisibleSteps(t *testing.T) {
	steps := sampleSteps()

	if got := nextProblemIndex(steps, 0, 1); got != 2 {
		t.Fatalf("next problem index = %d, want 2", got)
	}
	if got := nextProblemIndex(steps, 2, 1); got != 3 {
		t.Fatalf("next problem index = %d, want 3", got)
	}
	if got := nextProblemIndex(steps, 3, 1); got != 2 {
		t.Fatalf("next problem index = %d, want 2 after wrap", got)
	}
	if got := nextProblemIndex(steps, 2, -1); got != 3 {
		t.Fatalf("previous problem index = %d, want 3 after wrap", got)
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

func TestUpdateProblemNavigationAndReset(t *testing.T) {
	model := NewModel(sampleSteps(), "test.log")

	model = updateModel(t, model, keyRunes("n"))
	if model.selected != 2 {
		t.Fatalf("selected = %d, want first problem", model.selected)
	}

	model = updateModel(t, model, keyRunes("N"))
	if model.selected != 3 {
		t.Fatalf("selected = %d, want previous problem after wrap", model.selected)
	}

	model = updateModel(t, model, keyRunes("p"))
	if model.filter != FilterProblems {
		t.Fatalf("filter = %s, want problems", model.filter)
	}
	assertStepIDs(t, model.visible, []string{"#3", "#4"})

	model.search = "run"
	model.refreshVisible()
	model = updateModel(t, model, keyRunes("r"))
	if model.filter != FilterAll || model.search != "" || model.searching {
		t.Fatalf("filter/search = %s/%q/%v, want reset", model.filter, model.search, model.searching)
	}
	assertStepIDs(t, model.visible, []string{"#1", "#2", "#3", "#4"})
}

func TestNextProblemIndexReturnsMinusOneWhenNoProblems(t *testing.T) {
	steps := []dobl.Step{
		{ID: "#1", Status: dobl.EventStatusDone},
		{ID: "#2", Status: dobl.EventStatusCached},
	}
	if got := nextProblemIndex(steps, 0, 1); got != -1 {
		t.Fatalf("next problem index = %d, want -1", got)
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

func TestUpdateDetailScrollResetsWhenSelectionChanges(t *testing.T) {
	model := NewModel(sampleSteps(), "test.log")

	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyPgDown})
	if model.detailTop != 5 {
		t.Fatalf("detailTop = %d, want 5", model.detailTop)
	}

	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyDown})
	if model.detailTop != 0 {
		t.Fatalf("detailTop = %d, want 0", model.detailTop)
	}

	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyPgDown})
	model = updateModel(t, model, keyRunes("f"))
	if model.detailTop != 0 {
		t.Fatalf("detailTop = %d, want 0 after filter", model.detailTop)
	}
}

func TestUpdateTabFocusMakesJKScrollDetails(t *testing.T) {
	model := NewModel(sampleSteps(), "test.log")

	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyTab})
	if model.focus != FocusDetails {
		t.Fatalf("focus = %s, want details", model.focus)
	}

	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyDown})
	if model.selected != 0 {
		t.Fatalf("selected = %d, want unchanged", model.selected)
	}
	if model.detailTop != 1 {
		t.Fatalf("detailTop = %d, want 1", model.detailTop)
	}

	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyTab})
	if model.focus != FocusSteps {
		t.Fatalf("focus = %s, want steps", model.focus)
	}
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyDown})
	if model.selected != 1 {
		t.Fatalf("selected = %d, want 1", model.selected)
	}
}

func TestUpdateDirectionalFocusShortcuts(t *testing.T) {
	model := NewModel(sampleSteps(), "test.log")

	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRight})
	if model.focus != FocusDetails {
		t.Fatalf("focus = %s, want details", model.focus)
	}

	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyLeft})
	if model.focus != FocusSteps {
		t.Fatalf("focus = %s, want steps", model.focus)
	}

	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	if model.focus != FocusDetails {
		t.Fatalf("focus = %s, want details after enter", model.focus)
	}

	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyBackspace})
	if model.focus != FocusSteps {
		t.Fatalf("focus = %s, want steps after backspace", model.focus)
	}
}

func TestValidateTerminalOutputAllowsCustomWriter(t *testing.T) {
	if err := validateTerminalOutput(&bytes.Buffer{}); err != nil {
		t.Fatalf("validate terminal output returned error for custom writer: %v", err)
	}
}

func TestValidateTerminalOutputRejectsDumbTerminal(t *testing.T) {
	t.Setenv("TERM", "dumb")
	if err := validateTerminalOutput(&bytes.Buffer{}); err == nil {
		t.Fatalf("validate terminal output returned nil error for TERM=dumb")
	}
}

func TestValidateTerminalOutputRejectsNonTerminalFile(t *testing.T) {
	t.Setenv("TERM", "xterm")
	file, err := os.CreateTemp(t.TempDir(), "output")
	if err != nil {
		t.Fatalf("create temp output: %v", err)
	}
	defer file.Close()

	err = validateTerminalOutput(file)
	if err == nil {
		t.Fatalf("validate terminal output returned nil error for non-terminal file")
	}
	if !strings.Contains(err.Error(), "requires terminal output") {
		t.Fatalf("error = %q", err)
	}
}

func TestDetailLinesIncludesDiagnosticsAndOutputTail(t *testing.T) {
	lines := strings.Join(detailLines(sampleSteps()[3]), "\n")
	for _, want := range []string{
		"Details",
		"#4 ERROR",
		"Diagnostic",
		"Error: process did not complete successfully",
		"Output tail (2)",
		"missing dependency",
		"Metadata",
		"Instruction: RUN",
	} {
		if !strings.Contains(lines, want) {
			t.Fatalf("detail lines %q do not contain %q", lines, want)
		}
	}
	if strings.Index(lines, "Diagnostic") > strings.Index(lines, "Metadata") {
		t.Fatalf("detail lines put metadata before diagnostics: %q", lines)
	}
}

func TestViewHandlesEmptyAndNarrowScreens(t *testing.T) {
	model := NewModel(nil, "empty.log")
	model.width = 40
	model.height = 12

	view := model.View()
	for _, want := range []string{"Dobl TUI", "Timeline: (none)", "Steps", "(none)", "Details"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view %q does not contain %q", view, want)
		}
	}
}

func TestPaneWidthsFavorDetails(t *testing.T) {
	listWidth, detailWidth := paneWidths(120)
	if listWidth != 48 {
		t.Fatalf("list width = %d, want 48", listWidth)
	}
	if detailWidth != 70 {
		t.Fatalf("detail width = %d, want 70", detailWidth)
	}

	listWidth, detailWidth = paneWidths(180)
	if listWidth != 56 {
		t.Fatalf("wide list width = %d, want capped width 56", listWidth)
	}
	if detailWidth != 122 {
		t.Fatalf("wide detail width = %d, want 122", detailWidth)
	}
}

func TestEmptyViewExplainsFilterAndSearch(t *testing.T) {
	model := NewModel(sampleSteps(), "test.log")
	model.filter = FilterFailed
	model.search = "copy"
	model.refreshVisible()

	view := model.View()
	if want := `No steps match filter failed and search "copy"`; !strings.Contains(view, want) {
		t.Fatalf("view %q does not contain %q", view, want)
	}
}

func TestTimelineViewMarksSelectedAndProblemSteps(t *testing.T) {
	model := NewModel(sampleSteps(), "test.log")
	model.selected = 2

	timeline := model.timelineView(80)
	for _, want := range []string{"Timeline:", "#1D", "#2C", "[!#3W]", "x#4E"} {
		if !strings.Contains(timeline, want) {
			t.Fatalf("timeline %q does not contain %q", timeline, want)
		}
	}
}

func TestPanelTitlesShowFocusedPane(t *testing.T) {
	model := NewModel(sampleSteps(), "test.log")
	if got := strings.Split(model.listView(40, 4), "\n")[0]; got != "Steps (4/4) *" {
		t.Fatalf("list title = %q, want focused marker", got)
	}

	model.focus = FocusDetails
	if got := strings.Split(model.detailView(40, 4), "\n")[0]; got != "Details #1 DONE *" {
		t.Fatalf("detail title = %q, want focused marker", got)
	}
}

func TestHelpViewFollowsFocusedPane(t *testing.T) {
	model := NewModel(sampleSteps(), "test.log")
	if got := model.helpView(120); !strings.Contains(got, "enter/right details") {
		t.Fatalf("steps help = %q, want detail focus shortcut", got)
	}

	model.focus = FocusDetails
	if got := model.helpView(120); !strings.Contains(got, "left/backspace steps") {
		t.Fatalf("details help = %q, want steps focus shortcut", got)
	}
}

func TestListViewShowsStatusMarkers(t *testing.T) {
	model := NewModel(sampleSteps(), "test.log")
	view := model.listView(80, 6)
	for _, want := range []string{
		"> . #1   DONE",
		"  = #2   CACHED",
		"  ! #3   WARNING",
		"  x #4   ERROR",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("list view %q does not contain %q", view, want)
		}
	}
}

func TestTimelineViewTrimsToWidth(t *testing.T) {
	model := NewModel(sampleSteps(), "test.log")
	timeline := model.timelineView(18)
	if got := lipgloss.Width(timeline); got > 18 {
		t.Fatalf("timeline width = %d, want <= 18: %q", got, timeline)
	}
}

func TestTimelineViewCentersSelectedStepWhenNarrow(t *testing.T) {
	steps := make([]dobl.Step, 0, 12)
	for i := 1; i <= 12; i++ {
		steps = append(steps, dobl.Step{
			ID:     "#" + strconv.Itoa(i),
			Status: dobl.EventStatusDone,
		})
	}
	steps[9].Status = dobl.EventStatusError

	model := NewModel(steps, "long.log")
	model.selected = 6
	timeline := model.timelineView(34)
	if got := lipgloss.Width(timeline); got > 34 {
		t.Fatalf("timeline width = %d, want <= 34: %q", got, timeline)
	}
	for _, want := range []string{"Timeline:", "...", "[#7D]"} {
		if !strings.Contains(timeline, want) {
			t.Fatalf("timeline %q does not contain %q", timeline, want)
		}
	}
	if strings.Contains(timeline, "#1D") {
		t.Fatalf("timeline %q should not keep the far-left step when narrow", timeline)
	}
}

func TestSelectedListLineTrimsBeforeStyling(t *testing.T) {
	steps := sampleSteps()
	steps[0].DisplayName = strings.Repeat("long-name-", 20)
	model := NewModel(steps, "test.log")

	view := model.listView(24, 4)
	for _, line := range strings.Split(view, "\n") {
		if got := lipgloss.Width(line); got > 24 {
			t.Fatalf("line width = %d, want <= 24: %q", got, line)
		}
		if strings.Contains(line, "> ") && strings.Contains(line, "long-name-long-name") {
			t.Fatalf("selected line was not trimmed before styling: %q", line)
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
