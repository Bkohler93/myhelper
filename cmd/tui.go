package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type stepMsg string
type doneMsg struct{ err error }

// spinModel implements tea.Model for displaying an animated spinner with a
// current step label. The work function is NOT stored in the model; it is
// launched by RunWithSpinner before p.Run() blocks.
type spinModel struct {
	spinner  spinner.Model
	label    string
	quitting bool
	err      error
}

func (m spinModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m spinModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case stepMsg:
		m.label = string(msg)
		return m, nil
	case doneMsg:
		m.quitting = true
		m.err = msg.err
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m spinModel) View() string {
	if m.quitting && m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}
	if m.quitting {
		return "✓ Done. Wrote .myhelper/\n"
	}
	return m.spinner.View() + " " + m.label + "\n"
}

type updateLabelFn func(string)

var NilUpdateLabelFn = updateLabelFn(func(s string) {})

// RunWithSpinner runs workFn under an animated spinner displayed on stderr.
// workFn receives a progress callback to update the spinner label.
// Returns the error from workFn, if any.
func RunWithSpinner(workFn func(progress updateLabelFn) error) error {
	s := spinner.New(spinner.WithSpinner(spinner.Dot))
	m := spinModel{spinner: s, label: "Starting..."}
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr), tea.WithoutSignalHandler())

	go func() {
		err := workFn(func(label string) {
			p.Send(stepMsg(label))
		})
		p.Send(doneMsg{err: err})
	}()

	final, err := p.Run()
	if err != nil {
		return fmt.Errorf("tui: %w", err)
	}
	if fm, ok := final.(spinModel); ok && fm.err != nil {
		return fm.err
	}
	return nil
}
