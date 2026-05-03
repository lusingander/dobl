package tui

import (
	"fmt"
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lusingander/dobl"
)

const (
	defaultWidth  = 100
	defaultHeight = 30
	minPaneWidth  = 24
)

type FilterMode int

const (
	FilterAll FilterMode = iota
	FilterProblems
	FilterWarnings
	FilterFailed
)

type Options struct {
	Source string
	Input  io.Reader
	Output io.Writer
}

type Model struct {
	steps     []dobl.Step
	visible   []dobl.Step
	source    string
	filter    FilterMode
	search    string
	searching bool
	selected  int
	width     int
	height    int
}

func NewModel(steps []dobl.Step, source string) Model {
	m := Model{
		steps:  append([]dobl.Step(nil), steps...),
		source: source,
		width:  defaultWidth,
		height: defaultHeight,
	}
	m.refreshVisible()
	return m
}

func Run(steps []dobl.Step, options Options) error {
	source := options.Source
	if source == "" {
		source = "stdin"
	}

	model := NewModel(steps, source)
	programOptions := []tea.ProgramOption{tea.WithAltScreen(), tea.WithInputTTY()}
	if options.Input != nil {
		programOptions = append(programOptions, tea.WithInput(options.Input))
	}
	if options.Output != nil {
		programOptions = append(programOptions, tea.WithOutput(options.Output))
	}

	_, err := tea.NewProgram(model, programOptions...).Run()
	return err
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		return m.updateKey(msg)
	default:
		return m, nil
	}
}

func (m Model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.searching {
		switch msg.String() {
		case "esc":
			m.searching = false
			if m.search == "" {
				m.refreshVisible()
			}
			return m, nil
		case "enter":
			m.searching = false
			return m, nil
		case "backspace", "ctrl+h":
			if m.search != "" {
				runes := []rune(m.search)
				m.search = string(runes[:len(runes)-1])
				m.refreshVisible()
			}
			return m, nil
		case "ctrl+c":
			return m, tea.Quit
		default:
			if len(msg.Runes) > 0 {
				m.search += string(msg.Runes)
				m.refreshVisible()
			}
			return m, nil
		}
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "j", "down":
		m.move(1)
	case "k", "up":
		m.move(-1)
	case "g", "home":
		m.selected = 0
	case "G", "end":
		if len(m.visible) > 0 {
			m.selected = len(m.visible) - 1
		}
	case "f":
		m.filter = nextFilter(m.filter)
		m.refreshVisible()
	case "/":
		m.searching = true
	case "esc":
		if m.search != "" {
			m.search = ""
			m.refreshVisible()
		}
	}
	return m, nil
}

func (m *Model) move(delta int) {
	if len(m.visible) == 0 {
		m.selected = 0
		return
	}
	m.selected += delta
	if m.selected < 0 {
		m.selected = 0
	}
	if m.selected >= len(m.visible) {
		m.selected = len(m.visible) - 1
	}
}

func (m *Model) refreshVisible() {
	m.visible = filterSteps(m.steps, m.filter, m.search)
	if m.selected >= len(m.visible) {
		m.selected = len(m.visible) - 1
	}
	if m.selected < 0 {
		m.selected = 0
	}
}

func nextFilter(filter FilterMode) FilterMode {
	switch filter {
	case FilterAll:
		return FilterProblems
	case FilterProblems:
		return FilterWarnings
	case FilterWarnings:
		return FilterFailed
	default:
		return FilterAll
	}
}

func filterSteps(steps []dobl.Step, filter FilterMode, search string) []dobl.Step {
	var visible []dobl.Step
	for _, step := range steps {
		if !matchesFilter(step, filter) {
			continue
		}
		if !matchesSearch(step, search) {
			continue
		}
		visible = append(visible, step)
	}
	return visible
}

func matchesFilter(step dobl.Step, filter FilterMode) bool {
	switch filter {
	case FilterProblems:
		return isProblemStep(step)
	case FilterWarnings:
		return step.Status == dobl.EventStatusWarning || step.WarningDetail != "" || step.WarningCount > 0
	case FilterFailed:
		return step.Status == dobl.EventStatusError || step.Status == dobl.EventStatusCanceled || step.ErrorDetail != ""
	default:
		return true
	}
}

func isProblemStep(step dobl.Step) bool {
	return step.Status == dobl.EventStatusError ||
		step.Status == dobl.EventStatusCanceled ||
		step.Status == dobl.EventStatusWarning ||
		step.ErrorDetail != "" ||
		step.WarningDetail != ""
}

func matchesSearch(step dobl.Step, search string) bool {
	query := strings.ToLower(strings.TrimSpace(search))
	if query == "" {
		return true
	}
	fields := []string{
		step.ID,
		string(step.Status),
		string(step.Category),
		step.Name,
		step.DisplayName,
		step.Stage,
		step.Instruction,
		step.ErrorDetail,
		step.WarningDetail,
	}
	fields = append(fields, step.OutputTail...)
	return strings.Contains(strings.ToLower(strings.Join(fields, "\n")), query)
}

func (m Model) View() string {
	width := m.width
	if width <= 0 {
		width = defaultWidth
	}
	height := m.height
	if height <= 0 {
		height = defaultHeight
	}

	header := m.headerView(width)
	help := m.helpView(width)
	bodyHeight := height - lipgloss.Height(header) - lipgloss.Height(help) - 2
	if bodyHeight < 6 {
		bodyHeight = 6
	}

	var body string
	if width >= 82 {
		listWidth := width / 2
		if listWidth < minPaneWidth {
			listWidth = minPaneWidth
		}
		detailWidth := width - listWidth - 2
		body = lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.listView(listWidth, bodyHeight),
			"  ",
			m.detailView(detailWidth, bodyHeight),
		)
	} else {
		body = lipgloss.JoinVertical(
			lipgloss.Left,
			m.listView(width, bodyHeight/2),
			m.detailView(width, bodyHeight-bodyHeight/2),
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, body, help)
}

func (m Model) headerView(width int) string {
	stats := collectStats(m.steps)
	filterText := fmt.Sprintf("Filter: %s", m.filter)
	searchText := "Search: -"
	if m.search != "" {
		searchText = fmt.Sprintf("Search: %q", m.search)
	}
	if m.searching {
		searchText += "_"
	}

	line := fmt.Sprintf(
		"Dobl TUI  Source: %s\nSteps: %d  Done: %d  Cached: %d  Warnings: %d  Errors: %d  Canceled: %d  Outputs: %d  %s  %s",
		m.source,
		stats.Total,
		stats.Done,
		stats.Cached,
		stats.Warnings,
		stats.Errors,
		stats.Canceled,
		stats.Outputs,
		filterText,
		searchText,
	)
	return trimBlock(line, width)
}

func (m Model) listView(width int, height int) string {
	lines := []string{"Steps"}
	if len(m.visible) == 0 {
		lines = append(lines, "(none)")
		return padBlock(lines, width, height)
	}

	start := selectedWindowStart(m.selected, height-1, len(m.visible))
	end := start + height - 1
	if end > len(m.visible) {
		end = len(m.visible)
	}
	for i := start; i < end; i++ {
		step := m.visible[i]
		prefix := "  "
		if i == m.selected {
			prefix = "> "
		}
		line := fmt.Sprintf("%s%-4s %-8s %-8s %s", prefix, step.ID, statusText(step.Status), stepLabel(step), step.DisplayName)
		if i == m.selected {
			line = selectedStyle.Render(line)
		}
		lines = append(lines, line)
	}
	return padBlock(lines, width, height)
}

func (m Model) detailView(width int, height int) string {
	lines := []string{"Details"}
	if len(m.visible) == 0 {
		lines = append(lines, "(none)")
		return padBlock(lines, width, height)
	}

	step := m.visible[m.selected]
	lines = append(lines,
		fmt.Sprintf("%s %s", step.ID, statusText(step.Status)),
		fmt.Sprintf("Step: %s", firstNonEmpty(step.DisplayName, step.Name, "(unnamed)")),
		fmt.Sprintf("Category: %s", firstNonEmpty(string(step.Category), "other")),
	)
	if step.Duration != "" {
		lines = append(lines, fmt.Sprintf("Duration: %s", step.Duration))
	}
	if step.Instruction != "" {
		lines = append(lines, fmt.Sprintf("Instruction: %s", step.Instruction))
	}
	if index := formatStepIndex(step); index != "" {
		lines = append(lines, fmt.Sprintf("Dockerfile: %s", index))
	}
	if linesText := lineRange(step); linesText != "" {
		lines = append(lines, fmt.Sprintf("Lines: %s", linesText))
	}
	if step.ErrorDetail != "" {
		lines = append(lines, "", "Error:", step.ErrorDetail)
	}
	if step.WarningDetail != "" {
		lines = append(lines, "", "Warning:", step.WarningDetail)
	}
	if len(step.OutputTail) > 0 {
		lines = append(lines, "", "Output tail:")
		lines = append(lines, step.OutputTail...)
	}
	return padBlock(lines, width, height)
}

func (m Model) helpView(width int) string {
	mode := "j/k move  f filter  / search  esc clear  q quit"
	if m.searching {
		mode = "type to search  enter apply  esc close  ctrl+c quit"
	}
	return trimBlock(mode, width)
}

func selectedWindowStart(selected int, size int, total int) int {
	if size <= 0 || total <= size {
		return 0
	}
	start := selected - size/2
	if start < 0 {
		return 0
	}
	if start+size > total {
		return total - size
	}
	return start
}

func padBlock(lines []string, width int, height int) string {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	for i := range lines {
		lines[i] = trimLine(lines[i], width)
	}
	return strings.Join(lines, "\n")
}

func trimBlock(value string, width int) string {
	lines := strings.Split(value, "\n")
	for i := range lines {
		lines[i] = trimLine(lines[i], width)
	}
	return strings.Join(lines, "\n")
}

func trimLine(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(value) <= width {
		return value
	}
	runes := []rune(value)
	for len(runes) > 0 && lipgloss.Width(string(runes)) > width-3 {
		runes = runes[:len(runes)-1]
	}
	if width <= 3 {
		return string(runes)
	}
	return string(runes) + "..."
}

type stats struct {
	Total    int
	Done     int
	Cached   int
	Warnings int
	Errors   int
	Canceled int
	Outputs  int
}

func collectStats(steps []dobl.Step) stats {
	stats := stats{Total: len(steps)}
	for _, step := range steps {
		switch step.Status {
		case dobl.EventStatusDone:
			stats.Done++
		case dobl.EventStatusCached:
			stats.Cached++
		case dobl.EventStatusWarning:
			stats.Warnings++
		case dobl.EventStatusError:
			stats.Errors++
		case dobl.EventStatusCanceled:
			stats.Canceled++
		}
		stats.Outputs += step.OutputCount
	}
	return stats
}

func statusText(status dobl.EventStatus) string {
	if status == "" {
		return "UNKNOWN"
	}
	return string(status)
}

func stepLabel(step dobl.Step) string {
	if step.Instruction != "" {
		return step.Instruction
	}
	if step.Category != "" {
		return string(step.Category)
	}
	return "other"
}

func formatStepIndex(step dobl.Step) string {
	if step.Index == 0 || step.Total == 0 {
		return ""
	}
	index := fmt.Sprintf("%d/%d", step.Index, step.Total)
	if step.Stage == "" {
		return index
	}
	return strings.Join([]string{step.Stage, index}, " ")
}

func lineRange(step dobl.Step) string {
	if step.StartLine == 0 && step.EndLine == 0 {
		return ""
	}
	if step.StartLine == step.EndLine || step.EndLine == 0 {
		return fmt.Sprintf("%d", step.StartLine)
	}
	return fmt.Sprintf("%d-%d", step.StartLine, step.EndLine)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func (f FilterMode) String() string {
	switch f {
	case FilterProblems:
		return "problems"
	case FilterWarnings:
		return "warnings"
	case FilterFailed:
		return "failed"
	default:
		return "all"
	}
}

var selectedStyle = lipgloss.NewStyle().Bold(true)
