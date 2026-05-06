package git

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type WorktreeEntry struct {
	Path       string
	Branch     string
	IsMain     bool
	IsDetached bool
	Dirty      bool
	Added      int
	Modified   int
	Deleted    int
}

type RepoState struct {
	Root      string
	Branch    string
	Worktrees []WorktreeEntry
}

type CreateWorktreeRequest struct {
	RepoPath   string
	Mode       string
	BaseBranch string
	BranchName string
	TargetPath string
	Detach     bool
}

func InspectRepo(repoPath string) (RepoState, error) {
	root, err := runGit(repoPath, "rev-parse", "--show-toplevel")
	if err != nil {
		return RepoState{}, fmt.Errorf("resolve repo root: %w", err)
	}
	root = strings.TrimSpace(root)
	branch, _ := currentBranch(root)
	worktrees, err := listWorktrees(root)
	if err != nil {
		return RepoState{}, err
	}
	return RepoState{
		Root:      root,
		Branch:    branch,
		Worktrees: worktrees,
	}, nil
}

func ListBranches(repoPath string) ([]string, error) {
	out, err := runGit(repoPath, "for-each-ref", "--format=%(refname:short)", "refs/heads")
	if err != nil {
		return nil, err
	}
	var branches []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			branches = append(branches, line)
		}
	}
	return branches, nil
}

func CreateWorktree(req CreateWorktreeRequest) error {
	if req.BranchName == "" {
		return errors.New("branch name is required")
	}
	target := req.TargetPath
	args := []string{"worktree", "add"}
	switch req.Mode {
	case "existing":
		args = append(args, target, req.BranchName)
	case "new":
		base := req.BaseBranch
		if base == "" {
			base = "HEAD"
		}
		args = append(args, "-b", req.BranchName, target, base)
	default:
		return fmt.Errorf("unsupported create mode %q", req.Mode)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create parent dir: %w", err)
	}
	_, err := runGit(req.RepoPath, args...)
	return err
}

func RemoveWorktree(repoPath string, worktreePath string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, worktreePath)
	_, err := runGit(repoPath, args...)
	return err
}

func currentBranch(repoPath string) (string, error) {
	out, err := runGit(repoPath, "branch", "--show-current")
	return strings.TrimSpace(out), err
}

func listWorktrees(repoPath string) ([]WorktreeEntry, error) {
	out, err := runGit(repoPath, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("list worktrees: %w", err)
	}
	entries, err := parseWorktreePorcelain(out)
	if err != nil {
		return nil, err
	}
	for i := range entries {
		status, err := worktreeStatus(entries[i].Path)
		if err == nil {
			entries[i].Dirty = status.Dirty
			entries[i].Added = status.Added
			entries[i].Modified = status.Modified
			entries[i].Deleted = status.Deleted
		}
	}
	mainRoot := repoPath
	for i := range entries {
		entries[i].IsMain = filepath.Clean(entries[i].Path) == filepath.Clean(mainRoot)
	}
	return entries, nil
}

func parseWorktreePorcelain(raw string) ([]WorktreeEntry, error) {
	chunks := strings.Split(strings.TrimSpace(raw), "\n\n")
	entries := make([]WorktreeEntry, 0, len(chunks))
	for _, chunk := range chunks {
		if strings.TrimSpace(chunk) == "" {
			continue
		}
		var entry WorktreeEntry
		for _, line := range strings.Split(chunk, "\n") {
			switch {
			case strings.HasPrefix(line, "worktree "):
				entry.Path = strings.TrimSpace(strings.TrimPrefix(line, "worktree "))
			case strings.HasPrefix(line, "branch refs/heads/"):
				entry.Branch = strings.TrimSpace(strings.TrimPrefix(line, "branch refs/heads/"))
			case strings.HasPrefix(line, "branch "):
				entry.Branch = strings.TrimSpace(strings.TrimPrefix(line, "branch "))
			case line == "detached":
				entry.IsDetached = true
				if entry.Branch == "" {
					entry.Branch = "(detached)"
				}
			}
		}
		if entry.Path == "" {
			return nil, errors.New("worktree entry missing path")
		}
		if entry.Branch == "" && !entry.IsDetached {
			entry.Branch = "(unknown)"
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

type StatusCounts struct {
	Dirty    bool
	Added    int
	Modified int
	Deleted  int
}

func worktreeStatus(path string) (StatusCounts, error) {
	out, err := runGit(path, "status", "--porcelain")
	if err != nil {
		return StatusCounts{}, err
	}
	return parseStatusCounts(out)
}

func parseStatusCounts(out string) (StatusCounts, error) {
	status := StatusCounts{}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		status.Dirty = true
		if strings.HasPrefix(line, "??") {
			status.Added++
			continue
		}
		if len(line) < 2 {
			continue
		}
		x := line[0]
		y := line[1]
		switch {
		case x == 'A' || y == 'A' || x == 'C' || y == 'C':
			status.Added++
		case x == 'D' || y == 'D':
			status.Deleted++
		case x != ' ' || y != ' ':
			status.Modified++
		}
	}
	return status, nil
}

func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return stdout.String(), nil
}
