package main

import (
	"encoding/json"
	"fmt"
	"golang.design/x/clipboard"
	"io"
	"maps"
	"os"
	"slices"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

type Snippets map[string][]string

type model struct {
	textarea textarea.Model
	snippets Snippets
	cursor   int
	selected map[int]struct{}
	active   string
}

type readSnippetsMsg struct{ snippets Snippets }

func readData() tea.Msg {
	var snippets Snippets
	jsonFile, err := os.Open("data.json")
	defer jsonFile.Close()
	if err != nil {
		fmt.Println(err)
	}
	byteValue, _ := io.ReadAll(jsonFile)
	json.Unmarshal(byteValue, &snippets)
	return readSnippetsMsg{snippets}
}

type clipboardErrMsg struct{ err error }

func checkClipboard() tea.Msg {
	err := clipboard.Init()
	if err != nil {
		return clipboardErrMsg{err}
	}
	return nil
}

func (m model) save() error {
	b, err := json.Marshal(m.snippets)
	if err != nil {
		return err
	}
	err = os.WriteFile("data.json", b, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (m model) Init() tea.Cmd {
	return tea.Batch(readData, textarea.Blink, checkClipboard)
}
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var taCmd tea.Cmd

	m.textarea, taCmd = m.textarea.Update(msg)

	switch msg := msg.(type) {
	case clipboardErrMsg:
		fmt.Println("err with clipboard")
		return m, tea.Quit
	case readSnippetsMsg:
		m.snippets = msg.snippets
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
				if len(val) > 0 {
					if _, ok := m.snippets["go"]; ok {
						m.snippets["go"] = append(m.snippets["go"], val)
						m.textarea.Reset()
					}
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
				case "q":
					err := m.save()
					if err != nil {
						fmt.Println(err)
					}
					return m, tea.Quit
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
				case "y":
					currSnippet := m.snippets[m.active][m.cursor]
					clipboard.Write(clipboard.FmtText, []byte(currSnippet))
				case "c":
					b := clipboard.Read(clipboard.FmtText)
					m.snippets[m.active] = append(m.snippets[m.active], string(b))
				case "d":
					if len(m.snippets[m.active]) > 0 {
						newSnippets := []string{}
						for idx, snippet := range m.snippets[m.active] {
							if slices.Contains(slices.Collect(maps.Keys(m.selected)), idx) {
								delete(m.selected, idx)
							} else {
								newSnippets = append(newSnippets, snippet)
							}
						}
						m.snippets[m.active] = newSnippets
					}
				}
			}
		case tea.KeyCtrlC:
			err := m.save()
			if err != nil {
				fmt.Println(err)
			}
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
					Border(lipgloss.ASCIIBorder()).
					BorderForeground(lipgloss.Color("#898989"))
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

func initialModel() tea.Model {
	return model{
		textarea: textarea.New(),
		snippets: make(map[string][]string),
		selected: make(map[int]struct{}),
		active:   "go",
	}
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
