package compute

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"ostui/internal/client"
)

// limitRow holds raw data for one quota entry.
type limitRow struct {
	name  string
	used  int
	total int
	pct   float64
}

// LimitsModel displays quota usage for compute and volume services.
type LimitsModel struct {
	rows    []limitRow
	loading bool
	err     error
	spinner spinner.Model
	client  client.LimitsClient
	width   int
}

type limitsDataLoadedMsg struct {
	rows []limitRow
	err  error
}

// NewLimitsModel creates a new LimitsModel.
func NewLimitsModel(lc client.LimitsClient) LimitsModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return LimitsModel{client: lc, loading: true, spinner: s}
}

// colorForPct returns a lipgloss color based on usage percentage.
func colorForPct(pct float64) lipgloss.Color {
	if pct < 60 {
		return lipgloss.Color("#5CB85C") // green
	} else if pct < 85 {
		return lipgloss.Color("#F0AD4E") // yellow
	}
	return lipgloss.Color("#D9534F") // red
}

// renderBar creates a colored bar of length 20.
func renderBar(pct float64) string {
	const barLen = 20
	filled := int(pct / 100 * float64(barLen))
	if filled > barLen {
		filled = barLen
	}
	empty := barLen - filled
	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	return lipgloss.NewStyle().Foreground(colorForPct(pct)).Render(bar)
}

// Init fetches limits data.
func (m LimitsModel) Init() tea.Cmd {
	return func() tea.Msg {
		limits, err := m.client.GetLimits(context.Background())
		if err != nil {
			return limitsDataLoadedMsg{err: err}
		}

		var rows []limitRow

		add := func(name string, used, max int) {
			pct := 0.0
			if max > 0 {
				pct = float64(used) / float64(max) * 100
			}
			rows = append(rows, limitRow{name: name, used: used, total: max, pct: pct})
		}

		if limits.Compute != nil {
			c := limits.Compute.Absolute
			add("Instances", c.TotalInstancesUsed, c.MaxTotalInstances)
			add("vCPUs", c.TotalCoresUsed, c.MaxTotalCores)
			add("RAM (MiB)", c.TotalRAMUsed, c.MaxTotalRAMSize)
			add("Floating IPs", c.TotalFloatingIpsUsed, c.MaxTotalFloatingIps)
		}

		if limits.Volume != nil {
			v := limits.Volume.Absolute
			add("Volumes", v.TotalVolumesUsed, v.MaxTotalVolumes)
			add("Volume GB", v.TotalGigabytesUsed, v.MaxTotalVolumeGigabytes)
			add("Snapshots", v.TotalSnapshotsUsed, v.MaxTotalSnapshots)
			add("Backup GB", v.TotalBackupGigabytesUsed, v.MaxTotalBackupGigabytes)
		}

		return limitsDataLoadedMsg{rows: rows}
	}
}

// Update handles messages.
func (m LimitsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case limitsDataLoadedMsg:
		m.loading = false
		m.err = msg.err
		m.rows = msg.rows
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case tea.KeyMsg:
		return m, nil
	default:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

// View renders the limits view with colored bars.
func (m LimitsModel) View() string {
	if m.loading {
		return m.spinner.View()
	}
	if m.err != nil {
		return fmt.Sprintf("Error loading limits: %s", m.err)
	}
	if len(m.rows) == 0 {
		return "No quota data available."
	}

	width := m.width
	if width == 0 {
		width = 80
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#AAAAAA"))
	nameStyle := lipgloss.NewStyle().Width(16)
	separator := strings.Repeat("─", width)

	var sb strings.Builder
	sb.WriteString(headerStyle.Render(fmt.Sprintf("%-16s  %-22s  %12s  %6s", "Resource", "Usage", "Used/Total", "Pct")) + "\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#444444")).Render(separator) + "\n")

	for _, r := range m.rows {
		color := colorForPct(r.pct)
		valueStyle := lipgloss.NewStyle().Foreground(color)

		bar := renderBar(r.pct)
		usedTotal := fmt.Sprintf("%d/%d", r.used, r.total)
		pctStr := fmt.Sprintf("%.0f%%", r.pct)

		line := fmt.Sprintf("%s  %s  %12s  %6s",
			nameStyle.Render(r.name),
			bar,
			valueStyle.Render(usedTotal),
			valueStyle.Render(pctStr),
		)
		sb.WriteString(line + "\n")
	}

	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#444444")).Render(separator) + "\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render("[esc] back") + "\n")

	return sb.String()
}

var _ tea.Model = (*LimitsModel)(nil)
