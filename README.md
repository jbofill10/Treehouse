# Treehouse 🏡

Terminal worktree manager for Git repos that launches `claude` and `nvim` in `tmux`.

## Requirements

- [git](https://git-scm.com/)
- [tmux](https://github.com/tmux/tmux/wiki/Installing)
- [nvim](https://neovim.io/)
- [Claude Code](https://docs.anthropic.com/en/docs/claude-code/overview) (`claude` CLI)

## Install

Download the binary for your platform from the [releases page](https://github.com/jbofill10/Treehouse/releases):

| Platform | File |
|----------|------|
| macOS (Apple Silicon) | `treehouse_*_darwin_arm64` |
| macOS (Intel) | `treehouse_*_darwin_amd64` |
| Linux (x86-64) | `treehouse_*_linux_amd64` |
| Linux (ARM64) | `treehouse_*_linux_arm64` |

Make it executable and move it onto your `PATH`:

```bash
chmod +x treehouse_*_darwin_arm64
mv treehouse_*_darwin_arm64 /usr/local/bin/treehouse
```

## Usage

1. Run `treehouse` — on first launch you'll see an empty repo list
2. Press `a` to add a repo: type the path, use `↑/↓` to browse, `→` to accept, `enter` to confirm
3. Select the repo with `enter` to open the worktree view
4. Press `c` to create a worktree (new or existing branch), then `enter` or `o` to open it in tmux

Each session opens three windows: `claude` (window 0), `nvim` (window 1), and a plain shell (window 2).

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

## Dev

```bash
go run ./cmd/treehouse
```

## Keybindings

- Global: `?` toggle help, `q` quit
- Repo list: `a` add repo, `enter` open repo
- Add repo modal: type repo path first, `tab` move fields, `↑/↓` browse directories, `→` accept directory, `enter` submit, `esc` cancel
- Worktree view: `enter` or `o` open, `k` close session, `c` create, `x` remove, `p` edit default path, `r` refresh, `b` back
