package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

type profileApply struct {
	Profile string

	Width  int
	Height int
}

var _ tabModel = &profileApply{}

func ProfileApplyModel(name string) tabModel {
	return &profileApply{
		Profile: name,
	}
}

func (m profileApply) Name() string    { return "Profile: " + m.Profile }
func (m profileApply) InTextbox() bool { return false }
func (m profileApply) Closeable() bool { return true }

func (m *profileApply) Init() tea.Cmd {
	return nil
}

func (m *profileApply) UpdateTab(msg tea.Msg) (tabModel, tabModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
	case tea.KeyMsg:
	}

	return m, nil, nil
}

func (m *profileApply) View() string {
	return "Applying: " + m.Profile
}
