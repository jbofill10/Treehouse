# Treehouse 🏡

Terminal worktree manager for Git repos that launches `claude` and `nvim` in `tmux`.

## Features

- Save multiple repos and a default worktree base path per repo
- Add repos by typing a path first, with directory suggestions and auto-filled defaults
- Inspect the main repo and linked worktrees with branch state and compact A/M/D change counts
- See whether each worktree already has a running `tmux` session
- Create a worktree from a new branch or an existing branch
- Remove worktrees safely
- Close a running worktree `tmux` session from the worktree list
- Open a worktree in a dedicated `tmux` session with:
  - window 1: `claude`
  - window 2: `nvim`
- Always-visible key hints and first-run onboarding in the TUI

## Run

```bash
go run ./cmd/treehouse
```

## Keybindings

- Global: `?` toggle help, `q` quit
- Repo list: `a` add repo, `enter` open repo
- Add repo modal: type repo path first, `tab` move fields, `↑/↓` browse directories, `→` accept directory, `enter` submit, `esc` cancel
- Worktree view: `enter` or `o` open, `k` close session, `c` create, `x` remove, `p` edit default path, `r` refresh, `b` back
