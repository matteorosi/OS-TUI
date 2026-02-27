package ui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the global keybindings for the TUI.
// It centralizes all key definitions using the charmbracelet/bubbles/key package.
// The bindings can be used throughout the UI components.

type KeyMap struct {
	Quit        key.Binding // quit the application (ctrl+c or q)
	Help        key.Binding // toggle help view
	Refresh     key.Binding // refresh data
	Up          key.Binding // move selection up (k or up arrow)
	Down        key.Binding // move selection down (j or down arrow)
	Enter       key.Binding // select / confirm (enter)
	CloudSelect key.Binding // select cloud (c)
	Esc         key.Binding // go back / close modal (esc)
}

// GlobalKeyMap holds the keybindings used across the application.
var GlobalKeyMap = KeyMap{
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c", "q"),
		key.WithHelp("ctrl+c/q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("k/up", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("j/down", "move down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	CloudSelect: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "select cloud"),
	),
	Esc: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "go back"),
	),
}

// ShortHelp returns a slice of key bindings for the help view.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit, k.Help, k.Refresh, k.CloudSelect}
}

// FullHelp returns a matrix of key bindings for the help view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Up, k.Down, k.Enter, k.CloudSelect, k.Esc}, {k.Quit, k.Help, k.Refresh}}
}
