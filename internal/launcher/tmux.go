package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var invalidSessionChars = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

var (
	HasSession = func(name string) bool {
		cmd := exec.Command("tmux", "has-session", "-t", name)
		return cmd.Run() == nil
	}
	runCmd = func(cmd *exec.Cmd) error {
		return cmd.Run()
	}
	sleep     = time.Sleep
	loginShell = defaultLoginShell
)

func defaultLoginShell() string {
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}
	return "sh"
}

const (
	sessionPollInterval = 25 * time.Millisecond
	sessionPollAttempts = 20
)

type TerminalSize struct {
	Width  int
	Height int
}

func SessionName(repoName string, branch string) string {
	name := strings.ToLower(strings.TrimSpace(repoName + "-" + branch))
	name = invalidSessionChars.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")
	if name == "" {
		return "claude-manager"
	}
	return name
}

func DisplayTitle(repoName string, branch string) string {
	repo := strings.TrimSpace(repoName)
	branch = strings.TrimSpace(branch)

	switch {
	case repo == "" && branch == "":
		return "claude-manager"
	case repo == "":
		return branch
	case branch == "":
		return repo
	default:
		return repo + ":" + branch
	}
}

func EnsureSession(repoName string, branch string, path string, size TerminalSize) error {
	session := SessionName(repoName, branch)
	title := DisplayTitle(repoName, branch)
	if HasSession(session) {
		return nil
	}

	if err := runCmd(newSessionCommand(session, title, path, size)); err != nil {
		return err
	}
	if err := runCmd(newWindowCommand(session, title, path)); err != nil {
		return err
	}
	if err := runCmd(setTitlesCommand(session, title)); err != nil {
		return err
	}
	if err := runCmd(selectWindowCommand(session)); err != nil {
		return err
	}
	return nil
}

func KillSession(name string) error {
	if !HasSession(name) {
		return nil
	}
	if err := runCmd(exec.Command("tmux", "detach-client", "-s", name)); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() != 0 {
			// `detach-client -s` fails when no clients are attached; the session can still be closed.
		} else {
			return err
		}
	}
	if err := runCmd(exec.Command("tmux", "kill-session", "-t", name)); err != nil {
		return err
	}
	return waitForSessionGone(name)
}

func waitForSessionGone(name string) error {
	for attempt := 0; attempt < sessionPollAttempts; attempt++ {
		if !HasSession(name) {
			return nil
		}
		sleep(sessionPollInterval)
	}
	return fmt.Errorf("tmux session %q is still shutting down", name)
}

func AttachCommand(name string) *exec.Cmd {
	if insideTmux() {
		return exec.Command("tmux", "switch-client", "-t", name)
	}
	return exec.Command("tmux", "attach-session", "-t", name)
}

func insideTmux() bool {
	return strings.TrimSpace(os.Getenv("TMUX")) != ""
}

func newSessionCommand(session string, title string, path string, size TerminalSize) *exec.Cmd {
	args := []string{"new-session", "-d", "-s", session, "-c", path}
	if size.Width > 0 && size.Height > 0 {
		args = append(args, "-x", fmt.Sprintf("%d", size.Width), "-y", fmt.Sprintf("%d", size.Height))
	}
	args = append(args, "-n", title, loginShell(), "-lc", shellLaunchCommand(title, "claude", path))
	return exec.Command("tmux", args...)
}

func newWindowCommand(session string, title string, path string) *exec.Cmd {
	return exec.Command("tmux", "new-window", "-t", session+":2", "-c", path, "-n", title+" [nvim]", loginShell(), "-lc", shellLaunchCommand(title, "nvim", path))
}

func selectWindowCommand(session string) *exec.Cmd {
	return exec.Command("tmux", "select-window", "-t", session+":1")
}

func setTitlesCommand(session string, title string) *exec.Cmd {
	return exec.Command("tmux", "set-option", "-t", session, "set-titles", "on", ";", "set-option", "-t", session, "set-titles-string", title)
}

func shellLaunchCommand(title string, program string, path string) string {
	quotedTitle := strconv.Quote(title)
	quotedProgram := strconv.Quote(program)
	quotedPath := strconv.Quote(path)

	return fmt.Sprintf("printf '\\033]2;%%s\\007' %s; cd %s && exec %s", quotedTitle, quotedPath, quotedProgram)
}

func SuggestedTargetPath(basePath string, branch string) string {
	slug := invalidSessionChars.ReplaceAllString(strings.TrimSpace(branch), "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "worktree"
	}
	return filepath.Join(basePath, slug)
}

func ValidateRuntime() error {
	for _, tool := range []string{"git", "tmux", "nvim", "claude"} {
		if _, err := exec.LookPath(tool); err != nil {
			return fmt.Errorf("%s not found in PATH", tool)
		}
	}
	return nil
}
