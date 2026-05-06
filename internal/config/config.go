package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const appDir = "treehouse"
const configFile = "config.json"

func migrateLegacyConfigDir() {
	root, err := configRoot()
	if err != nil {
		return
	}
	newDir := filepath.Join(root, appDir)
	oldDir := filepath.Join(root, "claude-manager")
	if _, err := os.Stat(newDir); err == nil {
		return
	}
	if _, err := os.Stat(oldDir); err != nil {
		return
	}
	_ = os.Rename(oldDir, newDir)
}

type RepoConfig struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	RepoPath         string `json:"repo_path"`
	WorktreeBasePath string `json:"worktree_base_path"`
}

type Config struct {
	Repos []RepoConfig `json:"repos"`
}

func configRoot() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return xdg, nil
	}
	root, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return root, nil
}

func ConfigPath() (string, error) {
	root, err := configRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, appDir, configFile), nil
}

func Load() (Config, error) {
	migrateLegacyConfigDir()
	path, err := ConfigPath()
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	slices.SortFunc(cfg.Repos, func(a, b RepoConfig) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	return cfg, nil
}

func Save(cfg Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	slices.SortFunc(cfg.Repos, func(a, b RepoConfig) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("chmod config: %w", err)
	}
	return nil
}

func NormalizeRepo(repo RepoConfig) (RepoConfig, error) {
	var err error
	repo.Name = strings.TrimSpace(repo.Name)
	repo.RepoPath, err = normalizeAbs(repo.RepoPath)
	if err != nil {
		return RepoConfig{}, fmt.Errorf("normalize repo path: %w", err)
	}
	repo.WorktreeBasePath, err = normalizeAbs(repo.WorktreeBasePath)
	if err != nil {
		return RepoConfig{}, fmt.Errorf("normalize worktree base path: %w", err)
	}
	if repo.Name == "" {
		repo.Name = filepath.Base(repo.RepoPath)
	}
	if repo.ID == "" {
		repo.ID = repo.Name + "::" + repo.RepoPath
	}
	return repo, nil
}

func normalizeAbs(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("path is required")
	}
	path = os.ExpandEnv(path)
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			path = home
		} else {
			path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}
