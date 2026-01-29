package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
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
	Search    key.Binding
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
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
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
	searching       bool   // whether in search mode
	searchQuery     string // current search query
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

// filteredProcesses returns processes filtered by system port setting and search query
func (m Model) filteredProcesses() []Process {
	filtered := make([]Process, 0)

	for _, p := range m.processes {
		// First, apply system port filter
		if !m.showSystemPorts {
			hasUserPort := false
			for _, port := range p.Ports {
				if port >= 1024 {
					hasUserPort = true
					break
				}
			}
			if !hasUserPort {
				continue
			}
		}

		// Then, apply search filter if query is set
		if m.searchQuery != "" {
			query := strings.ToLower(m.searchQuery)
			matchesName := strings.Contains(strings.ToLower(p.Name), query)
			matchesPort := false
			for _, port := range p.Ports {
				if strings.Contains(strconv.Itoa(port), m.searchQuery) {
					matchesPort = true
					break
				}
			}
			if !matchesName && !matchesPort {
				continue
			}
		}

		filtered = append(filtered, p)
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
					return m, m.killProcess(p.PID, p.LowestPort(), len(m.toKill)-1)
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

		// Search mode key handling
		if m.searching {
			switch msg.Type {
			case tea.KeyEsc:
				// Clear search and exit search mode
				m.searching = false
				m.searchQuery = ""
				// Reset cursor if out of bounds
				filtered := m.filteredProcesses()
				if m.cursor >= len(filtered) {
					m.cursor = max(0, len(filtered)-1)
				}
				return m, nil
			case tea.KeyBackspace:
				if len(m.searchQuery) > 0 {
					m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
					// Reset cursor if out of bounds after filter change
					filtered := m.filteredProcesses()
					if m.cursor >= len(filtered) {
						m.cursor = max(0, len(filtered)-1)
					}
				}
				return m, nil
			case tea.KeyEnter:
				// Exit search mode but keep the filter
				m.searching = false
				return m, nil
			case tea.KeyRunes:
				// Append typed characters to search query
				m.searchQuery += string(msg.Runes)
				// Reset cursor to 0 when search changes
				m.cursor = 0
				return m, nil
			}
			return m, nil
		}

		// Normal mode key handling
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, keys.Search):
			m.searching = true
			return m, nil

		case key.Matches(msg, keys.Cancel):
			// Clear search filter if active (Esc when not searching)
			if m.searchQuery != "" {
				m.searchQuery = ""
				m.cursor = 0
				return m, nil
			}

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
		// Sort by lowest port number
		sort.Slice(m.processes, func(i, j int) bool {
			return m.processes[i].LowestPort() < m.processes[j].LowestPort()
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
			return m, m.killProcess(p.PID, p.LowestPort(), len(m.toKill)-m.killIndex-1)
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
	header := fmt.Sprintf("    %-18s %-8s %-15s %-12s %s",
		"PORT", "PID", "PROCESS", "USER", "COMMAND")
	s += headerStyle.Render(header) + "\n"

	// Process list
	filtered := m.filteredProcesses()

	if len(filtered) == 0 {
		if m.searchQuery != "" {
			s += emptyStyle.Render(fmt.Sprintf("No processes match '%s'", m.searchQuery)) + "\n"
		} else {
			s += emptyStyle.Render("No listening ports found") + "\n"
		}
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
				portStyle.Render(fmt.Sprintf("%-18s", formatPorts(p.Ports, 18))),
				pidStyle.Render(fmt.Sprintf("%-8d", p.PID)),
				nameStyle.Render(truncate(p.Name, 15)),
				userStyle.Render(truncate(p.User, 12)),
				commandStyle.Render(cmd),
			)

			if i == m.cursor {
				s += selectedStyle.Render(line) + "\n"
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
			portsStr := formatPorts(p.Ports, 40)
			if len(p.Ports) == 1 {
				s += confirmStyle.Render(fmt.Sprintf("\nKill process %d on port %s? (y/n)", p.PID, portsStr))
			} else {
				s += confirmStyle.Render(fmt.Sprintf("\nKill process %d on ports %s? (y/n)", p.PID, portsStr))
			}
		} else {
			s += confirmStyle.Render(fmt.Sprintf("\nKill %d selected processes? (y/n)", len(m.toKill)))
		}
	}

	// Status message (show for 3 seconds)
	if m.statusMessage != "" && time.Since(m.statusTime) < 3*time.Second {
		s += "\n" + statusStyle.Render(m.statusMessage)
	}

	// Search bar or Help
	if m.searching {
		s += "\n" + searchStyle.Render("/"+m.searchQuery+"▌")
	} else if m.searchQuery != "" {
		// Show filter indicator and modified help
		s += "\n" + searchFilterStyle.Render(fmt.Sprintf("filter: %s", m.searchQuery))
		help := "↑/k up • ↓/j down • space select • enter/d kill • / search • esc clear • q quit"
		s += "\n" + helpStyle.Render(help)
	} else {
		help := "↑/k up • ↓/j down • space select • a select all • enter/d kill • / search • r refresh • s system ports • q quit"
		s += "\n" + helpStyle.Render(help)
	}

	return s
}

// truncate truncates a string to maxLen
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return fmt.Sprintf("%-*s", maxLen, s)
	}
	return s[:maxLen-1] + "…"
}

// formatPorts formats a list of ports for display with truncation
// maxWidth is the maximum character width for the output
func formatPorts(ports []int, maxWidth int) string {
	if len(ports) == 0 {
		return ""
	}

	// Start with the first port
	result := fmt.Sprintf("%d", ports[0])

	// Try to add more ports
	portsShown := 1
	for i := 1; i < len(ports); i++ {
		next := fmt.Sprintf(", %d", ports[i])
		remaining := len(ports) - i - 1

		// Calculate space needed for "+N" suffix if we stop here
		suffixLen := 0
		if remaining > 0 {
			suffixLen = len(fmt.Sprintf(" +%d", remaining+1))
		}

		// Check if adding this port would exceed max width
		if len(result)+len(next)+suffixLen > maxWidth {
			// Can't fit more, add suffix for remaining
			remainingCount := len(ports) - portsShown
			if remainingCount > 0 {
				result += fmt.Sprintf(" +%d", remainingCount)
			}
			break
		}

		result += next
		portsShown++
	}

	return result
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
