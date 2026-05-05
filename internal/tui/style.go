package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/lusingander/dobl"
)

var (
	richTitleStyle        = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	richMutedStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	richHelpStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
	richSectionStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("111"))
	richSelectedRowStyle  = lipgloss.NewStyle().Background(lipgloss.Color("238")).Foreground(lipgloss.Color("252"))
	richPaneStyle         = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("238")).Padding(0, 1)
	richActivePaneStyle   = richPaneStyle.Copy().BorderForeground(lipgloss.Color("86"))
	richHeaderMetricStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
)

func richStatusStyle(status dobl.EventStatus) lipgloss.Style {
	switch status {
	case dobl.EventStatusError:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("203"))
	case dobl.EventStatusWarning:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("221"))
	case dobl.EventStatusCanceled:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("177"))
	case dobl.EventStatusCached:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("81"))
	case dobl.EventStatusDone:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("114"))
	case dobl.EventStatusProgress:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("75"))
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	}
}
