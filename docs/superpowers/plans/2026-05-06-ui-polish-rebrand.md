# UI Polish + Treehouse Rebrand Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Apply a warm/earthy colour palette consistently across all TUI surfaces, centralise styles into a theme struct, restyle the list delegate, and rebrand every "claude-manager" remnant to "Treehouse" (including a one-time config-dir migration).

**Architecture:** Extract all lipgloss styles into `internal/ui/styles.go` as a package-level `theme` struct and a `newListDelegate()` helper; update every call site in `app.go`; perform a module-path and filesystem rename for the rebrand; add a silent `migrateLegacyConfigDir()` call at the top of `config.Load()`.

**Tech Stack:** Go, Bubble Tea (`github.com/charmbracelet/bubbletea`), Lipgloss (`github.com/charmbracelet/lipgloss`), Bubbles list (`github.com/charmbracelet/bubbles/list`)

---

## File Map

| Action | Path | Responsibility |
|---|---|---|
| Modify | `go.mod` | Module path `claude-manager` → `treehouse` |
| Rename dir | `cmd/claude-manager/` → `cmd/treehouse/` | Binary entry point |
| Modify | `cmd/treehouse/main.go` | Fix import path |
| Modify | `README.md` | Name + run command |
| Modify | `internal/ui/app.go` | Remove old style var block, update all references, list delegate, title string, import paths |
| **Create** | `internal/ui/styles.go` | `theme` struct + `newListDelegate()` |
| Modify | `internal/launcher/tmux.go` | Fallback strings `claude-manager` → `treehouse` |
| Modify | `internal/launcher/tmux_test.go` | Test expectation for empty-fallback case |
| Modify | `internal/config/config.go` | `appDir` rename + `migrateLegacyConfigDir()` |
| Modify | `internal/config/config_test.go` | Add `TestMigrateLegacyConfigDir` |

---

## Task 1: Rename the Go module and cmd directory

**Files:**
- Modify: `go.mod:1`
- Rename: `cmd/claude-manager/` → `cmd/treehouse/`
- Modify: `cmd/treehouse/main.go:6`

- [ ] **Step 1: Update the module declaration in go.mod**

Change line 1 from:
```
module claude-manager
```
to:
```
module treehouse
```

- [ ] **Step 2: Rename the cmd directory**

```bash
git mv cmd/claude-manager cmd/treehouse
```

- [ ] **Step 3: Fix the import in cmd/treehouse/main.go**

Open `cmd/treehouse/main.go`. Change the import:
```go
// Before
"claude-manager/internal/ui"

// After
"treehouse/internal/ui"
```

- [ ] **Step 4: Fix imports in internal/ui/app.go (lines 10-12)**

```go
// Before
"claude-manager/internal/config"
"claude-manager/internal/git"
"claude-manager/internal/launcher"

// After
"treehouse/internal/config"
"treehouse/internal/git"
"treehouse/internal/launcher"
```

- [ ] **Step 5: Fix imports in internal/ui/app_test.go**

Same three imports — change `claude-manager` → `treehouse` for each.

- [ ] **Step 6: Build to confirm no broken imports**

```bash
go build ./...
```
Expected: clean exit, no output.

- [ ] **Step 7: Commit**

```bash
git add go.mod cmd/treehouse internal/ui/app.go internal/ui/app_test.go
git commit -m "refactor: rename Go module and cmd dir to treehouse"
```

---

## Task 2: Rebrand user-visible strings

**Files:**
- Modify: `internal/ui/app.go:458`
- Modify: `internal/launcher/tmux.go:53,64`
- Modify: `internal/launcher/tmux_test.go:35`
- Modify: `README.md`

- [ ] **Step 1: Update the rendered TUI title in app.go**

At `app.go:458` change:
```go
// Before
titleStyle.Render("Claude Manager"),

// After
titleStyle.Render("Treehouse"),
```

- [ ] **Step 2: Update the tmux session fallback in launcher/tmux.go**

At line 53 (empty `SessionName` fallback):
```go
// Before
return "claude-manager"

// After
return "treehouse"
```

At line 64 (empty `DisplayTitle` fallback):
```go
// Before
return "claude-manager"

// After
return "treehouse"
```

- [ ] **Step 3: Update the test expectation in tmux_test.go**

At `tmux_test.go:35`, in the `TestDisplayTitleFallsBackWhenValuesMissing` table row:
```go
// Before
{name: "empty", want: "claude-manager"},

// After
{name: "empty", want: "treehouse"},
```

- [ ] **Step 4: Update README.md**

Change the title:
```markdown
# Treehouse
```

Change the Run section:
```bash
go run ./cmd/treehouse
```

- [ ] **Step 5: Run tests to confirm**

```bash
go test ./internal/launcher/...
```
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/ui/app.go internal/launcher/tmux.go internal/launcher/tmux_test.go README.md
git commit -m "refactor: rebrand user-visible strings from Claude Manager to Treehouse"
```

---

## Task 3: Rename config dir and add migration

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: Write the failing test**

Add this test to `internal/config/config_test.go`:

```go
func TestMigrateLegacyConfigDir(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)

	oldDir := filepath.Join(root, "claude-manager")
	if err := os.MkdirAll(oldDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldDir, "config.json"), []byte(`{"repos":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	newDir := filepath.Join(root, "treehouse")
	if _, err := os.Stat(newDir); err != nil {
		t.Fatalf("new config dir not created: %v", err)
	}
	if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
		t.Fatalf("old config dir still exists after migration")
	}
	if _, err := os.Stat(filepath.Join(newDir, "config.json")); err != nil {
		t.Fatalf("config.json not present in new dir: %v", err)
	}
}
```

- [ ] **Step 2: Run the test to confirm it fails**

```bash
go test ./internal/config/... -run TestMigrateLegacyConfigDir -v
```
Expected: FAIL — the old dir still exists after Load() because migration hasn't been written.

- [ ] **Step 3: Rename appDir and add migrateLegacyConfigDir in config.go**

Change line 13:
```go
// Before
const appDir = "claude-manager"

// After
const appDir = "treehouse"
```

Add this function after the `const` block (before `ConfigPath`):

```go
func migrateLegacyConfigDir() {
	root, err := os.UserConfigDir()
	if err != nil {
		return
	}
	newDir := filepath.Join(root, appDir)
	oldDir := filepath.Join(root, "claude-manager")
	if _, err := os.Stat(newDir); err == nil {
		return
	}
	if _, err := os.Stat(oldDir); err != nil {
		return
	}
	_ = os.Rename(oldDir, newDir)
}
```

At the top of `Load()`, add the call before `ConfigPath()`:

```go
func Load() (Config, error) {
	migrateLegacyConfigDir()
	path, err := ConfigPath()
	// ... rest unchanged
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
go test ./internal/config/... -v
```
Expected: all PASS, including `TestMigrateLegacyConfigDir`, `TestSaveLoad`, `TestNormalizeRepo`.

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): rename appDir to treehouse with one-time migration from claude-manager"
```

---

## Task 4: Create styles.go with the warm/earthy theme

**Files:**
- Create: `internal/ui/styles.go`

- [ ] **Step 1: Create internal/ui/styles.go**

```go
package ui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

var theme = struct {
	title       lipgloss.Style
	subtitle    lipgloss.Style
	status      lipgloss.Style
	errStyle    lipgloss.Style
	border      lipgloss.Style
	modalBorder lipgloss.Style
	muted       lipgloss.Style
	focused     lipgloss.Style
	hintBar     lipgloss.Style
	section     lipgloss.Style
	emptyState  lipgloss.Style
}{
	title:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("173")),
	subtitle:    lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
	status:      lipgloss.NewStyle().Foreground(lipgloss.Color("108")),
	errStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("167")),
	border:      lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).Padding(1),
	modalBorder: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("180")).Padding(1),
	muted:       lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
	focused:     lipgloss.NewStyle().Foreground(lipgloss.Color("179")),
	hintBar:     lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("235")).Padding(0, 1),
	section:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("223")),
	emptyState:  lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).Padding(1),
}

func newListDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.
		Foreground(lipgloss.Color("173")).
		BorderLeftForeground(lipgloss.Color("173"))
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.
		Foreground(lipgloss.Color("173")).
		BorderLeftForeground(lipgloss.Color("173"))
	d.Styles.NormalTitle = d.Styles.NormalTitle.Foreground(lipgloss.Color("230"))
	d.Styles.NormalDesc = d.Styles.NormalDesc.Foreground(lipgloss.Color("244"))
	d.Styles.DimmedTitle = d.Styles.DimmedTitle.Foreground(lipgloss.Color("244"))
	d.Styles.DimmedDesc = d.Styles.DimmedDesc.Foreground(lipgloss.Color("240"))
	return d
}
```

- [ ] **Step 2: Build to confirm the new file compiles**

```bash
go build ./internal/ui/...
```
Expected: clean exit.

- [ ] **Step 3: Commit**

```bash
git add internal/ui/styles.go
git commit -m "feat(ui): add warm/earthy theme and themed list delegate"
```

---

## Task 5: Wire theme into app.go

**Files:**
- Modify: `internal/ui/app.go`

Replace the old style var block and every call site.

- [ ] **Step 1: Remove the old style var block (lines 115-126)**

Delete the entire block:
```go
var (
	titleStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	subtitleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	statusStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	errorStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	borderStyle     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1)
	mutedStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	focusedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	hintBarStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("238")).Padding(0, 1)
	sectionStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("221"))
	emptyStateStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Border(lipgloss.RoundedBorder()).Padding(1)
)
```

- [ ] **Step 2: Update the list delegate in newModel (line 155)**

```go
// Before
delegate := list.NewDefaultDelegate()

// After
delegate := newListDelegate()
```

- [ ] **Step 3: Update View() — header and footer (app.go ~456-480)**

```go
// Before
titleStyle.Render("Treehouse"),    // (already updated in Task 2)
subtitleStyle.Render(m.headerSubtitle()),
// ...
footer := hintBarStyle.Render(m.hintBar())
statusLine := statusStyle.Render(m.message)
if m.errMessage != "" {
    statusLine = errorStyle.Render(m.errMessage)
}

// After
theme.title.Render("Treehouse"),
theme.subtitle.Render(m.headerSubtitle()),
// ...
footer := theme.hintBar.Render(m.hintBar())
statusLine := theme.status.Render(m.message)
if m.errMessage != "" {
    statusLine = theme.errStyle.Render(m.errMessage)
}
```

- [ ] **Step 4: Update renderRepos() (~line 536)**

```go
// Before
return emptyStateStyle.Render(strings.Join([]string{

// After
return theme.emptyState.Render(strings.Join([]string{
```

- [ ] **Step 5: Update renderWorktrees() (~lines 551-568)**

```go
// Before (empty state)
return emptyStateStyle.Render("No repo selected. Press b to return.")

// After
return theme.emptyState.Render("No repo selected. Press b to return.")
```

```go
// Before (meta section header)
sectionStyle.Render("Repo"),

// After
theme.section.Render("Repo"),
```

```go
// Before (empty worktrees branch)
return borderStyle.Render(strings.Join(metaLines, "\n"))

// After
return theme.border.Render(strings.Join(metaLines, "\n"))
```

- [ ] **Step 6: Update renderHelp() (~lines 572-598)**

```go
// Before
sectionStyle.Render("Help"),
// ...
return borderStyle.Render(strings.Join(lines, "\n"))

// After
theme.section.Render("Help"),
// ...
return theme.border.Render(strings.Join(lines, "\n"))
```

- [ ] **Step 7: Update renderModal() — confirmation modals (~lines 603-623)**

The three simple confirmation modals (`modalRemoveRepo`, `modalRemoveWorktree`, `modalCloseSession`) use plain `borderStyle`. Change to `theme.modalBorder`:

```go
// modalRemoveRepo (line ~609)
// Before
return borderStyle.Render(text)
// After
return theme.modalBorder.Render(text)

// modalRemoveWorktree (line ~615)
// Before
return borderStyle.Render(text)
// After
return theme.modalBorder.Render(text)

// modalCloseSession (line ~622)
// Before
return borderStyle.Render(text)
// After
return theme.modalBorder.Render(text)
```

- [ ] **Step 8: Update renderModal() — form modals (lines ~624-666)**

```go
// Section headers inside the default modal case
// Before
lines = append(lines, sectionStyle.Render("Add Repo"))
lines = append(lines, sectionStyle.Render("Edit Default Worktree Path"))
lines = append(lines, sectionStyle.Render("Create Worktree"))
lines = append(lines, mutedStyle.Render("Mode: "+modeText+" (ctrl+m toggles)"))

// After
lines = append(lines, theme.section.Render("Add Repo"))
lines = append(lines, theme.section.Render("Edit Default Worktree Path"))
lines = append(lines, theme.section.Render("Create Worktree"))
lines = append(lines, theme.muted.Render("Mode: "+modeText+" (ctrl+m toggles)"))
```

```go
// Focused input label (~line 642)
// Before
label = focusedStyle.Render(label)
// After
label = theme.focused.Render(label)
```

```go
// Path suggestion list (~lines 646-653)
// Before
lines = append(lines, mutedStyle.Render("Directories"))
for idx, suggestion := range m.pathSuggestions {
    entry := "  " + shortenPath(suggestion.Value, 120)
    if idx == m.pathSuggestionIdx {
        entry = focusedStyle.Render("> " + shortenPath(suggestion.Value, 118))
    }
    lines = append(lines, entry)
}

// After
lines = append(lines, theme.muted.Render("Directories"))
for idx, suggestion := range m.pathSuggestions {
    entry := theme.muted.Render("  " + shortenPath(suggestion.Value, 120))
    if idx == m.pathSuggestionIdx {
        entry = theme.focused.Render("> " + shortenPath(suggestion.Value, 118))
    }
    lines = append(lines, entry)
}
```

```go
// Branch suggestions and "base branch ignored" hint (~lines 655-663)
// Before
lines = append(lines, mutedStyle.Render("Suggestions: "+strings.Join(suggestions, "  ")))
// ...
lines = append(lines, mutedStyle.Render("Base branch is ignored for existing-branch mode."))

// After
lines = append(lines, theme.muted.Render("Suggestions: "+strings.Join(suggestions, "  ")))
// ...
lines = append(lines, theme.muted.Render("Base branch is ignored for existing-branch mode."))
```

```go
// Footer hint inside modal (~line 665)
// Before
lines = append(lines, "", mutedStyle.Render("Enter submits. Tab moves. Right accepts directory. Esc cancels."))
return borderStyle.Render(strings.Join(lines, "\n"))

// After
lines = append(lines, "", theme.muted.Render("Enter submits. Tab moves. Right accepts directory. Esc cancels."))
return theme.modalBorder.Render(strings.Join(lines, "\n"))
```

- [ ] **Step 9: Build to confirm no compilation errors**

```bash
go build ./...
```
Expected: clean exit.

- [ ] **Step 10: Run all tests**

```bash
go test ./...
```
Expected: all PASS.

- [ ] **Step 11: Commit**

```bash
git add internal/ui/app.go
git commit -m "feat(ui): apply warm/earthy theme across all TUI surfaces"
```

---

## Verification

- [ ] **Build and test**

```bash
go build ./... && go test ./...
```
Expected: clean build, all tests pass.

- [ ] **Launch the app**

```bash
go run ./cmd/treehouse
```
Confirm:
- Title bar reads **"Treehouse"** in terracotta (not pink, not "Claude Manager")
- Selecting a list item shows terracotta/orange highlight with terracotta left-border (not stock pink/magenta)
- Footer hint bar is cream text on a deep brown background
- Open Add Repo modal (`a`): focused input prompt is gold, directory suggestions label is muted warm gray, selected suggestion is gold with `> ` prefix
- Open a form modal: section header is cream-bold, mode indicator is muted, border is a slightly warmer/brighter rounded border than the static panels
- Help panel (`?`) and worktree meta block use the muted-border treatment
- Error scenario (submit empty repo path): error line renders in rust, not harsh red
- Status messages (after save/refresh): sage green, not cyan

- [ ] **Verify config migration**

```bash
# Simulate a user who had the old config dir
mv ~/Library/Application\ Support/treehouse ~/Library/Application\ Support/claude-manager 2>/dev/null || true
go run ./cmd/treehouse
# App should launch cleanly with repos intact
# Confirm the dir was renamed back
ls ~/Library/Application\ Support/ | grep -E "treehouse|claude-manager"
```
Expected: only `treehouse` present.

- [ ] **Verify tmux fallback**

```bash
go test ./internal/launcher/... -run TestDisplayTitleFallsBackWhenValuesMissing -v
```
Expected: PASS, fallback string is `"treehouse"`.
