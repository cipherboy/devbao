package tui

import "github.com/charmbracelet/lipgloss"

var (
	inactiveTabBorder = tabBorderWithBottom("┴", "─", "┴")
	activeTabBorder   = tabBorderWithBottom("┘", " ", "└")
	docStyle          = lipgloss.NewStyle().Padding(1, 2, 1, 2)
	highlightColor    = lipgloss.Color("#4C83FC")
	lowlightColor     = lipgloss.AdaptiveColor{Light: "#454952", Dark: "#646C7D"}
	inactiveTabStyle  = lipgloss.NewStyle().Border(inactiveTabBorder, true).BorderForeground(highlightColor).Padding(0, 1)
	activeTabStyle    = inactiveTabStyle.Copy().Border(activeTabBorder, true)
	windowStyle       = lipgloss.NewStyle().BorderForeground(highlightColor).Padding(2, 0).Align(lipgloss.Left).Border(lipgloss.NormalBorder()).UnsetBorderTop()
	titleStyle        = lipgloss.NewStyle().Foreground(highlightColor)
	warningColor      = lipgloss.AdaptiveColor{Light: "#803526", Dark: "#CC2200"}
	warningStyle      = lipgloss.NewStyle().Foreground(warningColor)
)
