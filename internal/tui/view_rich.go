package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lusingander/dobl"
)

func (m Model) richView() string {
	width := m.width
	if width <= 0 {
		width = defaultWidth
	}
	height := m.height
	if height <= 0 {
		height = defaultHeight
	}

	header := m.richHeaderView(width)
	timeline := m.richTimelineView(width)
	help := m.richHelpView(width)
	bodyHeight := height - lipgloss.Height(header) - lipgloss.Height(timeline) - lipgloss.Height(help) - 3
	if bodyHeight < 8 {
		bodyHeight = 8
	}

	var body string
	if width >= 86 {
		listWidth, detailWidth := paneWidths(width)
		body = lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.richListView(listWidth, bodyHeight),
			"  ",
			m.richDetailView(detailWidth, bodyHeight),
		)
	} else {
		topHeight := bodyHeight / 2
		body = lipgloss.JoinVertical(
			lipgloss.Left,
			m.richListView(width, topHeight),
			m.richDetailView(width, bodyHeight-topHeight),
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, timeline, body, help)
}

func (m Model) richHeaderView(width int) string {
	stats := collectStats(m.steps)
	problems := stats.Errors + stats.Canceled + stats.Warnings
	title := richTitleStyle.Render("Dobl TUI")
	source := richMutedStyle.Render(trimLine(m.source, width-lipgloss.Width("Dobl TUI  ")))
	metrics := fmt.Sprintf(
		"Steps %d  OK %d  Cached %d  Problems %d  Outputs %d",
		stats.Total,
		stats.Done,
		stats.Cached,
		problems,
		stats.Outputs,
	)
	if problems > 0 {
		metrics = fmt.Sprintf("%s  (x%d !%d -%d)", metrics, stats.Errors, stats.Warnings, stats.Canceled)
	}
	scope := fmt.Sprintf("Filter %s", m.filter)
	if m.search != "" {
		scope += fmt.Sprintf("  Search %q", m.search)
	} else {
		scope += "  Search -"
	}
	if m.searching {
		scope += "_"
	}

	lines := []string{
		trimLine(title+"  "+source, width),
		trimLine(richHeaderMetricStyle.Render(metrics), width),
		trimLine(richMutedStyle.Render(scope), width),
	}
	return strings.Join(lines, "\n")
}

func (m Model) richTimelineView(width int) string {
	if len(m.steps) == 0 {
		return richMutedStyle.Render(trimLine("Timeline: (none)", width))
	}

	selectedID := ""
	if len(m.visible) > 0 {
		selectedID = m.visible[m.selected].ID
	}

	rawItems := make([]string, 0, len(m.steps))
	selectedIndex := 0
	for _, step := range m.steps {
		item := fmt.Sprintf("%s%s", step.ID, statusShort(step.Status))
		if isProblemStep(step) {
			item = problemMarker(step) + item
		}
		if step.ID == selectedID {
			item = "[" + item + "]"
			selectedIndex = len(rawItems)
		}
		rawItems = append(rawItems, item)
	}

	rawLine := "Timeline: " + strings.Join(rawItems, " ")
	if lipgloss.Width(rawLine) > width {
		rawLine = trimLine("Timeline: "+timelineWindow(rawItems, selectedIndex, width-lipgloss.Width("Timeline: ")), width)
	}

	parts := strings.Split(rawLine, " ")
	for i := 1; i < len(parts); i++ {
		parts[i] = m.richTimelineItem(parts[i])
	}
	return trimLine(richMutedStyle.Render(parts[0])+" "+strings.Join(parts[1:], " "), width)
}

func (m Model) richTimelineItem(item string) string {
	selected := strings.HasPrefix(item, "[") && strings.HasSuffix(item, "]")
	raw := strings.Trim(item, "[]")
	stepID := strings.TrimLeft(raw, "x!-")
	stepID = strings.TrimRight(stepID, "DCEXWP?")
	for _, step := range m.steps {
		if step.ID == stepID {
			rendered := richStatusStyle(step.Status).Render(raw)
			if selected {
				return richSelectedRowStyle.Render(" " + rendered + " ")
			}
			return rendered
		}
	}
	return item
}

func (m Model) richListView(width int, height int) string {
	innerWidth, innerHeight := paneInnerSize(width, height)
	title := fmt.Sprintf("Steps (%d/%d)", len(m.visible), len(m.steps))
	if m.focus == FocusSteps {
		title += " *"
	}
	lines := []string{richTitleStyle.Render(trimLine(title, innerWidth))}
	if len(m.visible) == 0 {
		lines = append(lines, richMutedStyle.Render(trimLine(m.emptyMessage(), innerWidth)))
		return renderRichPane(lines, width, height, m.focus == FocusSteps)
	}

	start := selectedWindowStart(m.selected, innerHeight-1, len(m.visible))
	end := start + innerHeight - 1
	if end > len(m.visible) {
		end = len(m.visible)
	}
	for i := start; i < end; i++ {
		lines = append(lines, richStepListLine(m.visible[i], innerWidth, i == m.selected))
	}
	return renderRichPane(lines, width, height, m.focus == FocusSteps)
}

func richStepListLine(step dobl.Step, width int, selected bool) string {
	pointer := " "
	if selected {
		pointer = ">"
	}
	marker := richStatusStyle(step.Status).Render(statusMarker(step))
	if width < 44 {
		prefix := fmt.Sprintf("%s %s %-4s %s %-7s ", pointer, marker, step.ID, richStatusStyle(step.Status).Render(statusShort(step.Status)), stepLabel(step))
		line := prefix + trimLine(step.DisplayName, width-lipgloss.Width(prefix))
		return renderRichRow(line, width, selected)
	}

	prefix := fmt.Sprintf("%s %s %-4s %-8s %-8s ", pointer, marker, step.ID, richStatusStyle(step.Status).Render(statusText(step.Status)), stepLabel(step))
	line := prefix + trimLine(step.DisplayName, width-lipgloss.Width(prefix))
	return renderRichRow(line, width, selected)
}

func renderRichRow(line string, width int, selected bool) string {
	line = padLine(line, width)
	if selected {
		return richSelectedRowStyle.Width(width).Render(line)
	}
	return line
}

func (m Model) richDetailView(width int, height int) string {
	innerWidth, innerHeight := paneInnerSize(width, height)
	if len(m.visible) == 0 {
		lines := []string{richTitleStyle.Render(trimLine(m.detailTitle(), innerWidth)), richMutedStyle.Render(trimLine(m.emptyMessage(), innerWidth))}
		return renderRichPane(lines, width, height, m.focus == FocusDetails)
	}

	step := m.visible[m.selected]
	lines := detailLines(step)
	lines[0] = m.detailTitle()
	maxTop := len(lines) - innerHeight
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

	visible := lines[top:]
	for i := range visible {
		visible[i] = richDetailLine(visible[i], innerWidth)
	}
	return renderRichPane(visible, width, height, m.focus == FocusDetails)
}

func richDetailLine(line string, width int) string {
	if strings.HasPrefix(line, "-- ") {
		return richSectionStyle.Render(trimLine(line, width))
	}
	if strings.HasPrefix(line, "Details") {
		return richTitleStyle.Render(trimLine(line, width))
	}
	if strings.Contains(line, " ERROR") {
		return richStatusStyle(dobl.EventStatusError).Render(trimLine(line, width))
	}
	if strings.Contains(line, " WARNING") {
		return richStatusStyle(dobl.EventStatusWarning).Render(trimLine(line, width))
	}
	return trimLine(line, width)
}

func (m Model) richHelpView(width int) string {
	return richHelpStyle.Render(m.helpView(width))
}

func paneInnerSize(width int, height int) (int, int) {
	innerWidth := width - 4
	if innerWidth < 1 {
		innerWidth = 1
	}
	innerHeight := height - 2
	if innerHeight < 1 {
		innerHeight = 1
	}
	return innerWidth, innerHeight
}

func renderRichPane(lines []string, width int, height int, active bool) string {
	innerWidth, innerHeight := paneInnerSize(width, height)
	body := padStyledBlock(lines, innerWidth, innerHeight)
	style := richPaneStyle
	if active {
		style = richActivePaneStyle
	}
	return style.Width(width - 2).Height(height - 2).Render(body)
}

func padStyledBlock(lines []string, width int, height int) string {
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
		lines[i] = padLine(trimLine(lines[i], width), width)
	}
	return strings.Join(lines, "\n")
}

func padLine(line string, width int) string {
	if width < 1 {
		return ""
	}
	lineWidth := lipgloss.Width(line)
	if lineWidth >= width {
		return line
	}
	return line + strings.Repeat(" ", width-lineWidth)
}
