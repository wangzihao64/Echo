package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	input    textinput.Model
	messages []string
	width    int
	height   int
}

func Run() error {
	_, err := tea.NewProgram(newModel(), tea.WithAltScreen()).Run()
	return err
}

func newModel() model {
	input := textinput.New()
	input.Placeholder = "输入你的问题…"
	input.Focus()
	input.CharLimit = 0
	input.Width = 80

	return model{input: input}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			text := strings.TrimSpace(m.input.Value())
			if text != "" {
				m.messages = append(m.messages, "你  > "+text)
				m.messages = append(m.messages, "Echo> 正在思考…")
				m.input.Reset()
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.Width = msg.Width - 8
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) View() string {
	header := " Echo — AI Coding Assistant\n"
	body := strings.Join(m.messages, "\n\n")

	return fmt.Sprintf(
		"%s\n%s\n\n%s\n\nEnter 发送 · Esc 退出\n",
		header,
		body,
		m.input.View(),
	)
}
