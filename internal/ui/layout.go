package ui

import "github.com/charmbracelet/lipgloss"

// Layout combines a sidebar and a main view side by side.
// The sidebar occupies 20% of the total width, the main view 80%.
// It uses lipgloss to enforce fixed widths and joins the two parts horizontally.
func Layout(sidebar, main string) string {
	// Define a total width. Adjust as needed for your terminal.
	const totalWidth = 100
	sidebarWidth := totalWidth / 5 // 20%
	mainWidth := totalWidth - sidebarWidth

	sb := lipgloss.NewStyle().Width(sidebarWidth).Render(sidebar)
	mn := lipgloss.NewStyle().Width(mainWidth).Render(main)
	return lipgloss.JoinHorizontal(lipgloss.Top, sb, mn)
}
