package main

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"golang.design/x/clipboard"
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
	out, err := glamour.Render(
		fmt.Sprintf("```%s\n%s\n```", s.Language, s.Content),
		"dracula",
	)
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
	New     key.Binding
	Select  key.Binding
	Delete  key.Binding
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
	New: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new"),
	),
	Select: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "select"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d", "backspace"),
		key.WithHelp("d/del", "delete"),
	),
}

type model struct {
	dimension dimension
	viewport  viewport.Model
	list      list.Model
	snippets  []Snippet
	ready     bool
	focus     focusTarget
	selected  map[int]struct{}
	textinput textinput.Model
}

func (m *model) AppendSnippet(s Snippet) {
	m.snippets = append(m.snippets, s)
}

type (
	readSnippetsMsg struct{ snippets []Snippet }
	clipboardMsg    error
)

func checkClipboard() tea.Msg {
	err := clipboard.Init()
	if err != nil {
		return clipboardMsg(err)
	}
	return nil
}

func (m model) Init() tea.Cmd {
	return checkClipboard
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case clipboardMsg:
		return m, tea.Quit

	case updateViewportContentMsg:
		m.viewport.SetContent(m.snippets[msg.index].RenderSnippet())

	case readSnippetsMsg:
		m.snippets = msg.snippets

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.New):
			if m.focus == focusViewport {
				break
			}
			b := clipboard.Read(clipboard.FmtText)
			if len(b) == 0 {
				return m, nil
			}

			newIdx := len(m.snippets)
			newSnippet := Snippet{
				Id:        "sklfjdaslk",
				Content:   string(b),
				Title:     fmt.Sprintf("My new snippet %d", newIdx),
				CreatedAt: time.Now(),
				Language:  "go",
			}
			m.AppendSnippet(newSnippet)
			return m, m.list.InsertItem(newIdx, newSnippet)

		case key.Matches(msg, keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, keys.Focus):
			m.focus = focusViewport

		case key.Matches(msg, keys.UnFocus):
			m.focus = focusList
		}

	case tea.WindowSizeMsg:
		m.dimension.UpdateDimension(msg.Height, msg.Width)

		// Configure list size
		m.list.SetSize(0, m.dimension.height-3)

		// Configure text input (full width)
		m.textinput.Width = m.dimension.width

		// Configure viewport
		vpWidth := m.dimension.width - lipgloss.Width(m.list.View()) - 4
		vpHeight := m.dimension.height - 3

		if !m.ready {
			m.viewport = viewport.New(vpWidth, vpHeight)
			m.viewport.SetContent(m.snippets[m.list.Index()].RenderSnippet())
			m.ready = true
		} else {
			m.viewport.Width = vpWidth
			m.viewport.Height = vpHeight
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
	var viewport string

	sidebar := lipgloss.NewStyle().
		MarginRight(2).
		Render(m.list.View())

	switch m.focus {
	case focusViewport:
		viewport = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("170")).
			Render(m.viewport.View())

	case focusList:
		viewport = lipgloss.NewStyle().
			Border(lipgloss.ASCIIBorder()).
			Render(m.viewport.View())
	}

	main := lipgloss.
		NewStyle().
		Render(lipgloss.JoinHorizontal(lipgloss.Top, sidebar, viewport))

	return lipgloss.JoinVertical(lipgloss.Top, main, m.textinput.View())
}

func initialModel() tea.Model {
	snippets := []Snippet{
		{
			Id:        "1",
			Title:     "Main example",
			Content:   "func main() {}",
			CreatedAt: time.Now(),
			Language:  "go",
		},
		{
			Id:        "2",
			Title:     "Foo example",
			Content:   "func foo() {}",
			CreatedAt: time.Now(),
			Language:  "go",
		},
		{
			Id:        "1",
			Title:     "Main exampleeeeeeeeeeeeeeeeeee",
			Content:   "func main() {}",
			CreatedAt: time.Now(),
			Language:  "go",
		},
	}
	ti := textinput.New()
	ti.Prompt = "Type something..."

	items := make([]list.Item, len(snippets))
	for i, s := range snippets {
		items[i] = s
	}
	list := list.New(items, listDelegate{}, 0, 0)
	list.SetShowTitle(false)
	list.SetShowStatusBar(false)
	list.SetShowHelp(false)
	list.DisableQuitKeybindings()
	list.SetFilteringEnabled(false)
	list.InfiniteScrolling = true

	return model{
		snippets:  snippets,
		list:      list,
		textinput: ti,
	}
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
