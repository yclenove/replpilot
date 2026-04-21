package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	defaultDirName  = ".replpilot"
	defaultFileName = "config.json"
)

type Host struct {
	ID       string `json:"id"`
	Address  string `json:"address"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	AuthType string `json:"auth_type"`
	KeyPath  string `json:"key_path,omitempty"`
}

type Source struct {
	ID         string `json:"id"`
	MasterHost string `json:"master_host"`
	MasterPort int    `json:"master_port"`
	ReplUser   string `json:"repl_user"`
}

type Config struct {
	Hosts   []Host   `json:"hosts"`
	Sources []Source `json:"sources"`
}

func (c *Config) FindHost(id string) (*Host, bool) {
	for _, item := range c.Hosts {
		if item.ID == id {
			host := item
			return &host, true
		}
	}
	return nil, false
}

func (c *Config) FindSource(id string) (*Source, bool) {
	for _, item := range c.Sources {
		if item.ID == id {
			source := item
			return &source, true
		}
	}
	return nil, false
}

func DefaultFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户目录失败: %w", err)
	}
	return filepath.Join(home, defaultDirName, defaultFileName), nil
}

func EnsureDefaultConfig() (string, error) {
	path, err := DefaultFilePath()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("创建配置目录失败: %w", err)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := Save(path, &Config{Hosts: []Host{}, Sources: []Source{}}); err != nil {
			return "", err
		}
	}
	return path, nil
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置失败: %w", err)
	}
	var cfg Config
	if len(data) == 0 {
		return &Config{Hosts: []Host{}, Sources: []Source{}}, nil
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}
	if cfg.Hosts == nil {
		cfg.Hosts = []Host{}
	}
	if cfg.Sources == nil {
		cfg.Sources = []Source{}
	}
	return &cfg, nil
}

func Save(path string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("写入配置失败: %w", err)
	}
	return nil
}
