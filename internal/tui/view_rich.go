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
	innerWidth := width - 2
	if innerWidth < 1 {
		innerWidth = 1
	}
	title := richTitleStyle.Render("Dobl TUI") + "  " + richMutedStyle.Render(trimLine(m.source, innerWidth-lipgloss.Width("Dobl TUI  ")))
	metrics := strings.Join([]string{
		richMetricBadge("Steps", stats.Total, richHeaderMetricStyle),
		richMetricBadge("OK", stats.Done, richStatusBadgeStyle(dobl.EventStatusDone)),
		richMetricBadge("Cached", stats.Cached, richStatusBadgeStyle(dobl.EventStatusCached)),
		richMetricBadge("Problems", problems, richProblemBadgeStyle(problems > 0)),
		richMetricBadge("Outputs", stats.Outputs, lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Background(lipgloss.Color("238"))),
	}, " ")
	if problems > 0 {
		metrics += " " + richMutedStyle.Render(fmt.Sprintf("x%d !%d -%d", stats.Errors, stats.Warnings, stats.Canceled))
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
		trimLine(title, innerWidth),
		trimLine(metrics, innerWidth),
		trimLine(richMutedStyle.Render(scope), innerWidth),
	}
	return richHeaderStyle.Width(width - 2).Render(padStyledBlock(lines, innerWidth, len(lines)))
}

func (m Model) richTimelineView(width int) string {
	if len(m.steps) == 0 {
		return richMutedStyle.Render(trimLine("Timeline: (none)", width))
	}

	selectedID := ""
	if len(m.visible) > 0 {
		selectedID = m.visible[m.selected].ID
	}

	entries := make([]richTimelineEntry, 0, len(m.steps))
	selectedIndex := 0
	for _, step := range m.steps {
		label := fmt.Sprintf("%s%s", step.ID, statusShort(step.Status))
		if isProblemStep(step) {
			label = problemMarker(step) + label
		}
		if step.ID == selectedID {
			selectedIndex = len(entries)
		}
		entries = append(entries, richTimelineEntry{label: label, status: step.Status})
	}

	available := width - lipgloss.Width("Timeline: ")
	left, right := richTimelineWindow(entries, selectedIndex, available)
	line := richMutedStyle.Render("Timeline:") + " " + richTimelineLine(entries, left, right, selectedIndex)
	return trimLine(richTimelineStyle.Render(line), width)
}

func (m Model) richListView(width int, height int) string {
	innerWidth, innerHeight := paneInnerSize(width, height)
	title := fmt.Sprintf("Steps (%d/%d)", len(m.visible), len(m.steps))
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
	if selected {
		return richSelectedStepListLine(step, width)
	}

	pointer := " "
	marker := richStatusBadge(statusMarker(step), step.Status)
	if width < 44 {
		status := richStatusBadge(statusShort(step.Status), step.Status)
		prefix := fmt.Sprintf("%s %s %-4s %s %-7s ", pointer, marker, step.ID, status, stepLabel(step))
		line := prefix + trimLine(step.DisplayName, width-lipgloss.Width(prefix))
		return renderRichRow(line, width, selected)
	}

	status := richStatusBadge(fixedWidth(statusText(step.Status), 8), step.Status)
	prefix := fmt.Sprintf("%s %s %-4s %s %-8s ", pointer, marker, step.ID, status, stepLabel(step))
	line := prefix + trimLine(step.DisplayName, width-lipgloss.Width(prefix))
	return renderRichRow(line, width, selected)
}

func richSelectedStepListLine(step dobl.Step, width int) string {
	if width < 44 {
		line := fmt.Sprintf("  %-3s %-4s %-3s %-7s %s", statusMarker(step), step.ID, statusShort(step.Status), stepLabel(step), step.DisplayName)
		return richSelectedRowStyle.Width(width).Render(padLine(trimLine(line, width), width))
	}
	line := fmt.Sprintf("  %-3s %-4s %-10s %-8s %s", statusMarker(step), step.ID, fixedWidth(statusText(step.Status), 8), stepLabel(step), step.DisplayName)
	return richSelectedRowStyle.Width(width).Render(padLine(trimLine(line, width), width))
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
		lines := []string{richTitleStyle.Render(trimLine(m.richDetailTitle(), innerWidth)), richMutedStyle.Render(trimLine(m.emptyMessage(), innerWidth))}
		return renderRichPane(lines, width, height, m.focus == FocusDetails)
	}

	step := m.visible[m.selected]
	lines := detailLines(step)
	lines[0] = m.richDetailTitle()
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
	section := ""
	for i := range visible {
		if strings.HasPrefix(visible[i], "-- ") {
			section = strings.TrimPrefix(visible[i], "-- ")
		}
		visible[i] = richDetailLine(visible[i], innerWidth, section)
	}
	return renderRichPane(visible, width, height, m.focus == FocusDetails)
}

func richDetailLine(line string, width int, section string) string {
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
	if section == "Diagnostic" && strings.HasPrefix(line, "  ") {
		status := dobl.EventStatusError
		if strings.HasPrefix(line, "  Warning:") {
			status = dobl.EventStatusWarning
		}
		return richStatusStyle(status).Render(trimLine(line, width))
	}
	if strings.HasPrefix(section, "Output tail") && strings.HasPrefix(line, "  ") {
		return richLogLineStyle.Render(trimLine(line, width))
	}
	return trimLine(line, width)
}

func (m Model) richDetailTitle() string {
	if len(m.visible) == 0 {
		return "Details"
	}
	step := m.visible[m.selected]
	return fmt.Sprintf("Details %s %s", step.ID, statusText(step.Status))
}

func (m Model) richHelpView(width int) string {
	return richHelpStyle.Width(width).Render(padLine(m.helpView(width), width))
}

func richMetricBadge(label string, value int, style lipgloss.Style) string {
	return style.Padding(0, 1).Render(fmt.Sprintf("%s %d", label, value))
}

func richStatusBadge(value string, status dobl.EventStatus) string {
	return richStatusBadgeStyle(status).Padding(0, 1).Render(value)
}

type richTimelineEntry struct {
	label  string
	status dobl.EventStatus
}

func richTimelineWindow(entries []richTimelineEntry, selected int, width int) (int, int) {
	if len(entries) == 0 {
		return 0, -1
	}
	if selected < 0 {
		selected = 0
	}
	if selected >= len(entries) {
		selected = len(entries) - 1
	}

	left := selected
	right := selected
	for {
		expanded := false
		if left > 0 && richTimelineWidth(entries, left-1, right) <= width {
			left--
			expanded = true
		}
		if right < len(entries)-1 && richTimelineWidth(entries, left, right+1) <= width {
			right++
			expanded = true
		}
		if !expanded {
			return left, right
		}
	}
}

func richTimelineWidth(entries []richTimelineEntry, left int, right int) int {
	if len(entries) == 0 || right < left {
		return 0
	}
	width := 0
	items := 0
	if left > 0 {
		width += lipgloss.Width("...")
		items++
	}
	for i := left; i <= right; i++ {
		width += lipgloss.Width(entries[i].label) + 2
		items++
	}
	if right < len(entries)-1 {
		width += lipgloss.Width("...")
		items++
	}
	if items > 1 {
		width += items - 1
	}
	return width
}

func richTimelineLine(entries []richTimelineEntry, left int, right int, selected int) string {
	if len(entries) == 0 || right < left {
		return ""
	}
	items := make([]string, 0, right-left+3)
	if left > 0 {
		items = append(items, richMutedStyle.Render("..."))
	}
	for i := left; i <= right; i++ {
		items = append(items, richTimelineBadge(entries[i], i == selected))
	}
	if right < len(entries)-1 {
		items = append(items, richMutedStyle.Render("..."))
	}
	return strings.Join(items, " ")
}

func richTimelineBadge(entry richTimelineEntry, selected bool) string {
	style := richStatusBadgeStyle(entry.status)
	if selected {
		style = richTimelineSelected
	}
	return style.Padding(0, 1).Render(entry.label)
}

func fixedWidth(value string, width int) string {
	if lipgloss.Width(value) >= width {
		return value
	}
	return value + strings.Repeat(" ", width-lipgloss.Width(value))
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
