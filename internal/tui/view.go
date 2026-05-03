package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
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

var selectedStyle = lipgloss.NewStyle().Bold(true)
