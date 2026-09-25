package tui

import (
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"Echo/internel/provider"

	tea "github.com/charmbracelet/bubbletea"
)

type fakeClient struct {
	streams  []provider.Stream
	err      error
	requests [][]provider.Message
	contexts []context.Context
}

func (c *fakeClient) StreamChat(ctx context.Context, messages []provider.Message) (provider.Stream, error) {
	request := append([]provider.Message(nil), messages...)
	c.requests = append(c.requests, request)
	c.contexts = append(c.contexts, ctx)
	if c.err != nil {
		return nil, c.err
	}
	stream := c.streams[0]
	c.streams = c.streams[1:]
	return stream, nil
}

type streamResult struct {
	chunk string
	err   error
}

type fakeStream struct {
	results    []streamResult
	closeCount int
}

func (s *fakeStream) Recv() (string, error) {
	result := s.results[0]
	s.results = s.results[1:]
	return result.chunk, result.err
}

func (s *fakeStream) Close() error {
	s.closeCount++
	return nil
}

func updateModel(t *testing.T, m model, msg tea.Msg) (model, tea.Cmd) {
	t.Helper()
	updated, cmd := m.Update(msg)
	return updated.(model), cmd
}

func submit(t *testing.T, m model, text string) (model, tea.Cmd) {
	t.Helper()
	m.input.SetValue(text)
	return updateModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})
}

func TestStreamedConversation(t *testing.T) {
	first := &fakeStream{results: []streamResult{
		{chunk: "你"},
		{chunk: "好"},
		{err: io.EOF},
	}}
	second := &fakeStream{results: []streamResult{{chunk: "可以"}, {err: io.EOF}}}
	client := &fakeClient{streams: []provider.Stream{first, second}}
	m := newModel(client)

	m, cmd := submit(t, m, "你好")
	if len(client.requests) != 0 {
		t.Fatal("StreamChat was called inside Update")
	}
	if !m.loading {
		t.Fatal("model should be loading after submission")
	}

	m, cmd = updateModel(t, m, cmd())
	m, cmd = updateModel(t, m, cmd())
	m, cmd = updateModel(t, m, cmd())
	m, cmd = updateModel(t, m, cmd())

	if m.loading {
		t.Fatal("model should stop loading at EOF")
	}
	if got := m.messages[len(m.messages)-1]; got != "Echo> 你好" {
		t.Fatalf("unexpected response: %q", got)
	}
	wantHistory := []provider.Message{
		{Role: provider.RoleUser, Content: "你好"},
		{Role: provider.RoleAssistant, Content: "你好"},
	}
	if !reflect.DeepEqual(m.history, wantHistory) {
		t.Fatalf("unexpected history: %#v", m.history)
	}
	if first.closeCount != 1 {
		t.Fatalf("stream closed %d times", first.closeCount)
	}

	m, cmd = submit(t, m, "继续")
	m, cmd = updateModel(t, m, cmd())
	m, cmd = updateModel(t, m, cmd())
	m, _ = updateModel(t, m, cmd())

	wantRequest := append(append([]provider.Message(nil), wantHistory...), provider.Message{
		Role: provider.RoleUser, Content: "继续",
	})
	if !reflect.DeepEqual(client.requests[1], wantRequest) {
		t.Fatalf("unexpected second request: %#v", client.requests[1])
	}
	if second.closeCount != 1 {
		t.Fatalf("second stream closed %d times", second.closeCount)
	}
}

func TestIgnoresEnterWhileLoading(t *testing.T) {
	client := &fakeClient{streams: []provider.Stream{&fakeStream{results: []streamResult{{err: io.EOF}}}}}
	m := newModel(client)
	m, startCmd := submit(t, m, "first")
	m.input.SetValue("second")

	m, cmd := updateModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("loading submission should not return a command")
	}
	if len(client.requests) != 0 {
		t.Fatal("loading submission should not start another request")
	}

	_ = startCmd()
}

func TestStreamErrors(t *testing.T) {
	t.Run("start", func(t *testing.T) {
		client := &fakeClient{err: errors.New("start failed")}
		m := newModel(client)
		m, cmd := submit(t, m, "hello")
		m, _ = updateModel(t, m, cmd())

		if m.loading {
			t.Fatal("model should recover from start error")
		}
		if len(m.history) != 0 {
			t.Fatal("failed request should not enter history")
		}
		if !strings.Contains(m.messages[len(m.messages)-1], "start failed") {
			t.Fatalf("error not shown: %q", m.messages[len(m.messages)-1])
		}
	})

	t.Run("receive", func(t *testing.T) {
		stream := &fakeStream{results: []streamResult{{chunk: "partial"}, {err: errors.New("stream failed")}}}
		client := &fakeClient{streams: []provider.Stream{stream}}
		m := newModel(client)
		m, cmd := submit(t, m, "hello")
		m, cmd = updateModel(t, m, cmd())
		m, cmd = updateModel(t, m, cmd())
		m, _ = updateModel(t, m, cmd())

		if m.loading {
			t.Fatal("model should recover from receive error")
		}
		if len(m.history) != 0 {
			t.Fatal("interrupted request should not enter history")
		}
		if !strings.Contains(m.messages[len(m.messages)-1], "生成中断：stream failed") {
			t.Fatalf("stream error not shown: %q", m.messages[len(m.messages)-1])
		}
		if stream.closeCount != 1 {
			t.Fatalf("stream closed %d times", stream.closeCount)
		}
	})
}

func TestExitCancelsRequest(t *testing.T) {
	stream := &fakeStream{results: []streamResult{{err: io.EOF}}}
	client := &fakeClient{streams: []provider.Stream{stream}}
	m := newModel(client)
	m, startCmd := submit(t, m, "hello")
	m, _ = updateModel(t, m, startCmd())

	m, cmd := updateModel(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("exit should return a command")
	}
	if err := client.contexts[0].Err(); !errors.Is(err, context.Canceled) {
		t.Fatalf("request context was not canceled: %v", err)
	}

	_ = closeStreamCmd(m.stream)()
	if stream.closeCount != 1 {
		t.Fatalf("stream closed %d times", stream.closeCount)
	}
}
