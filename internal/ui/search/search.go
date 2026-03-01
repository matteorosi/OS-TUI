package search

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/sync/errgroup"
	"ostui/internal/client"
)

// SearchResult represents a single search result.
type SearchResult struct {
	Category string // "Servers", "Networks", "Volumes", etc.
	ID       string
	Name     string
	Extra    string // additional info: status, size, CIDR, etc.
}

// Messages used by the SearchModel.
type searchResultsMsg struct {
	results []SearchResult
	err     error
}

type searchQueryMsg struct {
	query string
}

type SearchDoneMsg struct{}

type SearchSelectedMsg struct {
	Result SearchResult
}

// SearchModel holds the state for the global search UI.
type SearchModel struct {
	input         textinput.Model
	results       []SearchResult
	cursor        int
	loading       bool
	err           error
	spinner       spinner.Model
	query         string // last executed query
	width         int
	height        int
	computeClient client.ComputeClient
	networkClient client.NetworkClient
	storageClient client.StorageClient
	imageClient   client.ImageClient
}

// NewSearchModel creates a new SearchModel.
func NewSearchModel(cc client.ComputeClient, nc client.NetworkClient, sc client.StorageClient, ic client.ImageClient, w, h int) SearchModel {
	ti := textinput.New()
	ti.Placeholder = "search"
	ti.Focus()
	sp := spinner.New()
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return SearchModel{
		input:         ti,
		spinner:       sp,
		width:         w,
		height:        h,
		computeClient: cc,
		networkClient: nc,
		storageClient: sc,
		imageClient:   ic,
	}
}

// Init focuses the text input and starts the spinner.
func (m SearchModel) Init() tea.Cmd {
	// Start spinner tick and blink cursor.
	return tea.Batch(textinput.Blink, spinner.Tick)
}

// Update handles messages for the search model.
func (m SearchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			// Signal closing of search.
			return m, func() tea.Msg { return SearchDoneMsg{} }
		case "enter":
			if m.cursor >= 0 && m.cursor < len(m.results) {
				return m, func() tea.Msg { return SearchSelectedMsg{Result: m.results[m.cursor]} }
			}
			return m, nil
		case "j", "down":
			if m.cursor < len(m.results)-1 {
				m.cursor++
			}
			return m, nil
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		default:
			// Forward to textinput.
			oldVal := m.input.Value()
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			cmds = append(cmds, cmd)
			newVal := m.input.Value()
			if newVal != oldVal {
				// Reset cursor and schedule debounce.
				m.cursor = 0
				m.loading = true
				// Debounce after 300ms.
				cmds = append(cmds, tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg {
					return searchQueryMsg{query: newVal}
				}))
			}
			return m, tea.Batch(cmds...)
		}
	case searchQueryMsg:
		// Only fire if the query hasn't changed during debounce.
		if msg.query == m.input.Value() {
			m.query = msg.query
			// Trigger live search.
			return m, m.searchCmd(msg.query)
		}
		// Query changed, ignore.
		return m, nil
	case searchResultsMsg:
		m.results = msg.results
		m.loading = false
		m.err = msg.err
		m.cursor = 0
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

// searchCmd performs the parallel live search across OpenStack services.
func (m SearchModel) searchCmd(query string) tea.Cmd {
	return func() tea.Msg {
		if strings.TrimSpace(query) == "" {
			return searchResultsMsg{results: nil}
		}
		q := strings.ToLower(query)
		var mu sync.Mutex
		var allResults []SearchResult
		var g errgroup.Group

		// Servers
		g.Go(func() error {
			srvList, err := m.computeClient.ListInstances()
			if err != nil {
				return nil
			}
			for _, s := range srvList {
				if strings.Contains(strings.ToLower(s.Name), q) || strings.Contains(strings.ToLower(s.ID), q) {
					mu.Lock()
					allResults = append(allResults, SearchResult{Category: "Servers", ID: s.ID, Name: s.Name, Extra: s.Status})
					mu.Unlock()
				}
			}
			return nil
		})

		// Networks
		g.Go(func() error {
			netList, err := m.networkClient.ListNetworks()
			if err != nil {
				return nil
			}
			for _, n := range netList {
				if strings.Contains(strings.ToLower(n.Name), q) || strings.Contains(strings.ToLower(n.ID), q) {
					mu.Lock()
					allResults = append(allResults, SearchResult{Category: "Networks", ID: n.ID, Name: n.Name, Extra: n.Status})
					mu.Unlock()
				}
			}
			return nil
		})

		// Volumes
		g.Go(func() error {
			volList, err := m.storageClient.ListVolumes()
			if err != nil {
				return nil
			}
			for _, v := range volList {
				if strings.Contains(strings.ToLower(v.Name), q) || strings.Contains(strings.ToLower(v.ID), q) {
					mu.Lock()
					allResults = append(allResults, SearchResult{Category: "Volumes", ID: v.ID, Name: v.Name, Extra: fmt.Sprintf("%dGB %s", v.Size, v.Status)})
					mu.Unlock()
				}
			}
			return nil
		})

		// Floating IPs
		g.Go(func() error {
			fipList, err := m.networkClient.ListFloatingIPs()
			if err != nil {
				return nil
			}
			for _, f := range fipList {
				if strings.Contains(strings.ToLower(f.FloatingIP), q) || strings.Contains(strings.ToLower(f.ID), q) {
					mu.Lock()
					allResults = append(allResults, SearchResult{Category: "Floating IPs", ID: f.ID, Name: f.FloatingIP, Extra: f.Status})
					mu.Unlock()
				}
			}
			return nil
		})

		// Routers
		g.Go(func() error {
			ctx := context.Background()
			routerList, err := m.networkClient.ListRouters(ctx)
			if err != nil {
				return nil
			}
			for _, r := range routerList {
				if strings.Contains(strings.ToLower(r.Name), q) || strings.Contains(strings.ToLower(r.ID), q) {
					mu.Lock()
					allResults = append(allResults, SearchResult{Category: "Routers", ID: r.ID, Name: r.Name, Extra: r.Status})
					mu.Unlock()
				}
			}
			return nil
		})

		// Subnets
		g.Go(func() error {
			subList, err := m.networkClient.ListSubnets()
			if err != nil {
				return nil
			}
			for _, s := range subList {
				if strings.Contains(strings.ToLower(s.Name), q) || strings.Contains(strings.ToLower(s.ID), q) || strings.Contains(s.CIDR, q) {
					mu.Lock()
					allResults = append(allResults, SearchResult{Category: "Subnets", ID: s.ID, Name: s.Name, Extra: s.CIDR})
					mu.Unlock()
				}
			}
			return nil
		})

		// Wait for all goroutines.
		_ = g.Wait()

		// Sort by Category then Name.
		sort.Slice(allResults, func(i, j int) bool {
			if allResults[i].Category != allResults[j].Category {
				return allResults[i].Category < allResults[j].Category
			}
			return allResults[i].Name < allResults[j].Name
		})

		return searchResultsMsg{results: allResults}
	}
}

// View renders the search UI.
func (m SearchModel) View() string {
	// Header
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	var b strings.Builder
	b.WriteString(headerStyle.Render("Global Search"))
	b.WriteString("\n")

	// Input line with optional spinner.
	if m.loading {
		b.WriteString(m.spinner.View())
		b.WriteString(" ")
	}
	b.WriteString(m.input.View())
	b.WriteString("\n\n")

	// Content
	if m.loading {
		// Show nothing else while loading.
	} else if len(m.results) == 0 && strings.TrimSpace(m.query) != "" {
		b.WriteString(fmt.Sprintf("No results for '%s'", m.query))
	} else if len(m.results) > 0 {
		// Group results by category.
		groups := make(map[string][]SearchResult)
		order := []string{}
		for _, r := range m.results {
			if _, ok := groups[r.Category]; !ok {
				order = append(order, r.Category)
			}
			groups[r.Category] = append(groups[r.Category], r)
		}
		// Ensure deterministic order.
		sort.Strings(order)
		idx := 0
		for _, cat := range order {
			items := groups[cat]
			// Category header
			catHeader := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
			b.WriteString(catHeader.Render(fmt.Sprintf("%s (%d)", cat, len(items))))
			b.WriteString("\n")
			for _, res := range items {
				// Build line.
				extraStyled := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(res.Extra)
				line := fmt.Sprintf("%s  %s", res.Name, extraStyled)
				if idx == m.cursor {
					// Highlight selected line.
					line = lipgloss.NewStyle().Background(lipgloss.Color("236")).Render(line)
				}
				b.WriteString(line)
				b.WriteString("\n")
				idx++
			}
			b.WriteString("\n")
		}
	}

	// Wrap with border.
	border := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("205"))
	return border.Render(b.String())
}

// Ensure SearchModel implements tea.Model.
var _ tea.Model = (*SearchModel)(nil)
