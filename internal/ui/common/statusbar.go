package common

import "github.com/charmbracelet/lipgloss"

type StatusBar struct {
	message string
}

// NewStatusBar creates a status bar with the given message.
func NewStatusBar(msg string) StatusBar {
	return StatusBar{message: msg}
}

// SetMessage updates the status bar text.
func (s *StatusBar) SetMessage(msg string) {
	s.message = msg
}

// View renders the status bar with a simple style.
func (s StatusBar) View() string {
	style := lipgloss.NewStyle().
		Background(lipgloss.Color("#333")).
		Foreground(lipgloss.Color("#fff")).
		Padding(0, 1)
	return style.Render(s.message)
}
