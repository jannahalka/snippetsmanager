package main

import (
	"fmt"
	"maps"
	"os"
	"slices"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	textarea textarea.Model
	snippets map[string][]string
	cursor   int
	selected map[int]struct{}
	active   string
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var taCmd tea.Cmd

	m.textarea, taCmd = m.textarea.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			if m.textarea.Focused() {
				m.textarea.Blur()
			}
		case tea.KeyTab:
			if m.textarea.Focused() {
				m.textarea.InsertRune('\t')
			}
		case tea.KeyEnter:
			if !m.textarea.Focused() {
				val := m.textarea.Value()
				if _, ok := m.snippets["go"]; ok {
					m.snippets["go"] = append(m.snippets["go"], val)
					m.textarea.Reset()
				}
			}
		case tea.KeySpace:
			if !m.textarea.Focused() {
				if _, ok := m.selected[m.cursor]; ok {
					delete(m.selected, m.cursor)
				} else {
					m.selected[m.cursor] = struct{}{}
				}
			}
		case tea.KeyRunes:
			if !m.textarea.Focused() {
				switch msg.String() {
				case "k":
					if m.cursor > 0 {
						m.cursor--
					}
				case "j":
					if m.cursor < len(m.snippets[m.active])-1 {
						m.cursor++
					}
				case "i":
					m.textarea.Focus()
				case "d":
					if len(m.snippets[m.active]) > 0 {
						s := []string{}
						for idx, snippet := range m.snippets[m.active] {
							if slices.Contains(slices.Collect(maps.Keys(m.selected)), idx) {
								delete(m.selected, idx)
							} else {
								s = append(s, snippet)
							}
						}
						m.snippets[m.active] = s
					}
				}
			}
		case tea.KeyCtrlC:
			return m, tea.Quit
		}

	}
	return m, tea.Batch(taCmd)
}
func (m model) View() string {
	var in string

	for k, v := range m.snippets {
		for idx, s := range v {
			var style lipgloss.Style

			if m.cursor == idx {
				style = style.
					Border(lipgloss.NormalBorder())
			} else {
				style = style.
					Border(lipgloss.ASCIIBorder())
			}
			_, ok := m.selected[idx]
			if ok {
				style = style.BorderForeground(lipgloss.Color("#3B82F6"))
			}
			snippet := fmt.Sprintf("```%s\n%s\n```", k, s)
			out, _ := glamour.Render(snippet, "dark")
			in += style.Render(out)
			in += "\n"
		}
	}

	return fmt.Sprintf("%s\n\n%s", in, m.textarea.View())

}

func listSnippets() map[string][]string {
	// TODO: Get from json file
	snippets := make(map[string][]string)
	return snippets
}

func initialModel() tea.Model {
	ta := textarea.New()

	snippets := listSnippets()
	snippets["go"] = []string{`func main1() {
	return ""
}`, `func main2() {
	return ""
}`, `func main3() {
	return ""
}`}

	return model{
		textarea: ta, snippets: snippets,
		selected: make(map[int]struct{}),
		active:   "go",
	}
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
