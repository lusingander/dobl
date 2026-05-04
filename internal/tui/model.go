package tui

import (
	"fmt"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lusingander/dobl"
	"github.com/mattn/go-isatty"
)

const (
	defaultWidth  = 100
	defaultHeight = 30
)

type Options struct {
	Source        string
	InitialFilter FilterMode
	InitialSearch string
	Input         io.Reader
	Output        io.Writer
}

type Model struct {
	steps     []dobl.Step
	visible   []dobl.Step
	source    string
	filter    FilterMode
	search    string
	searching bool
	focus     Focus
	selected  int
	detailTop int
	width     int
	height    int
}

type Focus int

const (
	FocusSteps Focus = iota
	FocusDetails
)

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
	if err := validateTerminalOutput(options.Output); err != nil {
		return err
	}

	source := options.Source
	if source == "" {
		source = "stdin"
	}

	model := NewModel(steps, source)
	model.filter = options.InitialFilter
	model.search = options.InitialSearch
	model.refreshVisible()
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

func validateTerminalOutput(output io.Writer) error {
	if os.Getenv("TERM") == "dumb" {
		return fmt.Errorf("tui requires an interactive terminal; TERM=dumb is not supported")
	}

	if output == nil {
		output = os.Stdout
	}
	file, ok := output.(*os.File)
	if !ok {
		return nil
	}
	if isatty.IsTerminal(file.Fd()) {
		return nil
	}
	return fmt.Errorf("tui requires terminal output; redirect summary or report output instead")
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
		return m.updateSearchKey(msg)
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "tab":
		m.toggleFocus()
	case "j", "down":
		if m.focus == FocusDetails {
			m.scrollDetail(1)
		} else {
			m.move(1)
		}
	case "k", "up":
		if m.focus == FocusDetails {
			m.scrollDetail(-1)
		} else {
			m.move(-1)
		}
	case "g", "home":
		m.selected = 0
		m.detailTop = 0
	case "G", "end":
		if len(m.visible) > 0 {
			m.selected = len(m.visible) - 1
		}
		m.detailTop = 0
	case "pgdown", "ctrl+d":
		m.scrollDetail(5)
	case "pgup", "ctrl+u":
		m.scrollDetail(-5)
	case "n":
		m.moveProblem(1)
	case "N":
		m.moveProblem(-1)
	case "f":
		m.filter = nextFilter(m.filter)
		m.refreshVisible()
	case "p":
		m.filter = FilterProblems
		m.refreshVisible()
	case "r":
		m.filter = FilterAll
		m.search = ""
		m.searching = false
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

func (m Model) updateSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searching = false
		if m.search == "" {
			m.refreshVisible()
		}
	case "enter":
		m.searching = false
	case "backspace", "ctrl+h":
		if m.search != "" {
			runes := []rune(m.search)
			m.search = string(runes[:len(runes)-1])
			m.refreshVisible()
		}
	case "ctrl+c":
		return m, tea.Quit
	default:
		if len(msg.Runes) > 0 {
			m.search += string(msg.Runes)
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
	if delta != 0 {
		m.detailTop = 0
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
	m.detailTop = 0
}

func (m *Model) scrollDetail(delta int) {
	if len(m.visible) == 0 {
		m.detailTop = 0
		return
	}
	m.detailTop += delta
	if m.detailTop < 0 {
		m.detailTop = 0
	}
}

func (m *Model) moveProblem(direction int) {
	next := nextProblemIndex(m.visible, m.selected, direction)
	if next == -1 {
		return
	}
	m.selected = next
	m.detailTop = 0
}

func (m *Model) toggleFocus() {
	if m.focus == FocusDetails {
		m.focus = FocusSteps
		return
	}
	m.focus = FocusDetails
}

func (f Focus) String() string {
	if f == FocusDetails {
		return "details"
	}
	return "steps"
}
