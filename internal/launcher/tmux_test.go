package launcher

import (
	"errors"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestSessionName(t *testing.T) {
	got := SessionName("My Repo", "feature/test")
	if got != "my-repo-feature-test" {
		t.Fatalf("got %q", got)
	}
}

func TestDisplayTitle(t *testing.T) {
	got := DisplayTitle("My Repo", "feature/test")
	if got != "My Repo:feature/test" {
		t.Fatalf("got %q", got)
	}
}

func TestDisplayTitleFallsBackWhenValuesMissing(t *testing.T) {
	tests := []struct {
		name   string
		repo   string
		branch string
		want   string
	}{
		{name: "repo only", repo: "demo", want: "demo"},
		{name: "branch only", branch: "feature/test", want: "feature/test"},
		{name: "empty", want: "treehouse"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DisplayTitle(tt.repo, tt.branch); got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestSuggestedTargetPath(t *testing.T) {
	got := SuggestedTargetPath("/tmp/worktrees", "feature/test")
	want := filepath.Join("/tmp/worktrees", "feature-test")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestNewSessionCommandIncludesSizeWhenProvided(t *testing.T) {
	t.Setenv("SHELL", "sh")
	cmd := newSessionCommand("demo-feature", "demo:feature/test", "/tmp/demo-feature", TerminalSize{Width: 120, Height: 40}, ModeNormal)
	want := []string{"tmux", "new-session", "-d", "-s", "demo-feature", "-c", "/tmp/demo-feature", "-x", "120", "-y", "40", "-n", "claude", "sh", "-lc", "printf '\\033]2;%s\\007' \"demo:feature/test\"; clear; { cd \"/tmp/demo-feature\" && exec \"claude\"; } || exec \"sh\""}
	if !reflect.DeepEqual(cmd.Args, want) {
		t.Fatalf("args = %#v want %#v", cmd.Args, want)
	}
}

func TestNewSessionCommandOmitsSizeWhenUnavailable(t *testing.T) {
	t.Setenv("SHELL", "sh")
	cmd := newSessionCommand("demo-feature", "demo:feature/test", "/tmp/demo-feature", TerminalSize{}, ModeNormal)
	want := []string{"tmux", "new-session", "-d", "-s", "demo-feature", "-c", "/tmp/demo-feature", "-n", "claude", "sh", "-lc", "printf '\\033]2;%s\\007' \"demo:feature/test\"; clear; { cd \"/tmp/demo-feature\" && exec \"claude\"; } || exec \"sh\""}
	if !reflect.DeepEqual(cmd.Args, want) {
		t.Fatalf("args = %#v want %#v", cmd.Args, want)
	}
}

func TestNewSessionCommandDangerousMode(t *testing.T) {
	t.Setenv("SHELL", "sh")
	cmd := newSessionCommand("demo-feature", "demo:feature/test", "/tmp/demo-feature", TerminalSize{}, ModeDangerous)
	want := []string{"tmux", "new-session", "-d", "-s", "demo-feature", "-c", "/tmp/demo-feature", "-n", "claude", "sh", "-lc", "printf '\\033]2;%s\\007' \"demo:feature/test\"; clear; { cd \"/tmp/demo-feature\" && exec \"claude\" \"--dangerously-skip-permissions\"; } || exec \"sh\""}
	if !reflect.DeepEqual(cmd.Args, want) {
		t.Fatalf("args = %#v want %#v", cmd.Args, want)
	}
}

func TestNewWindowCommandUsesDisplayTitle(t *testing.T) {
	t.Setenv("SHELL", "sh")
	cmd := newWindowCommand("demo-feature", "demo:feature/test", "/tmp/demo-feature")
	want := []string{"tmux", "new-window", "-t", "demo-feature:1", "-c", "/tmp/demo-feature", "-n", "nvim", "sh", "-lc", "printf '\\033]2;%s\\007' \"demo:feature/test\"; clear; { cd \"/tmp/demo-feature\" && exec \"nvim\"; } || exec \"sh\""}
	if !reflect.DeepEqual(cmd.Args, want) {
		t.Fatalf("args = %#v want %#v", cmd.Args, want)
	}
}

func TestNewSessionCommandUsesUserShell(t *testing.T) {
	t.Setenv("SHELL", "/bin/zsh")
	cmd := newSessionCommand("demo-feature", "demo:feature/test", "/tmp/demo-feature", TerminalSize{}, ModeNormal)
	if len(cmd.Args) < 10 || cmd.Args[9] != "/bin/zsh" {
		t.Fatalf("expected /bin/zsh as shell, args = %#v", cmd.Args)
	}
}

func TestNewWindowCommandUsesUserShell(t *testing.T) {
	t.Setenv("SHELL", "/bin/zsh")
	cmd := newWindowCommand("demo-feature", "demo:feature/test", "/tmp/demo-feature")
	if len(cmd.Args) < 9 || cmd.Args[8] != "/bin/zsh" {
		t.Fatalf("expected /bin/zsh as shell, args = %#v", cmd.Args)
	}
}

func TestNewShellWindowCommand(t *testing.T) {
	t.Setenv("SHELL", "sh")
	cmd := newShellWindowCommand("demo-feature", "/tmp/demo-feature")
	want := []string{"tmux", "new-window", "-t", "demo-feature:2", "-c", "/tmp/demo-feature", "-n", "shell", "sh", "-l"}
	if !reflect.DeepEqual(cmd.Args, want) {
		t.Fatalf("args = %#v want %#v", cmd.Args, want)
	}
}

func TestLoginShellFallsBackToSh(t *testing.T) {
	t.Setenv("SHELL", "")
	if got := defaultLoginShell(); got != "sh" {
		t.Fatalf("got %q want %q", got, "sh")
	}
}

func TestSetTitlesCommandUsesSessionTitle(t *testing.T) {
	cmd := setTitlesCommand("demo-feature", "demo:feature/test")
	want := []string{"tmux", "set-option", "-t", "demo-feature", "set-titles", "on", ";", "set-option", "-t", "demo-feature", "set-titles-string", "demo:feature/test"}
	if !reflect.DeepEqual(cmd.Args, want) {
		t.Fatalf("args = %#v want %#v", cmd.Args, want)
	}
}

func TestShellLaunchCommand(t *testing.T) {
	t.Setenv("SHELL", "/bin/zsh")
	got := shellLaunchCommand("demo:feature/test", "claude", nil, "/tmp/demo-feature")
	want := "printf '\\033]2;%s\\007' \"demo:feature/test\"; clear; { cd \"/tmp/demo-feature\" && exec \"claude\"; } || exec \"/bin/zsh\""
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestShellLaunchCommandWithExtraArgs(t *testing.T) {
	t.Setenv("SHELL", "/bin/zsh")
	got := shellLaunchCommand("demo:feature/test", "claude", []string{"--dangerously-skip-permissions"}, "/tmp/demo-feature")
	want := "printf '\\033]2;%s\\007' \"demo:feature/test\"; clear; { cd \"/tmp/demo-feature\" && exec \"claude\" \"--dangerously-skip-permissions\"; } || exec \"/bin/zsh\""
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestAttachCommandOutsideTmux(t *testing.T) {
	t.Setenv("TMUX", "")
	cmd := AttachCommand("demo-feature")
	want := []string{"tmux", "attach-session", "-t", "demo-feature"}
	if !reflect.DeepEqual(cmd.Args, want) {
		t.Fatalf("args = %#v want %#v", cmd.Args, want)
	}
}

func TestAttachCommandInsideTmux(t *testing.T) {
	t.Setenv("TMUX", "/tmp/tmux-123/default,1,0")
	cmd := AttachCommand("demo-feature")
	want := []string{"tmux", "switch-client", "-t", "demo-feature"}
	if !reflect.DeepEqual(cmd.Args, want) {
		t.Fatalf("args = %#v want %#v", cmd.Args, want)
	}
}

func TestKillSessionSkipsMissingSession(t *testing.T) {
	origRunCmd := runCmd
	runCmd = func(cmd *exec.Cmd) error {
		t.Fatalf("runCmd should not be called when session is missing: %v", cmd.Args)
		return nil
	}
	t.Cleanup(func() {
		runCmd = origRunCmd
	})
	origSleep := sleep
	sleep = func(time.Duration) {}
	t.Cleanup(func() {
		sleep = origSleep
	})

	origHasSession := HasSession
	HasSession = func(string) bool {
		return false
	}
	t.Cleanup(func() {
		HasSession = origHasSession
	})

	if err := KillSession("demo-feature"); err != nil {
		t.Fatalf("KillSession() error = %v", err)
	}
}

func TestKillSessionDetachesThenKills(t *testing.T) {
	origRunCmd := runCmd
	var got [][]string
	runCmd = func(cmd *exec.Cmd) error {
		got = append(got, cmd.Args)
		return nil
	}
	t.Cleanup(func() {
		runCmd = origRunCmd
	})
	origSleep := sleep
	sleep = func(time.Duration) {}
	t.Cleanup(func() {
		sleep = origSleep
	})

	origHasSession := HasSession
	checks := 0
	HasSession = func(string) bool {
		checks++
		if checks >= 2 {
			return false
		}
		return true
	}
	t.Cleanup(func() {
		HasSession = origHasSession
	})

	if err := KillSession("demo-feature"); err != nil {
		t.Fatalf("KillSession() error = %v", err)
	}

	want := [][]string{
		{"tmux", "detach-client", "-s", "demo-feature"},
		{"tmux", "kill-session", "-t", "demo-feature"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("commands = %#v want %#v", got, want)
	}
}

func TestKillSessionIgnoresDetachWithoutAttachedClients(t *testing.T) {
	origRunCmd := runCmd
	var got [][]string
	runCmd = func(cmd *exec.Cmd) error {
		got = append(got, cmd.Args)
		if len(got) == 1 {
			return &exec.ExitError{}
		}
		return nil
	}
	t.Cleanup(func() {
		runCmd = origRunCmd
	})
	origSleep := sleep
	sleep = func(time.Duration) {}
	t.Cleanup(func() {
		sleep = origSleep
	})

	origHasSession := HasSession
	checks := 0
	HasSession = func(string) bool {
		checks++
		if checks >= 2 {
			return false
		}
		return true
	}
	t.Cleanup(func() {
		HasSession = origHasSession
	})

	if err := KillSession("demo-feature"); err != nil {
		t.Fatalf("KillSession() error = %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("commands = %#v, want detach and kill", got)
	}
}

func TestKillSessionReturnsKillError(t *testing.T) {
	origRunCmd := runCmd
	runCmd = func(cmd *exec.Cmd) error {
		if reflect.DeepEqual(cmd.Args, []string{"tmux", "kill-session", "-t", "demo-feature"}) {
			return errors.New("kill failed")
		}
		return nil
	}
	t.Cleanup(func() {
		runCmd = origRunCmd
	})
	origSleep := sleep
	sleep = func(time.Duration) {}
	t.Cleanup(func() {
		sleep = origSleep
	})

	origHasSession := HasSession
	HasSession = func(string) bool {
		return true
	}
	t.Cleanup(func() {
		HasSession = origHasSession
	})

	if err := KillSession("demo-feature"); err == nil || err.Error() != "kill failed" {
		t.Fatalf("KillSession() error = %v", err)
	}
}

func TestKillSessionWaitsForShutdown(t *testing.T) {
	origRunCmd := runCmd
	runCmd = func(cmd *exec.Cmd) error {
		return nil
	}
	t.Cleanup(func() {
		runCmd = origRunCmd
	})

	origSleep := sleep
	sleepCalls := 0
	sleep = func(time.Duration) {
		sleepCalls++
	}
	t.Cleanup(func() {
		sleep = origSleep
	})

	origHasSession := HasSession
	checks := 0
	HasSession = func(string) bool {
		checks++
		return checks < 4
	}
	t.Cleanup(func() {
		HasSession = origHasSession
	})

	if err := KillSession("demo-feature"); err != nil {
		t.Fatalf("KillSession() error = %v", err)
	}
	if sleepCalls != 2 {
		t.Fatalf("sleepCalls = %d, want 2", sleepCalls)
	}
}

func TestKillSessionReturnsErrorWhenShutdownDoesNotFinish(t *testing.T) {
	origRunCmd := runCmd
	runCmd = func(cmd *exec.Cmd) error {
		return nil
	}
	t.Cleanup(func() {
		runCmd = origRunCmd
	})

	origSleep := sleep
	sleep = func(time.Duration) {}
	t.Cleanup(func() {
		sleep = origSleep
	})

	origHasSession := HasSession
	HasSession = func(string) bool {
		return true
	}
	t.Cleanup(func() {
		HasSession = origHasSession
	})

	err := KillSession("demo-feature")
	if err == nil || err.Error() != `tmux session "demo-feature" is still shutting down` {
		t.Fatalf("KillSession() error = %v", err)
	}
}
