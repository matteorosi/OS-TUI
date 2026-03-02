package common

import (
	"fmt"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"strings"
)

// FilterableTable gestisce filter mode + tabella in modo riusabile.
type FilterableTable struct {
	Table      table.Model
	AllRows    []table.Row
	filterMode bool
	filter     textinput.Model
}

// NewFilterableTable crea un FilterableTable.
func NewFilterableTable() FilterableTable {
	ti := textinput.New()
	ti.Placeholder = "filter..."
	return FilterableTable{filter: ti}
}

// SetTable imposta la tabella e le righe iniziali.
func (f *FilterableTable) SetTable(t table.Model, rows []table.Row) {
	f.Table = t
	f.AllRows = rows
}

// IsFiltering ritorna true se il filter mode è attivo.
func (f *FilterableTable) IsFiltering() bool { return f.filterMode }

// Update gestisce i KeyMsg per filter mode. Ritorna (handled bool, cmd tea.Cmd).
// Se handled è true, il chiamante deve fare return senza processare ulteriormente il tasto.
func (f *FilterableTable) Update(msg tea.KeyMsg) (bool, tea.Cmd) {
	if !f.filterMode && msg.String() == "/" {
		f.filterMode = true
		f.filter.Focus()
		return true, textinput.Blink
	}
	if f.filterMode && msg.String() == "esc" {
		f.filterMode = false
		f.filter.Blur()
		f.filter.SetValue("")
		f.Table.SetRows(f.AllRows)
		return true, nil
	}
	if f.filterMode {
		var cmd tea.Cmd
		f.filter, cmd = f.filter.Update(msg)
		filterVal := strings.ToLower(f.filter.Value())
		var filtered []table.Row
		for _, row := range f.AllRows {
			for _, cell := range row {
				if strings.Contains(strings.ToLower(cell), filterVal) {
					filtered = append(filtered, row)
					break
				}
			}
		}
		f.Table.SetRows(filtered)
		return true, cmd
	}
	return false, nil
}

// UpdateTable forwarda msg alla table interna. Da chiamare sempre per scroll/navigazione.
func (f *FilterableTable) UpdateTable(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	f.Table, cmd = f.Table.Update(msg)
	return cmd
}

// View renderizza tabella + filter bar se attiva.
func (f *FilterableTable) View() string {
	if f.filterMode {
		return fmt.Sprintf("Filter: %s\n%s\nesc: clear", f.filter.View(), f.Table.View())
	}
	return f.Table.View()
}

// SetHeight imposta l'altezza della tabella interna.
func (f *FilterableTable) SetHeight(h int) {
	f.Table.SetHeight(h)
}

// SetColumns imposta le colonne della tabella interna.
func (f *FilterableTable) SetColumns(cols []table.Column) {
	f.Table.SetColumns(cols)
}

// SelectedRow ritorna la riga selezionata.
func (f *FilterableTable) SelectedRow() table.Row {
	return f.Table.SelectedRow()
}
