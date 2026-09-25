package tui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"Echo/internel/provider"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	client          provider.Client
	input           textinput.Model
	messages        []string
	history         []provider.Message
	loading         bool
	pendingUser     string
	pendingResponse string
	cancel          context.CancelFunc
	stream          *managedStream
	width           int
	height          int
}

type managedStream struct {
	provider.Stream
	closeOnce sync.Once
	closeErr  error
}

func (s *managedStream) Close() error {
	s.closeOnce.Do(func() {
		s.closeErr = s.Stream.Close()
	})
	return s.closeErr
}

type streamStartedMsg struct {
	stream *managedStream
}

type streamChunkMsg string

type streamDoneMsg struct{}

type streamErrorMsg struct {
	err error
}

func Run(client provider.Client) error {
	_, err := tea.NewProgram(newModel(client), tea.WithAltScreen()).Run()
	return err
}

func newModel(client provider.Client) model {
	input := textinput.New()
	input.Placeholder = "输入你的问题…"
	input.Focus()
	input.CharLimit = 0
	input.Width = 80

	return model{client: client, input: input}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			if m.cancel != nil {
				m.cancel()
			}
			if m.stream != nil {
				return m, tea.Sequence(closeStreamCmd(m.stream), tea.Quit)
			}
			return m, tea.Quit
		case "enter":
			if m.loading {
				return m, nil
			}
			text := strings.TrimSpace(m.input.Value())
			if text == "" {
				return m, nil
			}

			request := append([]provider.Message(nil), m.history...)
			request = append(request, provider.Message{Role: provider.RoleUser, Content: text})
			ctx, cancel := context.WithCancel(context.Background())

			m.messages = append(m.messages, "你  > "+text, "Echo> 正在思考…")
			m.pendingUser = text
			m.pendingResponse = ""
			m.loading = true
			m.cancel = cancel
			m.input.Reset()
			m.input.Blur()
			return m, startStreamCmd(ctx, m.client, request)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.Width = max(1, msg.Width-8)

	case streamStartedMsg:
		m.stream = msg.stream
		return m, recvStreamCmd(msg.stream)

	case streamChunkMsg:
		m.pendingResponse += string(msg)
		m.messages[len(m.messages)-1] = "Echo> " + m.pendingResponse
		return m, recvStreamCmd(m.stream)

	case streamDoneMsg:
		if m.pendingResponse == "" {
			m.messages[len(m.messages)-1] = "Echo> （模型未返回内容）"
		}
		m.history = append(m.history,
			provider.Message{Role: provider.RoleUser, Content: m.pendingUser},
			provider.Message{Role: provider.RoleAssistant, Content: m.pendingResponse},
		)
		return m.finishRequest()

	case streamErrorMsg:
		if m.pendingResponse == "" {
			m.messages[len(m.messages)-1] = "Echo> 请求失败：" + msg.err.Error()
		} else {
			m.messages[len(m.messages)-1] += "\n\n生成中断：" + msg.err.Error()
		}
		return m.finishRequest()
	}

	if m.loading {
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) finishRequest() (tea.Model, tea.Cmd) {
	if m.cancel != nil {
		m.cancel()
	}
	m.loading = false
	m.pendingUser = ""
	m.pendingResponse = ""
	m.cancel = nil
	m.stream = nil
	return m, m.input.Focus()
}

func startStreamCmd(ctx context.Context, client provider.Client, messages []provider.Message) tea.Cmd {
	return func() tea.Msg {
		stream, err := client.StreamChat(ctx, messages)
		if err != nil {
			return streamErrorMsg{err: err}
		}
		managed := &managedStream{Stream: stream}
		if err := ctx.Err(); err != nil {
			_ = managed.Close()
			return streamErrorMsg{err: err}
		}
		return streamStartedMsg{stream: managed}
	}
}

func recvStreamCmd(stream *managedStream) tea.Cmd {
	return func() tea.Msg {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			if err := stream.Close(); err != nil {
				return streamErrorMsg{err: err}
			}
			return streamDoneMsg{}
		}
		if err != nil {
			_ = stream.Close()
			return streamErrorMsg{err: err}
		}
		return streamChunkMsg(chunk)
	}
}

func closeStreamCmd(stream *managedStream) tea.Cmd {
	return func() tea.Msg {
		_ = stream.Close()
		return nil
	}
}

func (m model) View() string {
	header := " Echo — AI Coding Assistant\n"
	body := strings.Join(m.messages, "\n\n")
	footer := "Enter 发送 · Esc 退出"
	if m.loading {
		footer = "正在生成 · Esc 退出"
	}

	return fmt.Sprintf(
		"%s\n%s\n\n%s\n\n%s\n",
		header,
		body,
		m.input.View(),
		footer,
	)
}
