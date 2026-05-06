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
