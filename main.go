package main

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

type Snippet struct {
	Id        string    `json:"id"`
	Content   string    `json:"content"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	Language  string    `json:"language"`
}

func (s Snippet) ShowItem(i int) string {
	return fmt.Sprintf("%d. %s", i, s.Title)
}

func (s Snippet) RenderSnippet() string {
	out, err := glamour.Render(s.Content, "dracula")
	if err != nil {
		fmt.Println("TODO: Handle err")
	}
	return out
}

func (s Snippet) FilterValue() string { return s.Title }

type focusTarget int

const (
	focusList focusTarget = iota
	focusViewport
)

type dimension struct {
	height int
	width  int
}

func (d *dimension) UpdateDimension(height, width int) {
	d.height = height
	d.width = width
}

type keyMap struct {
	Focus   key.Binding
	UnFocus key.Binding
	Quit    key.Binding
}

var keys = keyMap{
	Focus: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "focus"),
	),
	UnFocus: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "unfocus"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q/ctrl+c", "quit"),
	),
}

type model struct {
	dimension dimension
	viewport  viewport.Model
	list      list.Model
	snippets  []Snippet
	ready     bool
	focus     focusTarget
}

type readSnippetsMsg struct{ snippets []Snippet }

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case updateViewportContentMsg:
		m.viewport.SetContent(m.snippets[msg.index].RenderSnippet())

	case readSnippetsMsg:
		m.snippets = msg.snippets

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, keys.Focus):
			m.focus = focusViewport
		case key.Matches(msg, keys.UnFocus):
			m.focus = focusList
		}

	case tea.WindowSizeMsg:
		m.dimension.UpdateDimension(msg.Height, msg.Width)
		width, height := m.dimension.width, m.dimension.height
		m.list.SetSize(width, height)
		fmt.Println(lipgloss.Width(m.list.View()))

		if !m.ready {
			m.viewport = viewport.New(width, height)
			m.viewport.SetContent(m.snippets[m.list.Index()].RenderSnippet())
			m.ready = true
		} else {
			m.viewport.Width = width
			m.viewport.Height = height
		}
	}

	switch m.focus {
	case focusList:
		m.list, cmd = m.list.Update(msg)

	case focusViewport:
		m.viewport, cmd = m.viewport.Update(msg)
	}

	return m, cmd
}

func (m model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}
	var main string

	const spacing = 4

	sidebar := lipgloss.NewStyle().
		MarginRight(spacing).
		Render(m.list.View())

	available := m.dimension.width - lipgloss.Width(sidebar)
	m.viewport.Width = available - 2 // Left + Right border = 2
	m.viewport.Height = m.dimension.height - 2

	switch m.focus {
	case focusViewport:
		main = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			Render(m.viewport.View())

	case focusList:
		main = lipgloss.NewStyle().
			Border(lipgloss.ASCIIBorder()).
			Render(m.viewport.View())
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main)
}

func initialModel() tea.Model {
	snippets := []Snippet{
		{
			Id:        "1",
			Title:     "Main example",
			Content:   "```go\nfunc main() {}\n```",
			CreatedAt: time.Now(),
			Language:  "go",
		},
		{
			Id:        "2",
			Title:     "Foo example",
			Content:   "```go\nfunc foo() {}\n```",
			CreatedAt: time.Now(),
			Language:  "go",
		},
		{
			Id:        "1",
			Title:     "Main exampleeeeeeeeeeeeeeeeeee",
			Content:   "```go\nfunc main() {}\n```",
			CreatedAt: time.Now(),
			Language:  "go",
		},
	}

	items := make([]list.Item, len(snippets))
	for i, s := range snippets {
		items[i] = s
	}
	list := list.New(items, listDelegate{}, 0, 0)
	list.SetShowTitle(false)
	list.SetShowStatusBar(false)
	list.SetShowHelp(false)
	list.KeyMap.Quit.SetEnabled(false)

	return model{snippets: snippets, list: list}
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
