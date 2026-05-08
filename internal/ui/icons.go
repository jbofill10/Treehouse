package ui

import "github.com/charmbracelet/lipgloss"

// Nerd Font v3 glyph constants.
const (
	iconBranch   = "" // nf-dev-git_branch
	iconRepo     = "" // nf-fa-folder_open
	iconMain     = "" // nf-fa-home
	iconDetached = "" // nf-fa-chain-broken
	iconClean    = "" // nf-fa-check-circle
	iconDirty    = "" // nf-fa-circle (dirty summary)
	iconAdded    = "" // nf-fa-plus-circle
	iconModified = "" // nf-fa-pencil
	iconDeleted  = "" // nf-fa-minus-circle
	iconTmux     = "" // nf-fa-terminal
	iconPlus     = "" // nf-fa-plus
	iconTrash    = "" // nf-fa-trash
	iconEdit     = "" // nf-fa-pencil-square-o
	iconRefresh  = "" // nf-fa-refresh
	iconBack     = "" // nf-fa-arrow-left
	iconHelp     = "" // nf-fa-question-circle
	iconError    = "" // nf-fa-times-circle
	iconSuccess  = "" // nf-fa-check
)

var (
	iconStyleClean    = lipgloss.NewStyle().Foreground(lipgloss.Color("108"))
	iconStyleAdded    = lipgloss.NewStyle().Foreground(lipgloss.Color("108"))
	iconStyleModified = lipgloss.NewStyle().Foreground(lipgloss.Color("179"))
	iconStyleDeleted  = lipgloss.NewStyle().Foreground(lipgloss.Color("167"))
	iconStyleBranch   = lipgloss.NewStyle().Foreground(lipgloss.Color("173"))
	iconStyleMain     = lipgloss.NewStyle().Foreground(lipgloss.Color("141"))
	iconStyleDetached = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	iconStyleTmuxOn   = lipgloss.NewStyle().Foreground(lipgloss.Color("108"))
	iconStyleTmuxOff  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	iconStyleError    = lipgloss.NewStyle().Foreground(lipgloss.Color("167"))
	iconStyleSuccess  = lipgloss.NewStyle().Foreground(lipgloss.Color("108"))
	iconStyleMuted    = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
)
