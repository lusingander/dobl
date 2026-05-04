package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lusingander/dobl"
)

const minPaneWidth = 24

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
	timeline := m.timelineView(width)
	help := m.helpView(width)
	bodyHeight := height - lipgloss.Height(header) - lipgloss.Height(timeline) - lipgloss.Height(help) - 3
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

	return lipgloss.JoinVertical(lipgloss.Left, header, timeline, body, help)
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

func (m Model) timelineView(width int) string {
	if len(m.steps) == 0 {
		return trimLine("Timeline: (none)", width)
	}

	selectedID := ""
	if len(m.visible) > 0 {
		selectedID = m.visible[m.selected].ID
	}

	items := make([]string, 0, len(m.steps))
	selectedIndex := 0
	for _, step := range m.steps {
		item := fmt.Sprintf("%s%s", step.ID, statusShort(step.Status))
		if isProblemStep(step) {
			item = problemMarker(step) + item
		}
		if step.ID == selectedID {
			item = "[" + item + "]"
			selectedIndex = len(items)
		}
		items = append(items, item)
	}

	line := "Timeline: " + strings.Join(items, " ")
	if lipgloss.Width(line) <= width {
		return line
	}
	return trimLine("Timeline: "+timelineWindow(items, selectedIndex, width-lipgloss.Width("Timeline: ")), width)
}

func timelineWindow(items []string, selected int, width int) string {
	if len(items) == 0 || width <= 0 {
		return ""
	}
	if selected < 0 {
		selected = 0
	}
	if selected >= len(items) {
		selected = len(items) - 1
	}

	left := selected
	right := selected
	best := timelineWindowLine(items, left, right)
	for {
		expanded := false
		if left > 0 {
			candidate := timelineWindowLine(items, left-1, right)
			if lipgloss.Width(candidate) <= width {
				left--
				best = candidate
				expanded = true
			}
		}
		if right < len(items)-1 {
			candidate := timelineWindowLine(items, left, right+1)
			if lipgloss.Width(candidate) <= width {
				right++
				best = candidate
				expanded = true
			}
		}
		if !expanded {
			return best
		}
	}
}

func timelineWindowLine(items []string, left int, right int) string {
	visible := append([]string(nil), items[left:right+1]...)
	if left > 0 {
		visible = append([]string{"..."}, visible...)
	}
	if right < len(items)-1 {
		visible = append(visible, "...")
	}
	return strings.Join(visible, " ")
}

func (m Model) listView(width int, height int) string {
	title := fmt.Sprintf("Steps (%d/%d)", len(m.visible), len(m.steps))
	if m.focus == FocusSteps {
		title += " *"
	}
	lines := []string{title}
	if len(m.visible) == 0 {
		lines = append(lines, m.emptyMessage())
		return padBlock(lines, width, height)
	}

	start := selectedWindowStart(m.selected, height-1, len(m.visible))
	end := start + height - 1
	if end > len(m.visible) {
		end = len(m.visible)
	}
	for i := start; i < end; i++ {
		step := m.visible[i]
		marker := statusMarker(step)
		prefix := fmt.Sprintf("  %s ", marker)
		if i == m.selected {
			prefix = fmt.Sprintf("> %s ", marker)
		}
		line := fmt.Sprintf("%s%-4s %-8s %-8s %s", prefix, step.ID, statusText(step.Status), stepLabel(step), step.DisplayName)
		if i == m.selected {
			line = trimLine(line, width)
			line = selectedStyle.Render(line)
		}
		lines = append(lines, line)
	}
	return padBlock(lines, width, height)
}

func (m Model) detailView(width int, height int) string {
	if len(m.visible) == 0 {
		return padBlock([]string{m.detailTitle(), m.emptyMessage()}, width, height)
	}

	step := m.visible[m.selected]
	lines := detailLines(step)
	lines[0] = m.detailTitle()
	maxTop := len(lines) - height
	if maxTop < 0 {
		maxTop = 0
	}
	top := m.detailTop
	if top > maxTop {
		top = maxTop
	}
	if top > 0 && len(lines) > 0 {
		lines[0] = fmt.Sprintf("Details (+%d)", top)
	}
	return padBlock(lines[top:], width, height)
}

func detailLines(step dobl.Step) []string {
	lines := []string{"Details",
		fmt.Sprintf("%s %s", step.ID, statusText(step.Status)),
		fmt.Sprintf("Step: %s", firstNonEmpty(step.DisplayName, step.Name, "(unnamed)")),
	}

	if step.ErrorDetail != "" || step.WarningDetail != "" {
		lines = append(lines, "", "Diagnostic")
		if step.ErrorDetail != "" {
			lines = append(lines, fmt.Sprintf("Error: %s", step.ErrorDetail))
		}
		if step.WarningDetail != "" {
			lines = append(lines, fmt.Sprintf("Warning: %s", step.WarningDetail))
		}
	}
	if len(step.OutputTail) > 0 {
		lines = append(lines, "", fmt.Sprintf("Output tail (%d)", len(step.OutputTail)))
		lines = append(lines, step.OutputTail...)
	}

	lines = append(lines, "", "Metadata")
	lines = append(lines, fmt.Sprintf("Category: %s", firstNonEmpty(string(step.Category), "other")))
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
	return lines
}

func (m Model) detailTitle() string {
	title := "Details"
	if len(m.visible) > 0 {
		step := m.visible[m.selected]
		title = fmt.Sprintf("Details %s %s", step.ID, statusText(step.Status))
	}
	if m.focus == FocusDetails {
		title += " *"
	}
	return title
}

func (m Model) emptyMessage() string {
	if m.search != "" && m.filter != FilterAll {
		return fmt.Sprintf("No steps match filter %s and search %q", m.filter, m.search)
	}
	if m.search != "" {
		return fmt.Sprintf("No steps match search %q", m.search)
	}
	if m.filter != FilterAll {
		return fmt.Sprintf("No steps match filter %s", m.filter)
	}
	return "(none)"
}

func (m Model) helpView(width int) string {
	mode := "steps: j/k move  enter/right details  n/N problem  f filter  p problems  r reset  / search  q quit"
	if m.focus == FocusDetails {
		mode = "details: j/k scroll  left/backspace steps  pgup/pgdown page  tab focus  / search  q quit"
	}
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

var selectedStyle = lipgloss.NewStyle().Bold(true)
