package tui

import (
	"fmt"
	"strings"
)

type ViewMode string

const (
	ViewClassic ViewMode = "classic"
	ViewRich    ViewMode = "rich"
)

func ParseViewMode(value string) (ViewMode, error) {
	mode := normalizeViewMode(ViewMode(strings.ToLower(strings.TrimSpace(value))))
	switch mode {
	case ViewClassic, ViewRich:
		return mode, nil
	default:
		return "", fmt.Errorf("unknown view %q", value)
	}
}

func normalizeViewMode(mode ViewMode) ViewMode {
	if mode == "" {
		return ViewClassic
	}
	return mode
}
