package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const DefaultProvider = "openai-compatible"

type Config struct {
	Provider   string `json:"provider"`
	APIKey     string `json:"api_key"`
	BaseURL    string `json:"base_url"`
	Model      string `json:"model"`
	APIVersion string `json:"api_version,omitempty"`
}

func Path() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".echo", "config.json")
}

func Load() (*Config, error) {
	b, err := os.ReadFile(Path())
	if errors.Is(err, os.ErrNotExist) || len(bytes.TrimSpace(b)) == 0 {
		return &Config{Provider: DefaultProvider}, nil
	}
	if err != nil {
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	if c.Provider == "" {
		c.Provider = DefaultProvider
	}
	return &c, nil
}

func (c *Config) Validate() error {
	if c.Provider == "" {
		c.Provider = DefaultProvider
	}
	if c.BaseURL == "" {
		return errors.New("base URL is required")
	}
	if c.Model == "" {
		return errors.New("model is required")
	}
	if c.Provider == "azure" {
		if c.APIKey == "" {
			return errors.New("API key is required for Azure OpenAI")
		}
		if c.APIVersion == "" {
			return errors.New("API version is required for Azure OpenAI")
		}
	}
	return nil
}

func (c *Config) Save() error {
	if err := c.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(Path()), 0700); err != nil {
		return err
	}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(Path(), b, 0600)
}
