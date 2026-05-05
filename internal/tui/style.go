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
	richSelectedRowStyle  = lipgloss.NewStyle().Background(lipgloss.Color("241")).Foreground(lipgloss.Color("255"))
	richPaneStyle         = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("238")).Padding(0, 1)
	richActivePaneStyle   = richPaneStyle.Copy().BorderForeground(lipgloss.Color("86"))
	richHeaderStyle       = lipgloss.NewStyle().Padding(0, 1)
	richHeaderMetricStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	richTimelineStyle     = lipgloss.NewStyle()
	richTimelineSelected  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("16")).Background(lipgloss.Color("86"))
	richLogLineStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
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

func richStatusBadgeStyle(status dobl.EventStatus) lipgloss.Style {
	switch status {
	case dobl.EventStatusError:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("210")).Background(lipgloss.Color("52"))
	case dobl.EventStatusWarning:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).Background(lipgloss.Color("58"))
	case dobl.EventStatusCanceled:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("219")).Background(lipgloss.Color("53"))
	case dobl.EventStatusCached:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("123")).Background(lipgloss.Color("24"))
	case dobl.EventStatusDone:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("157")).Background(lipgloss.Color("22"))
	case dobl.EventStatusProgress:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("153")).Background(lipgloss.Color("24"))
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Background(lipgloss.Color("238"))
	}
}

func richProblemBadgeStyle(hasProblems bool) lipgloss.Style {
	if hasProblems {
		return richStatusBadgeStyle(dobl.EventStatusError)
	}
	return richStatusBadgeStyle(dobl.EventStatusDone)
}
