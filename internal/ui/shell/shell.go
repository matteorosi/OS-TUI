package shell

import (
	"bytes"
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"os"
	"os/exec"
)

type ShellModel struct {
	command  string
	cloud    string
	loading  bool
	err      error
	output   string
	viewport viewport.Model
	spinner  spinner.Model
}

type shellOutputMsg struct {
	output string
	err    error
}

func NewShellModel(cloud, command string) ShellModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return ShellModel{cloud: cloud, command: command, loading: true, spinner: s, viewport: viewport.New(80, 24)}
}

func (m ShellModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, func() tea.Msg {
		cmd := exec.Command("/bin/sh", "-c", "openstack "+m.command)
		cmd.Env = append(os.Environ(), "OS_CLOUD="+m.cloud)
		var out bytes.Buffer
		var errOut bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &errOut
		err := cmd.Run()
		if err != nil {
			return shellOutputMsg{output: errOut.String(), err: fmt.Errorf("%s", errOut.String())}
		}
		return shellOutputMsg{output: out.String()}
	})
}

func (m ShellModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case shellOutputMsg:
		m.loading = false
		m.output = msg.output
		m.viewport.SetContent(m.output)
		return m, nil
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 3
		m.viewport.SetContent(m.output)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return m, func() tea.Msg { return CloseMsg{} }
		default:
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
	default:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m ShellModel) View() string {
	if m.loading {
		return m.spinner.View() + " Running: openstack " + m.command
	}
	header := fmt.Sprintf("openstack %s", m.command)
	footer := fmt.Sprintf(" %3.f%% | [j/k] scroll  [esc] close", m.viewport.ScrollPercent()*100)
	return header + "\n" + m.viewport.View() + "\n" + footer
}

type CloseMsg struct{}

var _ tea.Model = (*ShellModel)(nil)
