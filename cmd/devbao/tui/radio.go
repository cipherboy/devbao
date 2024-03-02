package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Radio struct {
	Title string
	Desc  string
	Id    string

	Options []string
	Value   int

	Selected bool
	Disabled bool

	Action func() tea.Cmd
}

var _ Focusable = &Radio{}

func NewRadio(namespace string, title string, desc string, opts []string) *Radio {
	return &Radio{
		Title:   title,
		Desc:    desc,
		Id:      fmt.Sprintf("%v-%v", namespace, title),
		Options: opts,
		Value:   0,
	}
}

func (m *Radio) Init() tea.Cmd {
	return nil
}

func (m *Radio) Focus() tea.Cmd {
	m.Selected = true
	return nil
}

func (m *Radio) Blur() {
	m.Selected = false
}

func (m *Radio) Focused() bool {
	return m.Selected
}

func (m *Radio) GetValue() string {
	return m.Options[m.Value]
}

func (m *Radio) Update(msg tea.Msg) (*Radio, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "n", "right":
			if !m.Disabled && m.Selected {
				m.Value = (m.Value + 1) % len(m.Options)
				if m.Action != nil {
					cmds = append(cmds, m.Action())
				}
			}
		case "p", "left":
			if !m.Disabled && m.Selected {
				m.Value = (m.Value - 1 + len(m.Options)) % len(m.Options)
				if m.Action != nil {
					cmds = append(cmds, m.Action())
				}
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *Radio) View() string {
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

	var fields []string
	for index, option := range m.Options {
		fill := " "
		if m.Value == index {
			fill = "x"
		}

		field := fmt.Sprintf("(%v) %v", fill, option)
		if m.Value == index {
			field = underlined.Render(field)
		}

		fields = append(fields, " "+field+" ")
	}

	rendered := m.Title + "\n"
	rendered += style.Render(strings.Join(fields, " "))

	if m.Selected {
		rendered += "\n" + m.Desc
	}

	return rendered
}
