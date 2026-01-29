package main

import "github.com/charmbracelet/bubbles/key"

// keyMap defines all keyboard bindings for the TUI
type keyMap struct {
	Up        key.Binding
	Down      key.Binding
	Kill      key.Binding
	Refresh   key.Binding
	Toggle    key.Binding
	Quit      key.Binding
	Confirm   key.Binding
	Cancel    key.Binding
	Select    key.Binding
	SelectAll key.Binding
	Search    key.Binding
}

// keys is the default set of key bindings
var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Kill: key.NewBinding(
		key.WithKeys("enter", "d"),
		key.WithHelp("enter/d", "kill"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	Toggle: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "toggle system ports"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Confirm: key.NewBinding(
		key.WithKeys("y"),
		key.WithHelp("y", "confirm"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("n", "esc"),
		key.WithHelp("n/esc", "cancel"),
	),
	Select: key.NewBinding(
		key.WithKeys(" ", "tab"),
		key.WithHelp("space/tab", "select"),
	),
	SelectAll: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "select all"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
}
