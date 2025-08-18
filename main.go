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
	return fmt.Sprintf("[%d] %s", i, s.Title)
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

type messageSeverity int

const (
	SUCCESS messageSeverity = iota
	WARNING
	ERROR
	INFO
)

type focusTarget int

const (
	focusList focusTarget = iota
	focusViewport
	focusTextinput
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
	Type    key.Binding
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
	Type: key.NewBinding(
		key.WithKeys("i"),
		key.WithHelp("i", "type to input"),
	),
}

type message struct {
	severity messageSeverity
	content  string
}

type styles struct {
	viewportStyle  lipgloss.Style
	listStyle      lipgloss.Style
	textinputStyle lipgloss.Style
}

type model struct {
	dimension dimension
	viewport  viewport.Model
	ready     bool
	list      list.Model
	snippets  []Snippet
	focus     focusTarget
	textinput textinput.Model
	status    message
	styles    styles
}

func (m *model) AppendSnippet(s Snippet) {
	m.snippets = append(m.snippets, s)
}

type (
	readSnippetsMsg struct{ snippets []Snippet }
	clipboardMsg    error
	statusChangeMsg message
)

func checkClipboard() tea.Msg {
	err := clipboard.Init()
	if err != nil {
		return clipboardMsg(err)
	}
	return nil
}

func (m model) Init() tea.Cmd {
	return tea.Batch(checkClipboard)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.focus {
	case focusList:
		m.list, cmd = m.list.Update(msg)

	case focusViewport:
		m.viewport, cmd = m.viewport.Update(msg)

	case focusTextinput:
		m.textinput, cmd = m.textinput.Update(msg)
	}

	switch msg := msg.(type) {
	case statusChangeMsg:
		m.status = message(msg)
	case clipboardMsg:
		return m, tea.Quit

	case updateViewportContentMsg:
		m.viewport.SetContent(m.snippets[msg.index].RenderSnippet())

	case readSnippetsMsg:
		m.snippets = msg.snippets

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.New):
			if m.focus == focusViewport || m.focus == focusTextinput {
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
			return m, func() tea.Msg { return statusChangeMsg{severity: INFO, content: "FOCUS"} }

		case key.Matches(msg, keys.UnFocus):
			m.focus = focusList
			return m, func() tea.Msg { return statusChangeMsg{severity: INFO, content: "NORMAL"} }

		case key.Matches(msg, keys.Type):
			m.focus = focusTextinput
			m.textinput.Focus()
			return m, func() tea.Msg { return statusChangeMsg{severity: INFO, content: "INSERT"} }
		}

	case tea.WindowSizeMsg:
		m.dimension = dimension{width: msg.Width, height: msg.Height}
		listWidth := 40
		textinputHeight := 1
		vpX, vpY := m.styles.viewportStyle.GetFrameSize()
		listX, listY := m.styles.listStyle.GetFrameSize()
		tiX, tiY := m.styles.textinputStyle.GetFrameSize()

		m.textinput.Width = msg.Width/4 - tiX
		m.list.SetSize(listWidth, msg.Height-listY-tiY-textinputHeight)

		if !m.ready {
			m.viewport = viewport.New(msg.Width-vpX-listWidth-listX, msg.Height-vpY-tiY-textinputHeight)
			m.viewport.SetContent(m.snippets[m.list.Index()].RenderSnippet())
			m.ready = true
		} else {
			m.viewport.Width = msg.Width - vpX - listWidth - listX
			m.viewport.Height = msg.Height - vpY - tiY - textinputHeight
		}
	}

	return m, cmd
}

func (m model) View() string {
	switch m.focus {
	case focusViewport:
		m.styles.viewportStyle = GetFocusStyle(&m.styles.viewportStyle)
	case focusTextinput:
		m.styles.textinputStyle = GetFocusStyle(&m.styles.textinputStyle)
	case focusList:
		m.styles.listStyle = GetFocusStyle(&m.styles.listStyle)
	}

	main := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.styles.listStyle.
			Width(m.list.Width()).
			Render(m.list.View()),
		m.styles.viewportStyle.Render(m.viewport.View()),
	)

	ti := m.styles.textinputStyle.
		Width(m.textinput.Width).
		Render(m.textinput.View())

	status := lipgloss.NewStyle().
		Width(m.dimension.width - lipgloss.Width(ti) - 2).
		AlignHorizontal(lipgloss.Left).
		AlignVertical(lipgloss.Center).
		PaddingLeft(1).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#e0faee")).
		Foreground(lipgloss.Color("#e0faee")).
		Render(m.status.content)

	return lipgloss.JoinVertical(
		lipgloss.Top,
		main,
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			ti,
			status,
		),
	)
}

func GetFocusStyle(style *lipgloss.Style) lipgloss.Style {
	return style.
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("170"))
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
	ti.Placeholder = "Type here..."
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("170"))

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

	styles := styles{
		viewportStyle: lipgloss.NewStyle().
			Border(lipgloss.ASCIIBorder()),
		listStyle: lipgloss.NewStyle().
			Border(lipgloss.ASCIIBorder()).
			MarginRight(1).
			PaddingTop(1),
		textinputStyle: lipgloss.NewStyle().
			Border(lipgloss.ASCIIBorder()).
			MarginRight(2),
	}

	return model{
		snippets:  snippets,
		list:      list,
		textinput: ti,
		styles:    styles,
		status:    message{severity: INFO, content: "NORMAL"},
	}
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
