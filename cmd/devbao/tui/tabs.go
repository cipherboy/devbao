package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tabModel interface {
	Name() string
	InTextbox() bool
	Closeable() bool

	Init() tea.Cmd
	UpdateTab(tea.Msg) (tabModel, tabModel, tea.Cmd)
	View() string
}

type tabs struct {
	Tabs   []tabModel
	Cursor int

	Width  int
	Height int
}

var _ tea.Model = &tabs{}

func TabModel() *tabs {
	return &tabs{
		Tabs: []tabModel{
			NodesModel(),
			ProfilesModel(),
		},
		Cursor: 0,
	}
}

func (m *tabs) InnerWidth() int {
	return m.Width - 11
}

func (m *tabs) InnerHeight() int {
	return m.Height - 13
}

func (m *tabs) Init() tea.Cmd {
	var cmds []tea.Cmd

	for _, tab := range m.Tabs {
		cmd := tab.Init()
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return tea.Batch(cmds...)
}

type TabCloseMsg string

var _ tea.Msg = TabCloseMsg("")

var TabCloseCmd = func() tea.Msg {
	return TabCloseMsg("close")
}

type TabFocusMsg string

var _ tea.Msg = TabFocusMsg("")

type TabReplaceMsg struct {
	OldTab tabModel
	NewTab tabModel
}

var _ tea.Msg = &TabReplaceMsg{}

func TabReplaceCmd(oldTab tabModel, newTab tabModel) func() tea.Msg {
	return func() tea.Msg {
		return &TabReplaceMsg{
			OldTab: oldTab,
			NewTab: newTab,
		}
	}
}

func (m *tabs) Update(src tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	pastTab := m.Cursor
	changedActive := false
	doTabClose := false
	switch msg := src.(type) {
	case tea.WindowSizeMsg:
		m.Height = msg.Height
		m.Width = msg.Width

		// Update the message to contain the padded dimensions, not the
		// exterior dimensions; this allows tabs to ignore that they're
		// rendered inside a box.
		msg.Height = m.InnerHeight()
		msg.Width = m.InnerWidth()
		src = msg

		// Tell every tab the new screen size.
		for index := range m.Tabs {
			var tabCmd tea.Cmd = nil
			m.Tabs[index], _, tabCmd = m.Tabs[index].UpdateTab(tea.WindowSizeMsg{
				Width:  m.InnerWidth(),
				Height: m.InnerHeight(),
			})
			if tabCmd != nil {
				cmds = append(cmds, tabCmd)
			}
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "ctrl+q":
			// Tell all the tabs we're shutting down.
			for _, tab := range m.Tabs {
				_, _, _ = tab.UpdateTab(tea.QuitMsg{})
			}

			return m, tea.Quit
		case "ctrl+pgup", "tab":
			changedActive = true
			m.Cursor = (m.Cursor + 1) % len(m.Tabs)
		case "h":
			if !m.Tabs[m.Cursor].InTextbox() {
				changedActive = true
				m.Cursor = (m.Cursor + 1) % len(m.Tabs)
			}
		case "ctrl+pgdown", "shift+tab":
			changedActive = true
			m.Cursor = (m.Cursor - 1 + len(m.Tabs)) % len(m.Tabs)
		case "l":
			if !m.Tabs[m.Cursor].InTextbox() {
				changedActive = true
				m.Cursor = (m.Cursor - 1 + len(m.Tabs)) % len(m.Tabs)
			}
		case "ctrl+w":
			doTabClose = true
		}
	case TabCloseMsg:
		if string(msg) == "close" {
			doTabClose = true
		}
	case *TabReplaceMsg:
		pastTab = m.Cursor
		tabIndex := -1
		if msg.OldTab != nil {
			for index, tab := range m.Tabs {
				if tab == msg.OldTab {
					tabIndex = index
					break
				}
			}
		}

		changedActive = true
		if tabIndex != -1 {
			_, _, tabCmd := m.Tabs[tabIndex].UpdateTab(TabFocusMsg("blurred"))
			cmds = append(cmds, tabCmd)

			_, _, tabCmd = m.Tabs[tabIndex].UpdateTab(tea.QuitMsg{})
			cmds = append(cmds, tabCmd)

			m.Tabs[tabIndex] = msg.NewTab
		} else {
			m.Tabs = append(m.Tabs, msg.NewTab)
			tabIndex = len(m.Tabs) - 1
		}

		cmds = append(cmds, m.Tabs[tabIndex].Init())

		m.Cursor = tabIndex
	}

	if doTabClose && m.Tabs[m.Cursor].Closeable() {
		_, _, tabCmd := m.Tabs[m.Cursor].UpdateTab(tea.QuitMsg{})
		if tabCmd != nil {
			cmds = append(cmds, tabCmd)
		}

		changedActive = true
		previousTabs := m.Tabs[0:m.Cursor]
		nextTabs := m.Tabs[m.Cursor+1:]
		m.Tabs = nil
		m.Tabs = append(m.Tabs, previousTabs...)
		m.Tabs = append(m.Tabs, nextTabs...)
		m.Cursor = (m.Cursor - 1 + len(m.Tabs)) % len(m.Tabs)
	}

	// Tell the new active tab that the screen has changed size so that
	// it knows it is active.
	if changedActive {
		var tabCmd tea.Cmd
		if !doTabClose {
			m.Tabs[pastTab], _, tabCmd = m.Tabs[pastTab].UpdateTab(TabFocusMsg("blurred"))
			if tabCmd != nil {
				cmds = append(cmds, tabCmd)
			}
		}

		m.Tabs[m.Cursor], _, tabCmd = m.Tabs[m.Cursor].UpdateTab(tea.WindowSizeMsg{
			Width:  m.InnerWidth(),
			Height: m.InnerHeight(),
		})

		if tabCmd != nil {
			cmds = append(cmds, tabCmd)
		}

		m.Tabs[m.Cursor], _, tabCmd = m.Tabs[m.Cursor].UpdateTab(TabFocusMsg("focused"))
		if tabCmd != nil {
			cmds = append(cmds, tabCmd)
		}
	}

	var tabCmd tea.Cmd = nil
	var newTab tabModel = nil
	m.Tabs[m.Cursor], newTab, tabCmd = m.Tabs[m.Cursor].UpdateTab(src)

	if newTab != nil {
		// Node has spawned a new tab; switch to it.
		previousTabs := m.Tabs[0 : m.Cursor+1]
		nextTabs := m.Tabs[m.Cursor+1:]
		m.Tabs = nil
		m.Tabs = append(m.Tabs, previousTabs...)
		m.Tabs = append(m.Tabs, newTab)
		m.Tabs = append(m.Tabs, nextTabs...)
		m.Cursor += 1
	}

	if tabCmd != nil {
		cmds = append(cmds, tabCmd)
	}

	return m, tea.Batch(cmds...)
}

func tabBorderWithBottom(left, middle, right string) lipgloss.Border {
	border := lipgloss.RoundedBorder()
	border.BottomLeft = left
	border.Bottom = middle
	border.BottomRight = right
	return border
}

func (m *tabs) View() string {
	doc := strings.Builder{}

	var renderedTabs []string

	for i, t := range m.Tabs {
		var style lipgloss.Style
		isFirst, isLast, isActive := i == 0, i == len(m.Tabs)-1, i == m.Cursor
		if isActive {
			style = activeTabStyle.Copy()
		} else {
			style = inactiveTabStyle.Copy()
		}

		border, _, _, _, _ := style.GetBorder()
		if isFirst && isActive {
			border.BottomLeft = "│"
		} else if isFirst && !isActive {
			border.BottomLeft = "├"
		} else if isLast && isActive {
			border.BottomRight = "└"
		} else if isLast && !isActive {
			border.BottomRight = "┴"
		}

		style = style.Border(border)
		renderedTabs = append(renderedTabs, style.Render(t.Name()))
	}

	remainder := m.Width - lipgloss.Width(lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)) - 1 /* to include the rounded pipe character */ - 3 /* for padding on either side of the terminal */
	if remainder <= 0 {
		remainder = 1
	}
	remainderSpace := "┐"
	remainderSpace = "\n\n" + strings.Repeat("─", remainder-1) + remainderSpace
	remainderSpace = lipgloss.NewStyle().Foreground(highlightColor).Render(remainderSpace)
	renderedTabs = append(renderedTabs, remainderSpace)

	activeTab := m.Tabs[m.Cursor]

	row := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)

	if lipgloss.Width(row)+2 > m.Width {
		return fmt.Sprintf("resize terminal to be at least %v characters wide", m.Width)
	}

	doc.WriteString(row)
	doc.WriteString("\n")
	tabContent := lipgloss.NewStyle().Padding(1, 2, 1, 2).Render(activeTab.View())
	doc.WriteString(windowStyle.Width((lipgloss.Width(row) - windowStyle.GetHorizontalFrameSize())).Render(tabContent))
	return docStyle.Render(doc.String())
}
