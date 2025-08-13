package main

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"golang.design/x/clipboard"
)

type keyMap struct {
	Up               key.Binding
	Down             key.Binding
	Quit             key.Binding
	Help             key.Binding
	Select           key.Binding
	Yank             key.Binding
	Paste            key.Binding
	Delete           key.Binding
	FocusTextInput   key.Binding
	UnFocusTextInput key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit, k.Help}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Paste, k.Yank, k.Delete, k.Select},
		{k.Quit, k.Help},
	}
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	Select: key.NewBinding(
		key.WithKeys(tea.KeySpace.String()),
		key.WithHelp("space", "select snippet"),
	),
	Yank: key.NewBinding(
		key.WithKeys("y"),
		key.WithHelp("y", "yank snippet"),
	),
	Paste: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "paste snippet"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d", tea.KeyBackspace.String()),
		key.WithHelp("d/backspace", "delete snippet"),
	),
	FocusTextInput: key.NewBinding(
		key.WithKeys("i"),
		key.WithHelp("i", "use textinput"),
	),
	UnFocusTextInput: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "unfocus textinput"),
	),
}

type Snippets map[string][]string

type model struct {
	keys      keyMap
	help      help.Model
	textinput textinput.Model
	snippets  Snippets
	cursor    int
	selected  map[int]struct{}
	active    string
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
	var cmd tea.Cmd
	m.textinput, cmd = m.textinput.Update(msg)
	switch msg := msg.(type) {
	case clipboardErrMsg:
		fmt.Println("err with clipboard")
		return m, tea.Quit
	case readSnippetsMsg:
		m.snippets = msg.snippets
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Up):
			if m.cursor > 0 && !m.textinput.Focused() {
				m.cursor--
			}

		case key.Matches(msg, m.keys.Down):
			if m.cursor < len(m.snippets[m.active])-1 && !m.textinput.Focused() {
				m.cursor++
			}

		case key.Matches(msg, m.keys.Select):
			if _, ok := m.selected[m.cursor]; ok {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}

		case key.Matches(msg, m.keys.Yank):
			currSnippet := m.snippets[m.active][m.cursor]
			clipboard.Write(clipboard.FmtText, []byte(currSnippet))

		case key.Matches(msg, m.keys.Paste):
			b := clipboard.Read(clipboard.FmtText)
			m.snippets[m.active] = append(m.snippets[m.active], string(b))

		case key.Matches(msg, m.keys.Delete):
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

		case key.Matches(msg, m.keys.FocusTextInput):
			m.textinput.Focus()

		case key.Matches(msg, m.keys.UnFocusTextInput):
			m.textinput.Blur()

		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
		case key.Matches(msg, m.keys.Quit):
			err := m.save()
			if err != nil {
				fmt.Println(err)
			}
			return m, tea.Quit
		}
	}
	return m, cmd
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
				style = style.
					BorderForeground(lipgloss.Color("#3B82F6"))
			}
			snippet := fmt.Sprintf("```%s\n%s\n```", k, s)
			out, _ := glamour.Render(snippet, "dark")
			in += style.Render(out)
			in += "\n"
		}
	}
	helpView := m.help.View(m.keys)
	return fmt.Sprintf("%s\n%s\n\n%s", in, m.textinput.View(), helpView)
}

func initialModel() tea.Model {
	ti := textinput.New()
	ti.Placeholder = "Type something..."
	ti.CharLimit = 156
	ti.Width = 20
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#3B82F6"))

	return model{
		keys:      keys,
		help:      help.New(),
		textinput: ti,
		snippets:  make(map[string][]string),
		selected:  make(map[int]struct{}),
		active:    "go",
	}
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
