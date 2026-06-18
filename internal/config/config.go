package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const DefaultPath = "fuse.yml"

type Config struct {
	Providers map[string]ProviderConfig `yaml:"providers"`
	Budgets   BudgetConfig              `yaml:"budgets"`
	OnHardCap string                    `yaml:"on_hard_cap"`
}

type ProviderConfig struct {
	BaseURL string                `yaml:"base_url"`
	APIKey  string                `yaml:"api_key"`
	Models  map[string]ModelCosts `yaml:"models"`
}

type ModelCosts struct {
	InputCostPer1K  float64 `yaml:"input_cost_per_1k"`
	OutputCostPer1K float64 `yaml:"output_cost_per_1k"`
}

type BudgetConfig struct {
	Daily   Budget `yaml:"daily"`
	Weekly  Budget `yaml:"weekly"`
	Monthly Budget `yaml:"monthly"`
}

type Budget struct {
	Soft float64 `yaml:"soft"`
	Hard float64 `yaml:"hard"`
}

func Load(path string) (*Config, error) {
	if path == "" {
		path = DefaultPath
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func WriteDefault(path string) error {
	if path == "" {
		path = DefaultPath
	}
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s already exists", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		return err
	}
	return os.WriteFile(path, []byte(DefaultYAML), 0o644)
}

func (c *Config) Validate() error {
	if len(c.Providers) == 0 {
		return errors.New("at least one provider is required")
	}
	if c.OnHardCap == "" {
		c.OnHardCap = "block"
	}
	if c.OnHardCap != "block" && c.OnHardCap != "warn" {
		return errors.New("on_hard_cap must be block or warn")
	}
	for name, provider := range c.Providers {
		if provider.BaseURL == "" {
			return fmt.Errorf("provider %q missing base_url", name)
		}
		if len(provider.Models) == 0 {
			return fmt.Errorf("provider %q must define at least one model", name)
		}
		for model, costs := range provider.Models {
			if costs.InputCostPer1K < 0 || costs.OutputCostPer1K < 0 {
				return fmt.Errorf("provider %q model %q costs must be non-negative", name, model)
			}
		}
	}
	return nil
}

func (c *Config) APIKey(provider string) string {
	value := c.Providers[provider].APIKey
	if strings.HasPrefix(value, "${") && strings.HasSuffix(value, "}") {
		return os.Getenv(strings.TrimSuffix(strings.TrimPrefix(value, "${"), "}"))
	}
	return value
}

func (c *Config) ModelCost(provider, model string) (ModelCosts, bool) {
	p, ok := c.Providers[provider]
	if !ok {
		return ModelCosts{}, false
	}
	costs, ok := p.Models[model]
	return costs, ok
}

const DefaultYAML = `# fuse.yml
providers:
  anthropic:
    base_url: "https://api.anthropic.com"
    api_key: "${ANTHROPIC_API_KEY}"
    models:
      claude-sonnet-4-20250514:
        input_cost_per_1k: 0.003
        output_cost_per_1k: 0.015
      claude-opus-4-20250514:
        input_cost_per_1k: 0.015
        output_cost_per_1k: 0.075

  openai:
    base_url: "https://api.openai.com"
    api_key: "${OPENAI_API_KEY}"
    models:
      gpt-4.1:
        input_cost_per_1k: 0.002
        output_cost_per_1k: 0.008

  google:
    base_url: "https://generativelanguage.googleapis.com"
    api_key: "${GEMINI_API_KEY}"
    models:
      gemini-2.5-pro:
        input_cost_per_1k: 0.00125
        output_cost_per_1k: 0.010

budgets:
  daily:
    soft: 50.00
    hard: 100.00
  weekly:
    soft: 200.00
    hard: 500.00
  monthly:
    soft: 500.00
    hard: 1000.00

on_hard_cap: block
`
