package ui

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"treehouse/internal/config"
	"treehouse/internal/git"
	"treehouse/internal/launcher"

	tea "github.com/charmbracelet/bubbletea"
)

func TestEmptyRepoViewShowsOnboardingAndHints(t *testing.T) {
	m := newModel(config.Config{}, t.TempDir())
	view := m.View()
	if !strings.Contains(view, "No saved repos yet.") {
		t.Fatalf("view missing empty state: %s", view)
	}
	if !strings.Contains(view, "a add repo") {
		t.Fatalf("view missing hint bar: %s", view)
	}
}

func TestWorktreeViewShowsCountsAndHelp(t *testing.T) {
	cfg := config.Config{
		Repos: []config.RepoConfig{{
			ID:               "repo-1",
			Name:             "demo",
			RepoPath:         "/tmp/demo",
			WorktreeBasePath: "/tmp/worktrees/demo",
		}},
	}
	m := newModel(cfg, t.TempDir())
	repo := cfg.Repos[0]
	m.selectedRepo = &repo
	m.screen = screenWorktrees
	m.selectedState = git.RepoState{
		Root: "/tmp/demo",
		Worktrees: []git.WorktreeEntry{{
			Path:     "/tmp/demo-feature",
			Branch:   "feature/x",
			Added:    2,
			Modified: 3,
			Deleted:  1,
			Dirty:    true,
		}},
	}
	origHasSession := hasSession
	hasSession = func(name string) bool {
		return name == "demo-feature-x"
	}
	t.Cleanup(func() {
		hasSession = origHasSession
	})
	m.syncWorktrees()
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	m = next.(model)
	view := m.View()
	if !strings.Contains(view, "A:2 M:3 D:1") {
		t.Fatalf("view missing counts: %s", view)
	}
	if !strings.Contains(view, "tmux:on") {
		t.Fatalf("view missing tmux status: %s", view)
	}
	if !strings.Contains(view, "enter/o open") {
		t.Fatalf("view missing worktree hints: %s", view)
	}
	if !strings.Contains(view, "k close session") {
		t.Fatalf("view missing close-session hints: %s", view)
	}
}

func TestEnterOnEmptyRepoListShowsGuidance(t *testing.T) {
	m := newModel(config.Config{}, t.TempDir())
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := next.(model)
	if updated.message != "No repos saved. Press a to add one." {
		t.Fatalf("message = %q", updated.message)
	}
}

func TestContextualEnterMovesFromRepoListToWorktrees(t *testing.T) {
	cfg := config.Config{
		Repos: []config.RepoConfig{{
			ID:               "repo-1",
			Name:             "demo",
			RepoPath:         "/tmp/demo",
			WorktreeBasePath: "/tmp/worktrees/demo",
		}},
	}
	m := newModel(cfg, t.TempDir())
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := next.(model)
	if updated.screen != screenWorktrees {
		t.Fatalf("screen = %v, want worktrees", updated.screen)
	}
	if updated.selectedRepo == nil || updated.selectedRepo.Name != "demo" {
		t.Fatalf("selected repo = %#v", updated.selectedRepo)
	}
}

func TestHelpToggleRendersOverlay(t *testing.T) {
	m := newModel(config.Config{}, t.TempDir())
	next, _ := m.Update(tea.KeyMsg{Runes: []rune{'?'}, Type: tea.KeyRunes})
	updated := next.(model)
	if !updated.helpVisible {
		t.Fatal("expected help to be visible")
	}
	if !strings.Contains(updated.View(), "Help") {
		t.Fatalf("expected help overlay in view: %s", updated.View())
	}
}

func TestStartAddRepoSeedsRepoPathAndDefaults(t *testing.T) {
	cwd := filepath.Join(t.TempDir(), "repo-root")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}

	m := newModel(config.Config{}, cwd)
	m.startAddRepo()

	if got := m.inputs[0].Value(); got != cwd {
		t.Fatalf("repo path = %q, want %q", got, cwd)
	}
	if got := m.inputs[1].Value(); got != defaultWorktreeBasePath(cwd) {
		t.Fatalf("worktree base path = %q, want %q", got, defaultWorktreeBasePath(cwd))
	}
}

func TestAddRepoModalShowsDirectorySuggestions(t *testing.T) {
	cwd := t.TempDir()
	for _, dir := range []string{"alpha", "beta"} {
		if err := os.Mkdir(filepath.Join(cwd, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(cwd, "notes.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModel(config.Config{}, cwd)
	m.startAddRepo()
	m.activeInput = 0
	m.refreshPathSuggestions()

	view := m.renderModal()
	if !strings.Contains(view, "Directories") {
		t.Fatalf("modal missing directory header: %s", view)
	}
	if !strings.Contains(view, filepath.Join(cwd, "alpha")) || !strings.Contains(view, filepath.Join(cwd, "beta")) {
		t.Fatalf("modal missing directory suggestions: %s", view)
	}
	if strings.Contains(view, "notes.txt") {
		t.Fatalf("modal should not show files: %s", view)
	}
}

func TestSelectingRepoSuggestionUpdatesDefaults(t *testing.T) {
	cwd := t.TempDir()
	selectedRepo := filepath.Join(cwd, "alpha")
	if err := os.Mkdir(selectedRepo, 0o755); err != nil {
		t.Fatal(err)
	}

	m := newModel(config.Config{}, cwd)
	m.startAddRepo()
	m.activeInput = 0
	m.refreshPathSuggestions()
	if !m.acceptPathSuggestion() {
		t.Fatal("expected suggestion acceptance")
	}

	if got := m.inputs[0].Value(); got != selectedRepo {
		t.Fatalf("repo path = %q, want %q", got, selectedRepo)
	}
	if got := m.inputs[1].Value(); got != defaultWorktreeBasePath(selectedRepo) {
		t.Fatalf("worktree base path = %q, want %q", got, defaultWorktreeBasePath(selectedRepo))
	}
}

func TestManualWorktreeBasePathIsPreserved(t *testing.T) {
	cwd := t.TempDir()
	firstRepo := filepath.Join(cwd, "alpha")
	secondRepo := filepath.Join(cwd, "beta")
	for _, dir := range []string{firstRepo, secondRepo} {
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	m := newModel(config.Config{}, cwd)
	m.startAddRepo()
	m.inputs[1].SetValue("/tmp/custom-worktrees")
	m.worktreeBaseDirty = true
	m.inputs[0].SetValue(secondRepo)
	m.applyAddRepoDefaults()

	if got := m.inputs[1].Value(); got != "/tmp/custom-worktrees" {
		t.Fatalf("worktree base path = %q, want custom value", got)
	}
}

func TestOpenSelectedWorktreeCmdEnsuresSessionThenLaunchesTmux(t *testing.T) {
	cfg := config.Config{
		Repos: []config.RepoConfig{{
			ID:               "repo-1",
			Name:             "demo",
			RepoPath:         "/tmp/demo",
			WorktreeBasePath: "/tmp/worktrees/demo",
		}},
	}
	m := newModel(cfg, t.TempDir())
	repo := cfg.Repos[0]
	m.selectedRepo = &repo
	m.screen = screenWorktrees
	m.width = 120
	m.height = 40
	m.selectedState = git.RepoState{
		Worktrees: []git.WorktreeEntry{{
			Path:   "/tmp/demo-feature",
			Branch: "feature/x",
		}},
	}
	m.syncWorktrees()

	called := false
	origExecProcess := execProcess
	execProcess = func(cmd *exec.Cmd, fn tea.ExecCallback) tea.Cmd {
		return func() tea.Msg {
			called = true
			if got := cmd.Args; strings.Join(got, " ") != "tmux attach-session -t demo-feature-x" {
				t.Fatalf("attach args = %v", got)
			}
			return fn(nil)
		}
	}
	t.Cleanup(func() {
		execProcess = origExecProcess
	})

	origAttach := attachCommand
	attachCommand = func(session string) *exec.Cmd {
		if session != "demo-feature-x" {
			t.Fatalf("session = %q", session)
		}
		return exec.Command("tmux", "attach-session", "-t", session)
	}
	t.Cleanup(func() {
		attachCommand = origAttach
	})

	origEnsure := ensureSession
	ensureSession = func(repoName string, branch string, path string, size launcher.TerminalSize, mode launcher.LaunchMode) error {
		if repoName != "demo" || branch != "feature/x" || path != "/tmp/demo-feature" {
			t.Fatalf("unexpected ensure inputs: %q %q %q", repoName, branch, path)
		}
		if size.Width != 120 || size.Height != 40 {
			t.Fatalf("size = %#v", size)
		}
		return nil
	}
	t.Cleanup(func() {
		ensureSession = origEnsure
	})

	msg := m.openSelectedWorktreeCmd(launcher.ModeNormal)()
	launch, ok := msg.(launchTmuxMsg)
	if !ok {
		t.Fatalf("msg = %T, want launchTmuxMsg", msg)
	}
	if launch.session != "demo-feature-x" {
		t.Fatalf("session = %q", launch.session)
	}

	_, cmd := m.Update(msg)
	result := cmd()
	action, ok := result.(actionMsg)
	if !ok {
		t.Fatalf("result = %T, want actionMsg", result)
	}
	if action.err != nil || action.message != "tmux session ready" {
		t.Fatalf("action = %#v", action)
	}
	if !called {
		t.Fatal("expected execProcess to run")
	}
}

func TestOpenSelectedWorktreeCmdReturnsEnsureError(t *testing.T) {
	cfg := config.Config{
		Repos: []config.RepoConfig{{
			ID:               "repo-1",
			Name:             "demo",
			RepoPath:         "/tmp/demo",
			WorktreeBasePath: "/tmp/worktrees/demo",
		}},
	}
	m := newModel(cfg, t.TempDir())
	repo := cfg.Repos[0]
	m.selectedRepo = &repo
	m.screen = screenWorktrees
	m.selectedState = git.RepoState{
		Worktrees: []git.WorktreeEntry{{
			Path:   "/tmp/demo-feature",
			Branch: "feature/x",
		}},
	}
	m.syncWorktrees()

	wantErr := errors.New("tmux failed")
	origEnsure := ensureSession
	ensureSession = func(string, string, string, launcher.TerminalSize, launcher.LaunchMode) error {
		return wantErr
	}
	t.Cleanup(func() {
		ensureSession = origEnsure
	})

	msg := m.openSelectedWorktreeCmd(launcher.ModeNormal)()
	action, ok := msg.(actionMsg)
	if !ok {
		t.Fatalf("msg = %T, want actionMsg", msg)
	}
	if !errors.Is(action.err, wantErr) {
		t.Fatalf("err = %v, want %v", action.err, wantErr)
	}
}

func TestCloseSessionKeyShowsModalWhenSessionIsActive(t *testing.T) {
	cfg := config.Config{
		Repos: []config.RepoConfig{{
			ID:               "repo-1",
			Name:             "demo",
			RepoPath:         "/tmp/demo",
			WorktreeBasePath: "/tmp/worktrees/demo",
		}},
	}
	m := newModel(cfg, t.TempDir())
	repo := cfg.Repos[0]
	m.selectedRepo = &repo
	m.screen = screenWorktrees
	m.selectedState = git.RepoState{
		Worktrees: []git.WorktreeEntry{{
			Path:   "/tmp/demo-feature",
			Branch: "feature/x",
		}},
	}
	origHasSession := hasSession
	hasSession = func(string) bool {
		return true
	}
	t.Cleanup(func() {
		hasSession = origHasSession
	})
	m.syncWorktrees()

	next, _ := m.Update(tea.KeyMsg{Runes: []rune{'k'}, Type: tea.KeyRunes})
	updated := next.(model)

	if updated.modal != modalCloseSession {
		t.Fatalf("modal = %v, want modalCloseSession", updated.modal)
	}
	if !strings.Contains(updated.renderModal(), "session: demo-feature-x") {
		t.Fatalf("modal missing session name: %s", updated.renderModal())
	}
}

func TestCloseSessionKeyShowsMessageWhenSessionIsMissing(t *testing.T) {
	cfg := config.Config{
		Repos: []config.RepoConfig{{
			ID:               "repo-1",
			Name:             "demo",
			RepoPath:         "/tmp/demo",
			WorktreeBasePath: "/tmp/worktrees/demo",
		}},
	}
	m := newModel(cfg, t.TempDir())
	repo := cfg.Repos[0]
	m.selectedRepo = &repo
	m.screen = screenWorktrees
	m.selectedState = git.RepoState{
		Worktrees: []git.WorktreeEntry{{
			Path:   "/tmp/demo-feature",
			Branch: "feature/x",
		}},
	}
	origHasSession := hasSession
	hasSession = func(string) bool {
		return false
	}
	t.Cleanup(func() {
		hasSession = origHasSession
	})
	m.syncWorktrees()

	next, _ := m.Update(tea.KeyMsg{Runes: []rune{'k'}, Type: tea.KeyRunes})
	updated := next.(model)

	if updated.message != "No tmux session running for this worktree." {
		t.Fatalf("message = %q", updated.message)
	}
}

func TestCloseSelectedSessionCmdClosesTmuxSession(t *testing.T) {
	cfg := config.Config{
		Repos: []config.RepoConfig{{
			ID:               "repo-1",
			Name:             "demo",
			RepoPath:         "/tmp/demo",
			WorktreeBasePath: "/tmp/worktrees/demo",
		}},
	}
	m := newModel(cfg, t.TempDir())
	repo := cfg.Repos[0]
	m.selectedRepo = &repo
	m.screen = screenWorktrees
	m.selectedState = git.RepoState{
		Worktrees: []git.WorktreeEntry{{
			Path:   "/tmp/demo-feature",
			Branch: "feature/x",
		}},
	}
	origHasSession := hasSession
	hasSession = func(string) bool {
		return true
	}
	t.Cleanup(func() {
		hasSession = origHasSession
	})
	m.syncWorktrees()

	var got string
	origKillSession := killSession
	killSession = func(session string) error {
		got = session
		return nil
	}
	t.Cleanup(func() {
		killSession = origKillSession
	})

	msg := m.closeSelectedSessionCmd()()
	action, ok := msg.(actionMsg)
	if !ok {
		t.Fatalf("msg = %T, want actionMsg", msg)
	}
	if action.err != nil || action.message != "tmux session closed." {
		t.Fatalf("action = %#v", action)
	}
	if got != "demo-feature-x" {
		t.Fatalf("closed session = %q", got)
	}
}
