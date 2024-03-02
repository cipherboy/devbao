package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Focusable interface {
	Focus() tea.Cmd
	Blur()
	Focused() bool
}

type Button struct {
	Text string
	Id   string

	Selected bool
	Disabled bool

	Action func() tea.Cmd
}

var _ Focusable = &Button{}

func NewButton(namespace string, text string, action func() tea.Cmd) *Button {
	return &Button{
		Text:   text,
		Id:     fmt.Sprintf("%v-%v", namespace, text),
		Action: action,
	}
}

func (m *Button) Init() tea.Cmd {
	return nil
}

func (m *Button) Focus() tea.Cmd {
	m.Selected = true
	return nil
}

func (m *Button) Blur() {
	m.Selected = false
}

func (m *Button) Focused() bool {
	return m.Selected
}

func (m *Button) Update(msg tea.Msg) (*Button, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", " ":
			if !m.Disabled && m.Selected && m.Action != nil {
				cmds = append(cmds, m.Action())
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *Button) View() string {
	style := lipgloss.NewStyle().Padding(0, 2).Border(lipgloss.RoundedBorder(), true, true, true, true)
	if !m.Disabled {
		style.BorderForeground(highlightColor)
	} else {
		style.BorderForeground(lowlightColor)
	}

	prefix := ""
	if m.Focused() {
		style.Underline(true)
		prefix = "> "
	}

	return style.Render(prefix + m.Text)
}
