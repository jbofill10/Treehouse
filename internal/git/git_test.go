package git

import "testing"

func TestParseWorktreePorcelain(t *testing.T) {
	raw := `worktree /repo
HEAD abc
branch refs/heads/main

worktree /tmp/repo-feature
HEAD def
branch refs/heads/feature/x

worktree /tmp/repo-detached
HEAD 123
detached
`
	entries, err := parseWorktreePorcelain(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("len = %d, want 3", len(entries))
	}
	if entries[0].Branch != "main" {
		t.Fatalf("branch[0] = %q", entries[0].Branch)
	}
	if entries[1].Branch != "feature/x" {
		t.Fatalf("branch[1] = %q", entries[1].Branch)
	}
	if !entries[2].IsDetached {
		t.Fatalf("detached = false")
	}
}

func TestWorktreeStatusCounts(t *testing.T) {
	out := " M tracked.txt\nA  staged.txt\n D gone.txt\n?? new.txt\nR  renamed.txt\n"
	status, err := parseStatusCounts(out)
	if err != nil {
		t.Fatal(err)
	}
	if !status.Dirty {
		t.Fatal("expected dirty status")
	}
	if status.Added != 2 {
		t.Fatalf("added = %d, want 2", status.Added)
	}
	if status.Modified != 2 {
		t.Fatalf("modified = %d, want 2", status.Modified)
	}
	if status.Deleted != 1 {
		t.Fatalf("deleted = %d, want 1", status.Deleted)
	}
}
