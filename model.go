package main

import (
	"fmt"
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles
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
			Width(7)

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
)

// Key bindings
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
}

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
}

// Messages
type tickMsg time.Time
type refreshMsg []Process
type killResultMsg struct {
	success   bool
	pid       int
	port      int
	remaining int // how many left to kill
}

// Model represents the TUI state
type Model struct {
	processes       []Process
	cursor          int
	selected        map[int]bool // PID -> selected
	showSystemPorts bool
	confirming      bool
	toKill          []Process // processes to kill in batch
	killIndex       int       // current index in batch kill
	statusMessage   string
	statusTime      time.Time
	width           int
	height          int
}

// NewModel creates a new Model
func NewModel() Model {
	return Model{
		processes:       []Process{},
		cursor:          0,
		selected:        make(map[int]bool),
		showSystemPorts: false,
		confirming:      false,
		toKill:          []Process{},
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.refreshPorts(),
		m.tickCmd(),
	)
}

// tickCmd returns a command that sends a tick every 2 seconds
func (m Model) tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// refreshPorts fetches the current listening ports
func (m Model) refreshPorts() tea.Cmd {
	return func() tea.Msg {
		processes, err := GetListeningPorts()
		if err != nil {
			return refreshMsg{}
		}
		return refreshMsg(processes)
	}
}

// killProcess kills the specified process
func (m Model) killProcess(pid int, port int, remaining int) tea.Cmd {
	return func() tea.Msg {
		err := KillProcess(pid)
		return killResultMsg{
			success:   err == nil,
			pid:       pid,
			port:      port,
			remaining: remaining,
		}
	}
}

// filteredProcesses returns processes filtered by system port setting
func (m Model) filteredProcesses() []Process {
	if m.showSystemPorts {
		return m.processes
	}

	filtered := make([]Process, 0)
	for _, p := range m.processes {
		if p.Port >= 1024 {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// selectedCount returns the number of selected processes
func (m Model) selectedCount() int {
	count := 0
	filtered := m.filteredProcesses()
	for _, p := range filtered {
		if m.selected[p.PID] {
			count++
		}
	}
	return count
}

// getSelectedProcesses returns all selected processes
func (m Model) getSelectedProcesses() []Process {
	var result []Process
	filtered := m.filteredProcesses()
	for _, p := range filtered {
		if m.selected[p.PID] {
			result = append(result, p)
		}
	}
	return result
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle confirmation mode
		if m.confirming {
			switch {
			case key.Matches(msg, keys.Confirm):
				m.confirming = false
				if len(m.toKill) > 0 {
					// Start batch kill
					m.killIndex = 0
					p := m.toKill[0]
					return m, m.killProcess(p.PID, p.Port, len(m.toKill)-1)
				}
				return m, nil
			case key.Matches(msg, keys.Cancel):
				m.confirming = false
				m.toKill = nil
				m.statusMessage = "Cancelled"
				m.statusTime = time.Now()
				return m, nil
			}
			return m, nil
		}

		// Normal mode key handling
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}

		case key.Matches(msg, keys.Down):
			filtered := m.filteredProcesses()
			if m.cursor < len(filtered)-1 {
				m.cursor++
			}

		case key.Matches(msg, keys.Select):
			filtered := m.filteredProcesses()
			if len(filtered) > 0 && m.cursor < len(filtered) {
				p := filtered[m.cursor]
				m.selected[p.PID] = !m.selected[p.PID]
				// Move cursor down after selecting
				if m.cursor < len(filtered)-1 {
					m.cursor++
				}
			}

		case key.Matches(msg, keys.SelectAll):
			filtered := m.filteredProcesses()
			// Check if all are selected
			allSelected := true
			for _, p := range filtered {
				if !m.selected[p.PID] {
					allSelected = false
					break
				}
			}
			// Toggle all
			for _, p := range filtered {
				m.selected[p.PID] = !allSelected
			}

		case key.Matches(msg, keys.Kill):
			filtered := m.filteredProcesses()
			if len(filtered) == 0 {
				return m, nil
			}

			// If we have selected items, kill those; otherwise kill current
			selected := m.getSelectedProcesses()
			if len(selected) > 0 {
				m.toKill = selected
				m.confirming = true
			} else if m.cursor < len(filtered) {
				m.toKill = []Process{filtered[m.cursor]}
				m.confirming = true
			}

		case key.Matches(msg, keys.Refresh):
			m.statusMessage = "Refreshing..."
			m.statusTime = time.Now()
			return m, m.refreshPorts()

		case key.Matches(msg, keys.Toggle):
			m.showSystemPorts = !m.showSystemPorts
			// Adjust cursor if needed
			filtered := m.filteredProcesses()
			if m.cursor >= len(filtered) {
				m.cursor = max(0, len(filtered)-1)
			}
			if m.showSystemPorts {
				m.statusMessage = "Showing all ports"
			} else {
				m.statusMessage = "Showing user ports only (>=1024)"
			}
			m.statusTime = time.Now()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		// Don't refresh while confirming
		if m.confirming {
			return m, m.tickCmd()
		}
		return m, tea.Batch(m.refreshPorts(), m.tickCmd())

	case refreshMsg:
		m.processes = []Process(msg)
		// Sort by port number
		sort.Slice(m.processes, func(i, j int) bool {
			return m.processes[i].Port < m.processes[j].Port
		})
		// Clean up selected map - remove PIDs that no longer exist
		existingPIDs := make(map[int]bool)
		for _, p := range m.processes {
			existingPIDs[p.PID] = true
		}
		for pid := range m.selected {
			if !existingPIDs[pid] {
				delete(m.selected, pid)
			}
		}
		// Adjust cursor if list shrunk
		filtered := m.filteredProcesses()
		if m.cursor >= len(filtered) {
			m.cursor = max(0, len(filtered)-1)
		}

	case killResultMsg:
		m.killIndex++

		if msg.success {
			// Remove from selected
			delete(m.selected, msg.pid)
		}

		// Check if more to kill
		if m.killIndex < len(m.toKill) {
			p := m.toKill[m.killIndex]
			return m, m.killProcess(p.PID, p.Port, len(m.toKill)-m.killIndex-1)
		}

		// All done
		killCount := len(m.toKill)
		m.toKill = nil
		m.killIndex = 0

		if killCount == 1 {
			if msg.success {
				m.statusMessage = fmt.Sprintf("Killed process on port %d", msg.port)
			} else {
				m.statusMessage = fmt.Sprintf("Failed to kill process %d", msg.pid)
			}
		} else {
			m.statusMessage = fmt.Sprintf("Killed %d processes", killCount)
		}
		m.statusTime = time.Now()
		return m, m.refreshPorts()
	}

	return m, nil
}

// View renders the UI
func (m Model) View() string {
	var s string

	// Title with selection count
	title := "portsweep"
	if m.showSystemPorts {
		title += " (all ports)"
	} else {
		title += " (user ports)"
	}
	if count := m.selectedCount(); count > 0 {
		title += " " + selectedCountStyle.Render(fmt.Sprintf("[%d selected]", count))
	}
	s += titleStyle.Render(title) + "\n"

	// Header
	header := fmt.Sprintf("    %-7s %-8s %-15s %-12s %s",
		"PORT", "PID", "PROCESS", "USER", "COMMAND")
	s += headerStyle.Render(header) + "\n"

	// Process list
	filtered := m.filteredProcesses()

	if len(filtered) == 0 {
		s += emptyStyle.Render("No listening ports found") + "\n"
	} else {
		for i, p := range filtered {
			// Checkbox
			checkbox := checkboxUnchecked
			if m.selected[p.PID] {
				checkbox = checkboxChecked
			}

			// Format command for display using smart formatters
			cmd := formatCommand(p.Command)
			maxCmdLen := 50
			if m.width > 60 {
				maxCmdLen = m.width - 55
			}
			if len(cmd) > maxCmdLen {
				cmd = cmd[:maxCmdLen-3] + "..."
			}

			line := fmt.Sprintf("%s %s %s %s %s %s",
				checkbox,
				portStyle.Render(fmt.Sprintf("%-7d", p.Port)),
				pidStyle.Render(fmt.Sprintf("%-8d", p.PID)),
				nameStyle.Render(truncate(p.Name, 15)),
				userStyle.Render(truncate(p.User, 12)),
				commandStyle.Render(cmd),
			)

			if i == m.cursor {
				// For selected row, we need to handle styling differently
				if m.selected[p.PID] {
					s += selectedStyle.Render(line) + "\n"
				} else {
					s += selectedStyle.Render(line) + "\n"
				}
			} else {
				if m.selected[p.PID] {
					s += checkedStyle.Render(line) + "\n"
				} else {
					s += normalStyle.Render(line) + "\n"
				}
			}
		}
	}

	// Show full command for focused row
	if len(filtered) > 0 && m.cursor < len(filtered) && !m.confirming {
		fullCmd := filtered[m.cursor].Command
		// Truncate to terminal width if needed
		maxLen := m.width - 4 // Account for "> " prefix and some padding
		if maxLen < 20 {
			maxLen = 80
		}
		if len(fullCmd) > maxLen {
			fullCmd = fullCmd[:maxLen-3] + "..."
		}
		s += "\n" + cmdDetailStyle.Render("> "+fullCmd)
	}

	// Confirmation prompt
	if m.confirming {
		if len(m.toKill) == 1 {
			p := m.toKill[0]
			s += confirmStyle.Render(fmt.Sprintf("\nKill process %d on port %d? (y/n)", p.PID, p.Port))
		} else {
			s += confirmStyle.Render(fmt.Sprintf("\nKill %d selected processes? (y/n)", len(m.toKill)))
		}
	}

	// Status message (show for 3 seconds)
	if m.statusMessage != "" && time.Since(m.statusTime) < 3*time.Second {
		s += "\n" + statusStyle.Render(m.statusMessage)
	}

	// Help
	help := "↑/k up • ↓/j down • space select • a select all • enter/d kill • r refresh • s system ports • q quit"
	s += "\n" + helpStyle.Render(help)

	return s
}

// truncate truncates a string to maxLen
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return fmt.Sprintf("%-*s", maxLen, s)
	}
	return s[:maxLen-1] + "…"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
