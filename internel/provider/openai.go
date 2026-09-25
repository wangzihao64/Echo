package provider

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/ssestream"
)

type openAIClient struct {
	client openai.Client
	model  string
}

type openAIStream struct {
	stream *ssestream.Stream[openai.ChatCompletionChunk]
}

func newOpenAIClient(options Options, azure bool) (*openAIClient, error) {
	baseURL := options.BaseURL
	clientOptions := []option.RequestOption{
		option.WithAPIKey(options.APIKey),
	}

	if azure {
		var err error
		baseURL, err = url.JoinPath(baseURL, "openai", "deployments", options.Model)
		if err != nil {
			return nil, fmt.Errorf("build Azure OpenAI URL: %w", err)
		}
		clientOptions = append(clientOptions,
			option.WithHeader("api-key", options.APIKey),
			option.WithHeaderDel("Authorization"),
			option.WithQuery("api-version", options.APIVersion),
		)
	} else if options.APIKey == "" {
		clientOptions = append(clientOptions, option.WithHeaderDel("Authorization"))
	}

	clientOptions = append(clientOptions, option.WithBaseURL(strings.TrimRight(baseURL, "/")+"/"))
	if options.HTTPClient != nil {
		clientOptions = append(clientOptions, option.WithHTTPClient(options.HTTPClient))
	}

	client := openai.NewClient(clientOptions...)
	return &openAIClient{client: client, model: options.Model}, nil
}

func (c *openAIClient) StreamChat(ctx context.Context, messages []Message) (Stream, error) {
	params := openai.ChatCompletionNewParams{Model: c.model}
	params.Messages = make([]openai.ChatCompletionMessageParamUnion, 0, len(messages))

	for _, message := range messages {
		switch message.Role {
		case RoleSystem:
			params.Messages = append(params.Messages, openai.SystemMessage(message.Content))
		case RoleUser:
			params.Messages = append(params.Messages, openai.UserMessage(message.Content))
		case RoleAssistant:
			params.Messages = append(params.Messages, openai.AssistantMessage(message.Content))
		default:
			return nil, fmt.Errorf("unsupported message role: %s", message.Role)
		}
	}

	stream := c.client.Chat.Completions.NewStreaming(ctx, params)
	return &openAIStream{stream: stream}, nil
}

func (s *openAIStream) Recv() (string, error) {
	for s.stream.Next() {
		chunk := s.stream.Current()
		if len(chunk.Choices) == 0 || chunk.Choices[0].Delta.Content == "" {
			continue
		}
		return chunk.Choices[0].Delta.Content, nil
	}
	if err := s.stream.Err(); err != nil {
		return "", err
	}
	return "", io.EOF
}

func (s *openAIStream) Close() error {
	return s.stream.Close()
}
