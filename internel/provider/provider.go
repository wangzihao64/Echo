package provider

import (
	"context"
	"errors"
	"net/http"
)

const (
	OpenAICompatible = "openai-compatible"
	AzureOpenAI      = "azure"
)

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type Message struct {
	Role    Role
	Content string
}

type Options struct {
	Provider   string
	APIKey     string
	BaseURL    string
	Model      string
	APIVersion string
	HTTPClient *http.Client
}

type Client interface {
	StreamChat(context.Context, []Message) (Stream, error)
}

type Stream interface {
	Recv() (string, error)
	Close() error
}

func New(options Options) (Client, error) {
	if options.Provider == "" {
		options.Provider = OpenAICompatible
	}
	if options.BaseURL == "" {
		return nil, errors.New("base URL is required")
	}
	if options.Model == "" {
		return nil, errors.New("model is required")
	}

	switch options.Provider {
	case OpenAICompatible:
		return newOpenAIClient(options, false)
	case AzureOpenAI:
		if options.APIKey == "" {
			return nil, errors.New("API key is required for Azure OpenAI")
		}
		if options.APIVersion == "" {
			return nil, errors.New("API version is required for Azure OpenAI")
		}
		return newOpenAIClient(options, true)
	default:
		return nil, errors.New("unsupported provider: " + options.Provider)
	}
}
