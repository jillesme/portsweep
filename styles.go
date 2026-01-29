package main

import "github.com/charmbracelet/lipgloss"

// UI styles for the TUI interface
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF6B6B")).
			MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#1a1a1a")).
			Background(lipgloss.Color("#7DCFFF"))

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#c0c0c0"))

	checkedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6B6B"))

	checkboxChecked   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B")).Render("[x]")
	checkboxUnchecked = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render("[ ]")

	portStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7DCFFF")).
			Width(18)

	pidStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9ECE6A")).
			Width(8)

	nameStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#BB9AF7")).
			Width(15)

	userStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0AF68")).
			Width(12)

	commandStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#737373"))

	cmdDetailStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#565656")).
			Italic(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			MarginTop(1)

	confirmStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF6B6B")).
			MarginTop(1)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9ECE6A")).
			MarginTop(1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#626262")).
			MarginBottom(0)

	emptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			Italic(true).
			MarginTop(2)

	selectedCountStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FF6B6B"))

	searchStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7DCFFF"))

	searchFilterStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#9ECE6A"))
)
