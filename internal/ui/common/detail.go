package common

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type DetailModel struct {
	title  string
	fields map[string]string
}

// NewDetail creates a detail view with a title and key/value pairs.
func NewDetail(title string, fields map[string]string) DetailModel {
	return DetailModel{title: title, fields: fields}
}

// View renders the detail view.
func (m DetailModel) View() string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render(m.title) + "\n")
	for k, v := range m.fields {
		b.WriteString(fmt.Sprintf("%s: %s\n", k, v))
	}
	return b.String()
}
