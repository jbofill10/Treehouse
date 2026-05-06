package ui

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type pathSuggestion struct {
	Value string
}

const maxPathSuggestions = 8

func resolveLaunchCwd(cwd string) string {
	if strings.TrimSpace(cwd) != "" {
		return cwd
	}
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

func defaultWorktreeBasePath(repoPath string) string {
	repoPath = strings.TrimSpace(repoPath)
	if repoPath == "" {
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "git", "worktrees", filepath.Base(filepath.Clean(repoPath)))
}

func listPathSuggestions(raw string, cwd string) ([]pathSuggestion, error) {
	browseDir, prefix, err := pathBrowseContext(raw, cwd)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(browseDir)
	if err != nil {
		return nil, err
	}

	lowerPrefix := strings.ToLower(prefix)
	suggestions := make([]pathSuggestion, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if lowerPrefix != "" && !strings.HasPrefix(strings.ToLower(name), lowerPrefix) {
			continue
		}
		suggestions = append(suggestions, pathSuggestion{
			Value: filepath.Join(browseDir, name),
		})
	}

	slices.SortFunc(suggestions, func(a, b pathSuggestion) int {
		return strings.Compare(strings.ToLower(filepath.Base(a.Value)), strings.ToLower(filepath.Base(b.Value)))
	})
	if len(suggestions) > maxPathSuggestions {
		suggestions = suggestions[:maxPathSuggestions]
	}
	return suggestions, nil
}

func pathBrowseContext(raw string, cwd string) (string, string, error) {
	resolved, err := resolvePathInput(raw, cwd)
	if err != nil {
		return "", "", err
	}

	info, err := os.Stat(resolved)
	if err == nil && info.IsDir() {
		return resolved, "", nil
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", "", err
	}

	search := resolved
	for {
		info, statErr := os.Stat(search)
		if statErr == nil && info.IsDir() {
			remainder := strings.TrimPrefix(resolved, search)
			remainder = strings.TrimPrefix(remainder, string(os.PathSeparator))
			if remainder == "" {
				return search, "", nil
			}
			prefix := remainder
			if idx := strings.IndexRune(remainder, os.PathSeparator); idx >= 0 {
				prefix = remainder[:idx]
			}
			return search, prefix, nil
		}
		if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
			return "", "", statErr
		}

		parent := filepath.Dir(search)
		if parent == search {
			break
		}
		search = parent
	}

	return cwd, "", nil
}

func resolvePathInput(raw string, cwd string) (string, error) {
	path := strings.TrimSpace(raw)
	if path == "" {
		return filepath.Clean(resolveLaunchCwd(cwd)), nil
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
	if !filepath.IsAbs(path) {
		path = filepath.Join(resolveLaunchCwd(cwd), path)
	}
	return filepath.Clean(path), nil
}
