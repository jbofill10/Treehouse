package ui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePathInputUsesLaunchCwdForRelativePaths(t *testing.T) {
	cwd := t.TempDir()
	got, err := resolvePathInput("./repo", cwd)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(cwd, "repo")
	if got != want {
		t.Fatalf("resolved path = %q, want %q", got, want)
	}
}

func TestResolvePathInputExpandsTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	got, err := resolvePathInput("~/repo", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, "repo")
	if got != want {
		t.Fatalf("resolved path = %q, want %q", got, want)
	}
}

func TestListPathSuggestionsUsesNearestExistingParent(t *testing.T) {
	cwd := t.TempDir()
	base := filepath.Join(cwd, "projects")
	if err := os.MkdirAll(filepath.Join(base, "alpha"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(base, "beta"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := listPathSuggestions(filepath.Join(base, "al"), cwd)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Value != filepath.Join(base, "alpha") {
		t.Fatalf("suggestions = %#v, want alpha only", got)
	}
}

func TestListPathSuggestionsFiltersFiles(t *testing.T) {
	cwd := t.TempDir()
	if err := os.Mkdir(filepath.Join(cwd, "alpha"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cwd, "alpha.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := listPathSuggestions(cwd, cwd)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Value != filepath.Join(cwd, "alpha") {
		t.Fatalf("suggestions = %#v, want directory-only results", got)
	}
}

func TestListPathSuggestionsStopsAtGitRepo(t *testing.T) {
	cwd := t.TempDir()
	// sibling is a plain directory — should appear
	if err := os.Mkdir(filepath.Join(cwd, "plain"), 0o755); err != nil {
		t.Fatal(err)
	}
	// repo is a git repo — should appear as a suggestion but not expand inside
	repoDir := filepath.Join(cwd, "repo")
	if err := os.MkdirAll(filepath.Join(repoDir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(repoDir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}

	// browsing cwd — both plain and repo should appear
	got, err := listPathSuggestions(cwd, cwd)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("suggestions at cwd = %#v, want 2 entries", got)
	}

	// browsing inside the git repo — should return nothing
	got, err = listPathSuggestions(repoDir, cwd)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("suggestions inside git repo = %#v, want empty", got)
	}
}

func TestListPathSuggestionsAreCapped(t *testing.T) {
	cwd := t.TempDir()
	for i := range maxPathSuggestions + 3 {
		dir := filepath.Join(cwd, string(rune('a'+i)))
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	got, err := listPathSuggestions(cwd, cwd)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != maxPathSuggestions {
		t.Fatalf("suggestion count = %d, want %d", len(got), maxPathSuggestions)
	}
}
