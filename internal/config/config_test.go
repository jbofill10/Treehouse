package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeRepo(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo, err := NormalizeRepo(RepoConfig{
		Name:             "",
		RepoPath:         "~/src/repo",
		WorktreeBasePath: "$HOME/worktrees/repo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, "src", "repo"); repo.RepoPath != want {
		t.Fatalf("repo path = %q, want %q", repo.RepoPath, want)
	}
	if want := filepath.Join(home, "worktrees", "repo"); repo.WorktreeBasePath != want {
		t.Fatalf("worktree path = %q, want %q", repo.WorktreeBasePath, want)
	}
	if repo.Name != "repo" {
		t.Fatalf("name = %q, want repo", repo.Name)
	}
}

func TestMigrateLegacyConfigDir(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)

	oldDir := filepath.Join(root, "claude-manager")
	if err := os.MkdirAll(oldDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldDir, "config.json"), []byte(`{"repos":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	newDir := filepath.Join(root, "treehouse")
	if _, err := os.Stat(newDir); err != nil {
		t.Fatalf("new config dir not created: %v", err)
	}
	if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
		t.Fatalf("old config dir still exists after migration")
	}
	if _, err := os.Stat(filepath.Join(newDir, "config.json")); err != nil {
		t.Fatalf("config.json not present in new dir: %v", err)
	}
}

func TestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	cfg := Config{
		Repos: []RepoConfig{{ID: "1", Name: "b", RepoPath: "/tmp/b", WorktreeBasePath: "/tmp/b-wt"}},
	}
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	path, err := ConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 600", info.Mode().Perm())
	}
	loaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Repos) != 1 || loaded.Repos[0].ID != "1" {
		t.Fatalf("loaded = %#v", loaded)
	}
}
