package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type listDelegate struct{}

func (d listDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	snippet, ok := item.(Snippet)
	if !ok {
		return
	}

	fn := lipgloss.NewStyle().PaddingLeft(4).Render

	if index == m.Index() {
		fn = func(strs ...string) string {
			return lipgloss.
				NewStyle().
				PaddingLeft(2).
				Foreground(lipgloss.Color("170")).
				Render("> " + strings.Join(strs, " "))
		}
	}

	fmt.Fprint(w, fn(snippet.ShowItem(index+1)))
}

func (d listDelegate) Height() int {
	return 2
}

func (d listDelegate) Spacing() int {
	return 1
}

type updateViewportContentMsg struct {
	index int
}

func (d listDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.KeyMap.CursorDown), key.Matches(msg, m.KeyMap.CursorUp):
			return func() tea.Msg {
				return updateViewportContentMsg{m.Index()}
			}
		}
	}

	return nil
}
