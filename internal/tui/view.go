package tui

func (m Model) View() string {
	switch m.viewMode {
	case ViewRich:
		return m.richView()
	default:
		return m.classicView()
	}
}
