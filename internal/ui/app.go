package ui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"treehouse/internal/config"
	"treehouse/internal/git"
	"treehouse/internal/launcher"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type screen int
type modalKind int

const (
	screenRepos screen = iota
	screenWorktrees
)

const (
	modalNone modalKind = iota
	modalAddRepo
	modalEditBasePath
	modalCreateWorktree
	modalRemoveWorktree
	modalRemoveRepo
	modalCloseSession
)

type repoItem struct {
	repo config.RepoConfig
}

func (r repoItem) FilterValue() string { return r.repo.Name + " " + r.repo.RepoPath }
func (r repoItem) Title() string       { return r.repo.Name }
func (r repoItem) Description() string {
	return shortenPath(r.repo.RepoPath, 80) + "  |  wt: " + shortenPath(r.repo.WorktreeBasePath, 48)
}

type worktreeItem struct {
	entry         git.WorktreeEntry
	sessionName   string
	sessionActive bool
}

func (w worktreeItem) FilterValue() string { return w.entry.Branch + " " + w.entry.Path }
func (w worktreeItem) Title() string {
	parts := []string{w.entry.Branch}
	if w.entry.IsMain {
		parts = append(parts, "[main]")
	}
	if w.entry.IsDetached {
		parts = append(parts, "[detached]")
	}
	if w.entry.Dirty {
		parts = append(parts, fmt.Sprintf("A:%d M:%d D:%d", w.entry.Added, w.entry.Modified, w.entry.Deleted))
	} else {
		parts = append(parts, "clean")
	}
	if w.sessionActive {
		parts = append(parts, "tmux:on")
	} else {
		parts = append(parts, "tmux:off")
	}
	return strings.Join(parts, "  ")
}
func (w worktreeItem) Description() string { return shortenPath(w.entry.Path, 110) }

type repoStateMsg struct {
	state git.RepoState
	err   error
}

type actionMsg struct {
	message string
	err     error
}

type launchTmuxMsg struct {
	session string
}

type model struct {
	cfg               config.Config
	repos             list.Model
	worktrees         list.Model
	launchCwd         string
	selectedRepo      *config.RepoConfig
	selectedState     git.RepoState
	screen            screen
	modal             modalKind
	inputs            []textinput.Model
	activeInput       int
	pathSuggestions    []pathSuggestion
	pathSuggestionIdx  int
	branchSuggestionIdx int
	worktreeBaseDirty  bool
	createMode         string
	branchOptions      []string
	helpVisible       bool
	message           string
	errMessage        string
	width             int
	height            int
	quitting          bool
}

var (
	ensureSession = launcher.EnsureSession
	attachCommand = launcher.AttachCommand
	hasSession    = launcher.HasSession
	killSession   = launcher.KillSession
	execProcess   = tea.ExecProcess
)

func Run() error {
	if err := launcher.ValidateRuntime(); err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	launchCwd, err := os.Getwd()
	if err != nil {
		launchCwd = "."
	}
	m := newModel(cfg, launchCwd)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}

func newModel(cfg config.Config, launchCwd string) model {
	delegate := newListDelegate()

	repos := list.New([]list.Item{}, delegate, 0, 0)
	repos.Title = "Saved Repos"
	repos.SetShowStatusBar(false)
	repos.SetFilteringEnabled(false)
	repos.SetShowHelp(false)
	repos.SetShowPagination(false)

	worktrees := list.New([]list.Item{}, delegate, 0, 0)
	worktrees.Title = "Worktrees"
	worktrees.SetShowStatusBar(false)
	worktrees.SetFilteringEnabled(false)
	worktrees.SetShowHelp(false)
	worktrees.SetShowPagination(false)

	m := model{
		cfg:        cfg,
		repos:      repos,
		worktrees:  worktrees,
		launchCwd:  resolveLaunchCwd(launchCwd),
		createMode: "new",
	}
	m.syncRepos()
	return m
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.repos.SetSize(max(20, msg.Width-6), max(8, msg.Height-11))
		m.worktrees.SetSize(max(20, msg.Width-6), max(8, msg.Height-14))
		return m, nil
	case tea.KeyMsg:
		if m.modal != modalNone {
			return m.updateModal(msg)
		}
		return m.updateBase(msg)
	case repoStateMsg:
		if msg.err != nil {
			m.errMessage = msg.err.Error()
			return m, nil
		}
		m.selectedState = msg.state
		m.syncWorktrees()
		m.errMessage = ""
		return m, nil
	case actionMsg:
		if msg.err != nil {
			m.errMessage = msg.err.Error()
			return m, nil
		}
		m.message = msg.message
		m.errMessage = ""
		if m.selectedRepo != nil {
			return m, refreshRepoCmd(m.selectedRepo.RepoPath)
		}
		return m, nil
	case launchTmuxMsg:
		return m, execProcess(attachCommand(msg.session), func(err error) tea.Msg {
			if err != nil {
				return actionMsg{err: err}
			}
			return actionMsg{message: "tmux session ready"}
		})
	}

	var cmd tea.Cmd
	if m.screen == screenRepos {
		m.repos, cmd = m.repos.Update(msg)
	} else {
		m.worktrees, cmd = m.worktrees.Update(msg)
	}
	return m, cmd
}

func (m model) updateBase(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "?":
		m.helpVisible = !m.helpVisible
		return m, nil
	}

	switch m.screen {
	case screenRepos:
		switch msg.String() {
		case "a":
			m.startAddRepo()
			return m, nil
		case "x":
			if _, ok := m.selectedRepoItem(); !ok {
				m.message = "No repo selected."
				return m, nil
			}
			m.modal = modalRemoveRepo
			return m, nil
		case "up":
			if m.repos.Index() == 0 {
				m.repos.Select(len(m.repos.Items()) - 1)
				return m, nil
			}
		case "down":
			if m.repos.Index() == len(m.repos.Items())-1 {
				m.repos.Select(0)
				return m, nil
			}
		case "enter":
			item, ok := m.selectedRepoItem()
			if !ok {
				m.message = "No repos saved. Press a to add one."
				return m, nil
			}
			repo := item.repo
			m.selectedRepo = &repo
			m.screen = screenWorktrees
			m.message = "Loading worktrees..."
			return m, refreshRepoCmd(repo.RepoPath)
		}
	case screenWorktrees:
		switch msg.String() {
		case "b":
			m.screen = screenRepos
			m.message = "Back to saved repos."
			return m, nil
		case "r":
			if m.selectedRepo == nil {
				m.message = "Select a repo first."
				return m, nil
			}
			m.message = "Refreshing worktrees..."
			return m, refreshRepoCmd(m.selectedRepo.RepoPath)
		case "o", "enter":
			return m, m.openSelectedWorktreeCmd(launcher.ModeNormal)
		case "D":
			return m, m.openSelectedWorktreeCmd(launcher.ModeDangerous)
		case "p":
			m.startEditBasePath()
			return m, nil
		case "c":
			m.startCreateWorktree()
			return m, nil
		case "x":
			if _, ok := m.selectedWorktreeItem(); !ok {
				m.message = "No worktree selected."
				return m, nil
			}
			m.modal = modalRemoveWorktree
			return m, nil
		case "k":
			item, ok := m.selectedWorktreeItem()
			if !ok {
				m.message = "No worktree selected."
				return m, nil
			}
			if !item.sessionActive {
				m.message = "No tmux session running for this worktree."
				return m, nil
			}
			m.modal = modalCloseSession
			return m, nil
		case "up":
			if m.worktrees.Index() == 0 {
				m.worktrees.Select(len(m.worktrees.Items()) - 1)
				return m, nil
			}
		case "down":
			if m.worktrees.Index() == len(m.worktrees.Items())-1 {
				m.worktrees.Select(0)
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	if m.screen == screenRepos {
		m.repos, cmd = m.repos.Update(msg)
	} else {
		m.worktrees, cmd = m.worktrees.Update(msg)
	}
	return m, cmd
}

func (m model) updateModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.modal {
	case modalRemoveRepo:
		switch msg.String() {
		case "esc", "n":
			m.modal = modalNone
			m.message = "Remove cancelled."
			return m, nil
		case "y":
			cmd := m.removeSelectedRepoCmd()
			m.modal = modalNone
			return m, cmd
		}
		return m, nil
	case modalRemoveWorktree:
		switch msg.String() {
		case "esc", "n":
			m.modal = modalNone
			m.message = "Remove cancelled."
			return m, nil
		case "y":
			cmd := m.removeSelectedWorktreeCmd(false)
			m.modal = modalNone
			return m, cmd
		case "f":
			cmd := m.removeSelectedWorktreeCmd(true)
			m.modal = modalNone
			return m, cmd
		}
		return m, nil
	case modalCloseSession:
		switch msg.String() {
		case "esc", "n":
			m.modal = modalNone
			m.message = "Close session cancelled."
			return m, nil
		case "y":
			cmd := m.closeSelectedSessionCmd()
			m.modal = modalNone
			return m, cmd
		}
		return m, nil
	default:
		switch msg.String() {
		case "esc":
			m.modal = modalNone
			m.inputs = nil
			m.branchOptions = nil
			m.pathSuggestions = nil
			m.pathSuggestionIdx = 0
			m.worktreeBaseDirty = false
			m.message = "Action cancelled."
			return m, nil
		case "tab", "shift+tab":
			m.cycleInputs(msg.String())
			return m, nil
		case "up":
			if m.modal == modalCreateWorktree && m.createMode == "existing" {
				m.navigateBranchSuggestions(-1)
				return m, nil
			}
			if m.navigatePathSuggestions(-1) {
				return m, nil
			}
			m.cycleInputs(msg.String())
			return m, nil
		case "down":
			if m.modal == modalCreateWorktree && m.createMode == "existing" {
				m.navigateBranchSuggestions(1)
				return m, nil
			}
			if m.navigatePathSuggestions(1) {
				return m, nil
			}
			m.cycleInputs(msg.String())
			return m, nil
		case "right":
			if m.modal == modalCreateWorktree && m.createMode == "existing" {
				m.acceptBranchSuggestion()
				return m, nil
			}
			if m.acceptPathSuggestion() {
				return m, nil
			}
		case "ctrl+t":
			if m.modal == modalCreateWorktree {
				if m.createMode == "new" {
					m.createMode = "existing"
					m.inputs = []textinput.Model{newInput("Branch: ", "")}
					m.activeInput = 0
					m.branchSuggestionIdx = 0
				} else {
					m.createMode = "new"
					defaultBase := defaultBaseBranch(m.branchOptions)
					m.inputs = []textinput.Model{
						newInput("Branch name: ", ""),
						newInput("Base branch: ", defaultBase),
					}
					m.activeInput = 0
				}
				m.applyInputFocus()
				return m, nil
			}
		case "enter":
			return m, m.submitModal()
		}
	}

	m.applyInputFocus()
	var cmd tea.Cmd
	if m.modal == modalAddRepo && isTypingKey(msg) {
		if m.activeInput == 1 {
			m.worktreeBaseDirty = true
		}
	}
	m.inputs[m.activeInput], cmd = m.inputs[m.activeInput].Update(msg)
	if m.modal == modalAddRepo {
		m.applyAddRepoDefaults()
	}
	if m.modal == modalCreateWorktree && m.createMode == "existing" && isTypingKey(msg) {
		m.branchSuggestionIdx = 0
	}
	m.refreshPathSuggestions()
	return m, cmd
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	header := lipgloss.JoinVertical(
		lipgloss.Left,
		theme.title.Render("Treehouse"),
		theme.subtitle.Render(m.headerSubtitle()),
	)

	body := m.renderRepos()
	if m.screen == screenWorktrees {
		body = m.renderWorktrees()
	}

	footer := theme.hintBar.Render(m.hintBar())
	statusLine := theme.status.Render(m.message)
	if m.errMessage != "" {
		statusLine = theme.errStyle.Render(m.errMessage)
	}

	view := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		body,
		"",
		footer,
		statusLine,
	)

	if m.helpVisible {
		view = lipgloss.JoinVertical(lipgloss.Left, view, "", m.renderHelp())
	}
	if m.modal != modalNone {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.renderModal())
	}
	return view
}

func (m model) headerSubtitle() string {
	switch m.screen {
	case screenRepos:
		return "Manage saved repos and jump into their worktrees."
	case screenWorktrees:
		if m.selectedRepo == nil {
			return "No repo selected."
		}
		return fmt.Sprintf("%s worktrees", m.selectedRepo.Name)
	default:
		return ""
	}
}

func (m *model) syncRepos() {
	items := make([]list.Item, 0, len(m.cfg.Repos))
	for _, repo := range m.cfg.Repos {
		items = append(items, repoItem{repo: repo})
	}
	m.repos.SetItems(items)
}

func (m *model) syncWorktrees() {
	items := make([]list.Item, 0, len(m.selectedState.Worktrees))
	for _, entry := range m.selectedState.Worktrees {
		sessionName := ""
		sessionActive := false
		if m.selectedRepo != nil {
			sessionName = launcher.SessionName(m.selectedRepo.Name, entry.Branch)
			sessionActive = hasSession(sessionName)
		}
		items = append(items, worktreeItem{
			entry:         entry,
			sessionName:   sessionName,
			sessionActive: sessionActive,
		})
	}
	if m.selectedRepo != nil {
		m.worktrees.Title = fmt.Sprintf("Worktrees: %s", m.selectedRepo.Name)
	}
	m.worktrees.SetItems(items)
}

func (m model) renderRepos() string {
	if len(m.cfg.Repos) == 0 {
		return theme.emptyState.Render(strings.Join([]string{
			"No saved repos yet.",
			"",
			"Press a to add a repo.",
			"Pick a repo path from the current directory or type to narrow it.",
			"A default worktree base path will be suggested automatically.",
			"Press ? for the full key reference.",
		}, "\n"))
	}
	return m.repos.View()
}

func (m model) renderWorktrees() string {
	if m.selectedRepo == nil {
		return theme.emptyState.Render("No repo selected. Press b to return.")
	}

	metaLines := []string{
		theme.section.Render("Repo"),
		fmt.Sprintf("name: %s", m.selectedRepo.Name),
		fmt.Sprintf("root: %s", shortenPath(m.selectedRepo.RepoPath, 120)),
		fmt.Sprintf("default worktree path: %s", shortenPath(m.selectedRepo.WorktreeBasePath, 120)),
	}
	if len(m.selectedState.Worktrees) == 0 {
		metaLines = append(metaLines, "", "No worktrees found for this repo yet.")
		return theme.border.Render(strings.Join(metaLines, "\n"))
	}
	return lipgloss.JoinVertical(
		lipgloss.Left,
		strings.Join(metaLines, "\n"),
		"",
		m.worktrees.View(),
	)
}

func (m model) renderHelp() string {
	lines := []string{
		theme.section.Render("Help"),
		"",
		"Global: q quit | ? toggle help",
	}
	if m.screen == screenRepos {
		lines = append(lines,
			"Repos: ↑/↓ move | enter open repo | a add repo | x remove repo",
			"Add repo: tab move fields | ↑/↓ browse dirs | → accept dir | enter submit | esc cancel",
			"When empty: press a to create the first saved repo entry.",
		)
	} else {
		lines = append(lines,
			"Worktrees: ↑/↓ move | enter or o open session | D open dangerous | k close session | c create | x remove | p edit base path | r refresh | b back",
			"Remove confirmation: y remove | f force remove | esc cancel",
		)
		if m.modal == modalCloseSession {
			lines = append(lines, "Close session confirmation: y close | esc cancel")
		}
	}
	if m.modal == modalCreateWorktree {
		lines = append(lines,
			"Create form: tab move fields | ctrl+t toggle new/existing mode | enter submit | esc cancel",
		)
	}
	return theme.border.Render(strings.Join(lines, "\n"))
}

func (m model) renderModal() string {
	switch m.modal {
	case modalRemoveRepo:
		item, ok := m.selectedRepoItem()
		if !ok {
			return ""
		}
		text := fmt.Sprintf("Remove repo?\n\n%s\n%s\n\n[y] remove  [n/esc] cancel", item.repo.Name, item.repo.RepoPath)
		return theme.modalBorder.Render(text)
	case modalRemoveWorktree:
		item, ok := m.selectedWorktreeItem()
		if !ok {
			return ""
		}
		text := fmt.Sprintf("Remove worktree?\n\n%s\n\n[y] remove  [f] force remove  [n/esc] cancel", item.entry.Path)
		return theme.modalBorder.Render(text)
	case modalCloseSession:
		item, ok := m.selectedWorktreeItem()
		if !ok {
			return ""
		}
		text := fmt.Sprintf("Close tmux session?\n\n%s\n\nsession: %s\n\n[y] close  [n/esc] cancel", item.entry.Path, item.sessionName)
		return theme.modalBorder.Render(text)
	default:
		lines := []string{}
		switch m.modal {
		case modalAddRepo:
			lines = append(lines, theme.section.Render("Add Repo"))
		case modalEditBasePath:
			lines = append(lines, theme.section.Render("Edit Default Worktree Path"))
		case modalCreateWorktree:
			lines = append(lines, theme.section.Render("Create Worktree"))
			modeText := "new branch from base"
			if m.createMode == "existing" {
				modeText = "existing branch"
			}
			lines = append(lines, theme.muted.Render("Mode: "+modeText+" (ctrl+t toggles)"))
		}
		if m.modal == modalCreateWorktree && m.createMode == "existing" {
			label := theme.focused.Render(m.inputs[0].Prompt)
			lines = append(lines, label+m.inputs[0].View())
			branches := m.filteredBranchSuggestions()
			if len(branches) > 0 {
				const maxVisible = 8
				start := max(0, m.branchSuggestionIdx-maxVisible+1)
				if start > m.branchSuggestionIdx {
					start = m.branchSuggestionIdx
				}
				end := min(start+maxVisible, len(branches))
				if end-start < maxVisible && start > 0 {
					start = max(0, end-maxVisible)
				}
				header := fmt.Sprintf("Branches (%d)", len(branches))
				lines = append(lines, theme.muted.Render(header))
				for idx := start; idx < end; idx++ {
					entry := theme.muted.Render("  " + branches[idx])
					if idx == m.branchSuggestionIdx {
						entry = theme.focused.Render("> " + branches[idx])
					}
					lines = append(lines, entry)
				}
			} else {
				lines = append(lines, theme.muted.Render("  (no matching branches)"))
			}
			lines = append(lines, "", theme.muted.Render("up/down select  |  enter confirm  |  ctrl+t switch mode  |  esc cancel"))
		} else {
			for i := range m.inputs {
				label := m.inputs[i].Prompt
				if i == m.activeInput {
					label = theme.focused.Render(label)
				}
				lines = append(lines, label+m.inputs[i].View())
				if m.shouldRenderPathSuggestions(i) {
					lines = append(lines, theme.muted.Render("Directories"))
					for idx, suggestion := range m.pathSuggestions {
						entry := theme.muted.Render("  " + shortenPath(suggestion.Value, 120))
						if idx == m.pathSuggestionIdx {
							entry = theme.focused.Render("> " + shortenPath(suggestion.Value, 118))
						}
						lines = append(lines, entry)
					}
				}
				if m.modal == modalCreateWorktree && shouldShowSuggestions(m.createMode, i) {
					suggestions := m.branchSuggestions(i)
					if len(suggestions) > 0 {
						lines = append(lines, theme.muted.Render("Suggestions: "+strings.Join(suggestions, "  ")))
					}
				}
			}
			lines = append(lines, "", theme.muted.Render("Enter submits. Tab moves. Esc cancels."))
		}
		return theme.modalBorder.Render(strings.Join(lines, "\n"))
	}
}

func (m *model) startAddRepo() {
	m.modal = modalAddRepo
	m.inputs = []textinput.Model{
		newInput("Repo path: ", m.launchCwd),
		newInput("Worktree base path: ", ""),
	}
	m.activeInput = 0
	m.worktreeBaseDirty = false
	m.applyAddRepoDefaults()
	m.refreshPathSuggestions()
	m.applyInputFocus()
}

func (m *model) startEditBasePath() {
	if m.selectedRepo == nil {
		m.message = "Select a repo first."
		return
	}
	m.modal = modalEditBasePath
	m.inputs = []textinput.Model{
		newInput("Worktree base path: ", m.selectedRepo.WorktreeBasePath),
	}
	m.activeInput = 0
	m.refreshPathSuggestions()
	m.applyInputFocus()
}

func (m *model) startCreateWorktree() {
	if m.selectedRepo == nil {
		m.message = "Select a repo first."
		return
	}
	branches, err := git.ListBranches(m.selectedRepo.RepoPath)
	if err != nil {
		m.errMessage = err.Error()
	}
	defaultBase := defaultBaseBranch(branches)
	m.modal = modalCreateWorktree
	m.inputs = []textinput.Model{
		newInput("Branch name: ", ""),
		newInput("Base branch: ", defaultBase),
	}
	m.activeInput = 0
	m.createMode = "new"
	m.branchOptions = branches
	m.applyInputFocus()
}

func (m *model) applyInputFocus() {
	for i := range m.inputs {
		if i == m.activeInput {
			m.inputs[i].Focus()
		} else {
			m.inputs[i].Blur()
		}
	}
}

func (m *model) cycleInputs(key string) {
	if len(m.inputs) == 0 {
		return
	}
	if key == "up" || key == "shift+tab" {
		m.activeInput--
	} else {
		m.activeInput++
	}
	if m.activeInput >= len(m.inputs) {
		m.activeInput = 0
	}
	if m.activeInput < 0 {
		m.activeInput = len(m.inputs) - 1
	}
	m.refreshPathSuggestions()
	m.applyInputFocus()
}

func (m *model) submitModal() tea.Cmd {
	switch m.modal {
	case modalAddRepo:
		return m.submitAddRepo()
	case modalEditBasePath:
		return m.submitEditBasePath()
	case modalCreateWorktree:
		return m.submitCreateWorktree()
	default:
		return nil
	}
}

func (m *model) submitAddRepo() tea.Cmd {
	repo, err := config.NormalizeRepo(config.RepoConfig{
		RepoPath:         m.inputs[0].Value(),
		WorktreeBasePath: m.inputs[1].Value(),
	})
	if err != nil {
		m.errMessage = err.Error()
		return nil
	}
	for _, existing := range m.cfg.Repos {
		if existing.RepoPath == repo.RepoPath {
			m.errMessage = "repo already saved"
			return nil
		}
	}
	if _, err := git.InspectRepo(repo.RepoPath); err != nil {
		m.errMessage = err.Error()
		return nil
	}
	m.cfg.Repos = append(m.cfg.Repos, repo)
	if err := config.Save(m.cfg); err != nil {
		m.errMessage = err.Error()
		return nil
	}
	m.syncRepos()
	m.modal = modalNone
	m.inputs = nil
	m.pathSuggestions = nil
	m.pathSuggestionIdx = 0
	m.worktreeBaseDirty = false
	m.message = "Repo saved."
	m.errMessage = ""
	return nil
}

func (m *model) submitEditBasePath() tea.Cmd {
	if m.selectedRepo == nil {
		m.message = "Select a repo first."
		return nil
	}
	basePath, err := filepath.Abs(strings.TrimSpace(m.inputs[0].Value()))
	if err != nil {
		m.errMessage = err.Error()
		return nil
	}
	for i := range m.cfg.Repos {
		if m.cfg.Repos[i].ID == m.selectedRepo.ID {
			m.cfg.Repos[i].WorktreeBasePath = basePath
			updated := m.cfg.Repos[i]
			m.selectedRepo = &updated
			break
		}
	}
	if err := config.Save(m.cfg); err != nil {
		m.errMessage = err.Error()
		return nil
	}
	m.syncRepos()
	m.modal = modalNone
	m.inputs = nil
	m.pathSuggestions = nil
	m.pathSuggestionIdx = 0
	m.message = "Default worktree path updated."
	m.errMessage = ""
	return nil
}

func (m *model) submitCreateWorktree() tea.Cmd {
	if m.selectedRepo == nil {
		m.message = "Select a repo first."
		return nil
	}
	branch := strings.TrimSpace(m.inputs[0].Value())
	if branch == "" && m.createMode == "existing" {
		// accept highlighted suggestion if input is empty
		sug := m.filteredBranchSuggestions()
		if m.branchSuggestionIdx < len(sug) {
			branch = sug[m.branchSuggestionIdx]
		}
	}
	if branch == "" {
		m.errMessage = "branch name is required"
		return nil
	}
	var base, target string
	if m.createMode == "existing" {
		target = launcher.SuggestedTargetPath(m.selectedRepo.WorktreeBasePath, branch)
	} else {
		base = strings.TrimSpace(m.inputs[1].Value())
		target = launcher.SuggestedTargetPath(m.selectedRepo.WorktreeBasePath, branch)
	}
	target, err := filepath.Abs(target)
	if err != nil {
		m.errMessage = err.Error()
		return nil
	}
	req := git.CreateWorktreeRequest{
		RepoPath:   m.selectedRepo.RepoPath,
		Mode:       m.createMode,
		BaseBranch: base,
		BranchName: branch,
		TargetPath: target,
	}
	m.modal = modalNone
	m.inputs = nil
	m.branchOptions = nil
	m.pathSuggestions = nil
	m.pathSuggestionIdx = 0
	m.message = "Creating worktree..."
	return func() tea.Msg {
		err := git.CreateWorktree(req)
		if err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{message: "Worktree created."}
	}
}

func (m model) openSelectedWorktreeCmd(mode launcher.LaunchMode) tea.Cmd {
	if m.selectedRepo == nil {
		return func() tea.Msg {
			return actionMsg{message: "Select a repo first."}
		}
	}
	item, ok := m.selectedWorktreeItem()
	if !ok {
		return func() tea.Msg {
			return actionMsg{message: "No worktree selected."}
		}
	}
	session := launcher.SessionName(m.selectedRepo.Name, item.entry.Branch)
	size := launcher.TerminalSize{Width: m.width, Height: m.height}
	return func() tea.Msg {
		err := ensureSession(m.selectedRepo.Name, item.entry.Branch, item.entry.Path, size, mode)
		if err != nil {
			return actionMsg{err: err}
		}
		return launchTmuxMsg{session: session}
	}
}

func (m model) removeSelectedWorktreeCmd(force bool) tea.Cmd {
	if m.selectedRepo == nil {
		return func() tea.Msg {
			return actionMsg{message: "Select a repo first."}
		}
	}
	item, ok := m.selectedWorktreeItem()
	if !ok {
		return func() tea.Msg {
			return actionMsg{message: "No worktree selected."}
		}
	}
	if item.entry.IsMain {
		return func() tea.Msg {
			return actionMsg{err: errors.New("cannot remove the main repo worktree")}
		}
	}
	return func() tea.Msg {
		err := git.RemoveWorktree(m.selectedRepo.RepoPath, item.entry.Path, force)
		if err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{message: "Worktree removed."}
	}
}

func (m *model) removeSelectedRepoCmd() tea.Cmd {
	item, ok := m.selectedRepoItem()
	if !ok {
		return func() tea.Msg { return actionMsg{message: "No repo selected."} }
	}
	newRepos := make([]config.RepoConfig, 0, len(m.cfg.Repos))
	for _, r := range m.cfg.Repos {
		if r.ID != item.repo.ID {
			newRepos = append(newRepos, r)
		}
	}
	m.cfg.Repos = newRepos
	if err := config.Save(m.cfg); err != nil {
		return func() tea.Msg { return actionMsg{err: err} }
	}
	if m.selectedRepo != nil && m.selectedRepo.ID == item.repo.ID {
		m.selectedRepo = nil
	}
	m.syncRepos()
	m.message = "Repo removed."
	m.errMessage = ""
	return nil
}

func (m model) closeSelectedSessionCmd() tea.Cmd {
	item, ok := m.selectedWorktreeItem()
	if !ok {
		return func() tea.Msg {
			return actionMsg{message: "No worktree selected."}
		}
	}
	if !item.sessionActive {
		return func() tea.Msg {
			return actionMsg{message: "No tmux session running for this worktree."}
		}
	}
	return func() tea.Msg {
		if err := killSession(item.sessionName); err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{message: "tmux session closed."}
	}
}

func refreshRepoCmd(repoPath string) tea.Cmd {
	return func() tea.Msg {
		state, err := git.InspectRepo(repoPath)
		return repoStateMsg{state: state, err: err}
	}
}

func newInput(prompt string, value string) textinput.Model {
	in := textinput.New()
	in.Prompt = prompt
	in.SetValue(value)
	in.CharLimit = 512
	in.Width = 60
	return in
}

func (m *model) shouldRenderPathSuggestions(inputIdx int) bool {
	return inputIdx == m.activeInput && m.activeInputIsPath() && len(m.pathSuggestions) > 0
}

func (m *model) activeInputIsPath() bool {
	switch m.modal {
	case modalAddRepo:
		return m.activeInput == 0 || m.activeInput == 1
	case modalEditBasePath:
		return m.activeInput == 0
	default:
		return false
	}
}

func (m *model) refreshPathSuggestions() {
	if !m.activeInputIsPath() || len(m.inputs) == 0 {
		m.pathSuggestions = nil
		m.pathSuggestionIdx = 0
		return
	}

	suggestions, err := listPathSuggestions(m.inputs[m.activeInput].Value(), m.launchCwd)
	if err != nil {
		m.pathSuggestions = nil
		m.pathSuggestionIdx = 0
		return
	}
	m.pathSuggestions = suggestions
	if len(m.pathSuggestions) == 0 {
		m.pathSuggestionIdx = 0
		return
	}
	if m.pathSuggestionIdx >= len(m.pathSuggestions) {
		m.pathSuggestionIdx = len(m.pathSuggestions) - 1
	}
	if m.pathSuggestionIdx < 0 {
		m.pathSuggestionIdx = 0
	}
}

func (m *model) navigatePathSuggestions(delta int) bool {
	if !m.activeInputIsPath() || len(m.pathSuggestions) == 0 {
		return false
	}
	m.pathSuggestionIdx += delta
	if m.pathSuggestionIdx < 0 {
		m.pathSuggestionIdx = len(m.pathSuggestions) - 1
	}
	if m.pathSuggestionIdx >= len(m.pathSuggestions) {
		m.pathSuggestionIdx = 0
	}
	return true
}

func (m *model) acceptPathSuggestion() bool {
	if !m.activeInputIsPath() || len(m.pathSuggestions) == 0 {
		return false
	}
	m.inputs[m.activeInput].SetValue(m.pathSuggestions[m.pathSuggestionIdx].Value)
	if m.modal == modalAddRepo && m.activeInput == 1 {
		m.worktreeBaseDirty = true
	}
	if m.modal == modalAddRepo {
		m.applyAddRepoDefaults()
	}
	m.refreshPathSuggestions()
	return true
}

func (m *model) filteredBranchSuggestions() []string {
	if len(m.branchOptions) == 0 {
		return nil
	}
	query := ""
	if len(m.inputs) > 0 {
		query = strings.ToLower(strings.TrimSpace(m.inputs[0].Value()))
	}
	var matches []string
	for _, b := range m.branchOptions {
		if query == "" || strings.Contains(strings.ToLower(b), query) {
			matches = append(matches, b)
		}
	}
	return matches
}

func (m *model) navigateBranchSuggestions(delta int) {
	branches := m.filteredBranchSuggestions()
	if len(branches) == 0 {
		return
	}
	m.branchSuggestionIdx += delta
	if m.branchSuggestionIdx < 0 {
		m.branchSuggestionIdx = len(branches) - 1
	}
	if m.branchSuggestionIdx >= len(branches) {
		m.branchSuggestionIdx = 0
	}
}

func (m *model) acceptBranchSuggestion() {
	branches := m.filteredBranchSuggestions()
	if m.branchSuggestionIdx < len(branches) {
		m.inputs[0].SetValue(branches[m.branchSuggestionIdx])
	}
}

func (m *model) applyAddRepoDefaults() {
	if m.modal != modalAddRepo || len(m.inputs) < 2 {
		return
	}

	repoPath := strings.TrimSpace(m.inputs[0].Value())
	if repoPath == "" {
		if !m.worktreeBaseDirty {
			m.inputs[1].SetValue("")
		}
		return
	}

	if !m.worktreeBaseDirty {
		m.inputs[1].SetValue(defaultWorktreeBasePath(repoPath))
	}
}

func isTypingKey(msg tea.KeyMsg) bool {
	switch msg.Type {
	case tea.KeyRunes, tea.KeySpace, tea.KeyBackspace, tea.KeyDelete:
		return true
	default:
		return false
	}
}

func (m model) selectedRepoItem() (repoItem, bool) {
	item, ok := m.repos.SelectedItem().(repoItem)
	return item, ok
}

func (m model) selectedWorktreeItem() (worktreeItem, bool) {
	item, ok := m.worktrees.SelectedItem().(worktreeItem)
	return item, ok
}

func (m model) hintBar() string {
	if m.screen == screenRepos {
		return "a add repo  |  x remove repo  |  enter open repo  |  ? help  |  q quit"
	}
	return "enter/o open  |  D dangerous open  |  k close session  |  c create  |  x remove  |  p edit path  |  r refresh  |  b back  |  ? help  |  q quit"
}

func (m model) branchSuggestions(inputIdx int) []string {
	if len(m.branchOptions) == 0 || inputIdx >= len(m.inputs) {
		return nil
	}
	query := strings.ToLower(strings.TrimSpace(m.inputs[inputIdx].Value()))
	matches := make([]string, 0, 5)
	for _, option := range m.branchOptions {
		if query == "" || strings.Contains(strings.ToLower(option), query) {
			matches = append(matches, option)
		}
		if len(matches) == 5 {
			break
		}
	}
	return matches
}

func shouldShowSuggestions(mode string, inputIdx int) bool {
	if inputIdx == 0 {
		return true
	}
	return mode == "new" && inputIdx == 1
}

func defaultBaseBranch(branches []string) string {
	for _, candidate := range []string{"main", "master"} {
		for _, branch := range branches {
			if branch == candidate {
				return candidate
			}
		}
	}
	if len(branches) > 0 {
		return branches[0]
	}
	return "main"
}

func shortenPath(path string, maxLen int) string {
	if maxLen <= 0 || len(path) <= maxLen {
		return path
	}
	if maxLen <= 3 {
		return path[:maxLen]
	}
	return "..." + path[len(path)-(maxLen-3):]
}
