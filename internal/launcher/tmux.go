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

type LaunchMode int

const (
	ModeNormal    LaunchMode = iota
	ModeDangerous            // passes --dangerously-skip-permissions to claude
)

func (m LaunchMode) ClaudeArgs() []string {
	if m == ModeDangerous {
		return []string{"--dangerously-skip-permissions"}
	}
	return nil
}

func (m LaunchMode) Label() string {
	if m == ModeDangerous {
		return "dangerous"
	}
	return "normal"
}

var (
	HasSession = func(name string) bool {
		cmd := exec.Command("tmux", "has-session", "-t", name)
		return cmd.Run() == nil
	}
	runCmd = func(cmd *exec.Cmd) error {
		return cmd.Run()
	}
	sleep      = time.Sleep
	loginShell = defaultLoginShell

	resolvedClaudePath = "claude"
	resolvedNvimPath   = "nvim"
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

	// claudeStartupModelCmd is sent to every new claude window after startup.
	// Enterprise managed settings can pin the model above the user settings level,
	// making --model and ANTHROPIC_MODEL ineffective. Issuing /model in-session is
	// the only way to land on opusplan.
	claudeStartupModelCmd = "/model opusplan"
	claudeStartupDelaySec = 2 // seconds to wait for the claude TUI to accept input
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
		return "treehouse"
	}
	return name
}

func DisplayTitle(repoName string, branch string) string {
	repo := strings.TrimSpace(repoName)
	branch = strings.TrimSpace(branch)

	switch {
	case repo == "" && branch == "":
		return "treehouse"
	case repo == "":
		return branch
	case branch == "":
		return repo
	default:
		return repo + ":" + branch
	}
}

func EnsureSession(repoName string, branch string, path string, size TerminalSize, mode LaunchMode) error {
	session := SessionName(repoName, branch)
	title := DisplayTitle(repoName, branch)
	if HasSession(session) {
		return nil
	}

	if err := runCmd(newSessionCommand(session, title, path, size, mode)); err != nil {
		return err
	}
	if err := runCmd(newWindowCommand(session, title, path)); err != nil {
		return err
	}
	if err := runCmd(newShellWindowCommand(session, path)); err != nil {
		return err
	}
	// set-titles is a global option; ignore failure on configs that disallow it
	_ = runCmd(setTitlesCommand(session, title))
	if err := runCmd(exec.Command("tmux", "select-window", "-t", session+":0")); err != nil {
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
	for range sessionPollAttempts {
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

// startupKeysScript returns a backgrounded subshell that waits for the claude
// TUI to initialize, then types the model-selection command. The subshell is
// forked before exec replaces the login shell, so it survives the process swap.
func startupKeysScript(session string) string {
	return fmt.Sprintf("( sleep %d; tmux send-keys -t %s %s Enter ) & ",
		claudeStartupDelaySec, strconv.Quote(session+":0"), strconv.Quote(claudeStartupModelCmd))
}

func newSessionCommand(session string, title string, path string, size TerminalSize, mode LaunchMode) *exec.Cmd {
	args := []string{"new-session", "-d", "-s", session, "-c", path}
	if size.Width > 0 && size.Height > 0 {
		args = append(args, "-x", fmt.Sprintf("%d", size.Width), "-y", fmt.Sprintf("%d", size.Height))
	}
	launch := startupKeysScript(session) + shellLaunchCommand(title, resolvedClaudePath, mode.ClaudeArgs(), path)
	args = append(args, "-n", "claude", loginShell(), "-lc", launch)
	return exec.Command("tmux", args...)
}

func newWindowCommand(session string, title string, path string) *exec.Cmd {
	return exec.Command("tmux", "new-window", "-t", session+":1", "-c", path, "-n", "nvim", loginShell(), "-lc", shellLaunchCommand(title, resolvedNvimPath, nil, path))
}

func newShellWindowCommand(session string, path string) *exec.Cmd {
	return exec.Command("tmux", "new-window", "-t", session+":2", "-c", path, "-n", "shell", loginShell(), "-l")
}

func setTitlesCommand(session string, title string) *exec.Cmd {
	return exec.Command("tmux", "set-option", "-t", session, "set-titles", "on", ";", "set-option", "-t", session, "set-titles-string", title)
}

func shellLaunchCommand(title string, program string, extraArgs []string, path string) string {
	quotedTitle := strconv.Quote(title)
	quotedPath := strconv.Quote(path)

	var execBuilder strings.Builder
	execBuilder.WriteString(strconv.Quote(program))
	for _, a := range extraArgs {
		execBuilder.WriteString(" ")
		execBuilder.WriteString(strconv.Quote(a))
	}
	execParts := execBuilder.String()

	// On failure (bad path, binary not found, etc.) drop into an interactive shell
	// so the error is visible instead of the window silently closing.
	return fmt.Sprintf(
		"printf '\\033]2;%%s\\007' %s; clear; { cd %s && exec %s; } || exec %s",
		quotedTitle, quotedPath, execParts, strconv.Quote(loginShell()),
	)
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
		p, err := exec.LookPath(tool)
		if err != nil {
			return fmt.Errorf("%s not found in PATH", tool)
		}
		switch tool {
		case "claude":
			resolvedClaudePath = p
		case "nvim":
			resolvedNvimPath = p
		}
	}
	return nil
}
