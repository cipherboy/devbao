package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Checkbox struct {
	Title string
	Desc  string
	Id    string

	Value bool

	Selected bool
	Disabled bool

	Action func() tea.Cmd
}

var _ Focusable = &Checkbox{}

func NewCheckbox(namespace string, title string, desc string) *Checkbox {
	return &Checkbox{
		Title: title,
		Desc:  desc,
		Id:    fmt.Sprintf("%v-%v", namespace, desc),
	}
}

func (m *Checkbox) Init() tea.Cmd {
	return nil
}

func (m *Checkbox) Focus() tea.Cmd {
	m.Selected = true
	return nil
}

func (m *Checkbox) Blur() {
	m.Selected = false
}

func (m *Checkbox) Focused() bool {
	return m.Selected
}

func (m *Checkbox) Update(msg tea.Msg) (*Checkbox, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", " ", "left", "right":
			if !m.Disabled && m.Selected {
				m.Value = !m.Value
				if m.Action != nil {
					cmds = append(cmds, m.Action())
				}
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *Checkbox) View() string {
	style := lipgloss.NewStyle().Padding(0, 2).Border(lipgloss.RoundedBorder(), true, true, true, true)
	if !m.Disabled {
		style.BorderForeground(highlightColor)
	} else {
		style.BorderForeground(lowlightColor)
	}

	underlined := lipgloss.NewStyle()
	if m.Focused() {
		underlined = underlined.Underline(true)
	}

	fill := " "
	if m.Value {
		fill = "x"
	}

	field := fmt.Sprintf("[%v] %v", fill, m.Title)
	if m.Focused() {
		field = underlined.Render(field)
	}
	field = style.Render(field)

	if m.Focused() {
		field += "\n" + m.Desc
	}

	return field
}
