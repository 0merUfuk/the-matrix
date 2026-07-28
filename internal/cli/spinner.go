// Package cli provides shared terminal output formatting for all the-matrix tools.
// spinner.go wraps a bubbletea spinner for use during blocking operations.
package cli

import (
	"errors"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/spinner"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/term"
)

// doneMsg signals that the spinner's background work has completed.
type doneMsg struct {
	err error
}

// spinnerModel implements tea.Model to show a spinner while work runs.
type spinnerModel struct {
	spinner spinner.Model
	title   string
	fn      func() error
	done    bool
	err     error
}

func (m spinnerModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.runWork)
}

// runWork executes the user's function and returns a doneMsg.
func (m spinnerModel) runWork() tea.Msg {
	err := m.fn()
	return doneMsg{err: err}
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		k := msg.Key()
		if k.Mod == tea.ModCtrl && k.Code == 'c' {
			m.done = true
			m.err = errors.New("interrupted")
			return m, tea.Quit
		}
	case doneMsg:
		m.done = true
		m.err = msg.err
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m spinnerModel) View() tea.View {
	if m.done {
		return tea.NewView("")
	}
	return tea.NewView(fmt.Sprintf("  %s %s", m.spinner.View(), m.title))
}

// WithSpinner runs fn while displaying a spinner animation. If stdout is not
// a TTY (piped output, CI environment), it skips the spinner and prints the
// title with a "done"/"error" suffix instead.
func WithSpinner(title string, fn func() error) error {
	// Skip spinner for non-interactive environments.
	// Check stderr since the spinner renders to stderr (stdout may be piped).
	if !term.IsTerminal(os.Stderr.Fd()) || os.Getenv("CI") != "" {
		fmt.Fprintf(os.Stderr, "  %s ...", title)
		err := fn()
		if err != nil {
			fmt.Fprintln(os.Stderr, " error")
		} else {
			fmt.Fprintln(os.Stderr, " done")
		}
		return err
	}

	s := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("#00D787"))),
	)

	model := spinnerModel{
		spinner: s,
		title:   title,
		fn:      fn,
	}

	p := tea.NewProgram(model, tea.WithOutput(os.Stderr))
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("spinner program error: %w", err)
	}

	if sm, ok := finalModel.(spinnerModel); ok && sm.err != nil {
		return sm.err
	}
	return nil
}
