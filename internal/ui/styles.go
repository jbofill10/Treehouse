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
