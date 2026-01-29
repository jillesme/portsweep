package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// Configuration constants
const (
	// RefreshInterval is how often the process list auto-refreshes
	RefreshInterval = 2 * time.Second

	// StatusDisplayDuration is how long status messages are shown
	StatusDisplayDuration = 3 * time.Second

	// SystemPortThreshold is the boundary between system and user ports
	SystemPortThreshold = 1024

	// DefaultCommandWidth is the minimum width for the command column
	DefaultCommandWidth = 50

	// MinTerminalWidth is the threshold for adjusting command width
	MinTerminalWidth = 60

	// ColumnWidthOffset accounts for other columns when calculating command width
	ColumnWidthOffset = 55

	// MinFullCommandWidth is the minimum width for the full command detail line
	MinFullCommandWidth = 20

	// DefaultFullCommandWidth is used when terminal width is unknown
	DefaultFullCommandWidth = 80
)

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
	initialFilter   string // filter from CLI argument (port or name)
	filterApplied   bool   // whether we've applied the initial filter
	searching       bool   // whether in search mode
	searchQuery     string // current search query
	lastError       error  // last error from port scanning
}

// NewModel creates a new Model with optional initial filter
func NewModel(initialFilter string) Model {
	return Model{
		processes:       []Process{},
		cursor:          0,
		selected:        make(map[int]bool),
		showSystemPorts: false,
		confirming:      false,
		toKill:          []Process{},
		initialFilter:   initialFilter,
		filterApplied:   false,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.refreshPorts(),
		m.tickCmd(),
	)
}

// tickCmd returns a command that sends a tick at the refresh interval
func (m Model) tickCmd() tea.Cmd {
	return tea.Tick(RefreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// refreshPorts fetches the current listening ports
func (m Model) refreshPorts() tea.Cmd {
	return func() tea.Msg {
		processes, err := GetListeningPorts()
		return refreshMsg{processes: processes, err: err}
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
				if port >= SystemPortThreshold {
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
			matchesCommand := strings.Contains(strings.ToLower(p.Command), query)
			matchesPort := false
			for _, port := range p.Ports {
				if strings.Contains(strconv.Itoa(port), m.searchQuery) {
					matchesPort = true
					break
				}
			}
			if !matchesName && !matchesCommand && !matchesPort {
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

// applyInitialFilter pre-selects processes matching the CLI filter argument
func (m *Model) applyInitialFilter() {
	if m.initialFilter == "" {
		return
	}

	// Check if filter is a port number (exact match)
	if port, err := strconv.Atoi(m.initialFilter); err == nil {
		// It's a port number - exact match
		for _, p := range m.processes {
			for _, pPort := range p.Ports {
				if pPort == port {
					m.selected[p.PID] = true
					break
				}
			}
		}
	} else {
		// It's a name/command filter - case-insensitive substring match
		filterLower := strings.ToLower(m.initialFilter)
		for _, p := range m.processes {
			nameLower := strings.ToLower(p.Name)
			cmdLower := strings.ToLower(p.Command)
			if strings.Contains(nameLower, filterLower) || strings.Contains(cmdLower, filterLower) {
				m.selected[p.PID] = true
			}
		}
	}
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
				m.statusMessage = fmt.Sprintf("Showing user ports only (>=%d)", SystemPortThreshold)
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
		// Handle refresh errors
		if msg.err != nil {
			m.lastError = msg.err
			m.statusMessage = "Error scanning ports"
			m.statusTime = time.Now()
			return m, nil
		}
		m.lastError = nil

		m.processes = msg.processes
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

		// Apply initial filter from CLI argument (only once)
		if m.initialFilter != "" && !m.filterApplied {
			m.filterApplied = true
			m.applyInitialFilter()
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
	var sb strings.Builder

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
	sb.WriteString(titleStyle.Render(title))
	sb.WriteByte('\n')

	// Header
	header := fmt.Sprintf("    %-18s %-8s %-15s %-12s %s",
		"PORT", "PID", "PROCESS", "USER", "COMMAND")
	sb.WriteString(headerStyle.Render(header))
	sb.WriteByte('\n')

	// Process list
	filtered := m.filteredProcesses()

	if len(filtered) == 0 {
		if m.searchQuery != "" {
			sb.WriteString(emptyStyle.Render(fmt.Sprintf("No processes match '%s'", m.searchQuery)))
		} else {
			sb.WriteString(emptyStyle.Render("No listening ports found"))
		}
		sb.WriteByte('\n')
	} else {
		for i, p := range filtered {
			// Checkbox
			checkbox := checkboxUnchecked
			if m.selected[p.PID] {
				checkbox = checkboxChecked
			}

			// Format command for display using smart formatters
			cmd := formatCommand(p.Command)
			maxCmdLen := DefaultCommandWidth
			if m.width > MinTerminalWidth {
				maxCmdLen = m.width - ColumnWidthOffset
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
				sb.WriteString(selectedStyle.Render(line))
			} else if m.selected[p.PID] {
				sb.WriteString(checkedStyle.Render(line))
			} else {
				sb.WriteString(normalStyle.Render(line))
			}
			sb.WriteByte('\n')
		}
	}

	// Show full command for focused row
	if len(filtered) > 0 && m.cursor < len(filtered) && !m.confirming {
		fullCmd := filtered[m.cursor].Command
		// Truncate to terminal width if needed
		maxLen := m.width - 4 // Account for "> " prefix and some padding
		if maxLen < MinFullCommandWidth {
			maxLen = DefaultFullCommandWidth
		}
		if len(fullCmd) > maxLen {
			fullCmd = fullCmd[:maxLen-3] + "..."
		}
		sb.WriteByte('\n')
		sb.WriteString(cmdDetailStyle.Render("> " + fullCmd))
	}

	// Confirmation prompt
	if m.confirming {
		if len(m.toKill) == 1 {
			p := m.toKill[0]
			portsStr := formatPorts(p.Ports, 40)
			if len(p.Ports) == 1 {
				sb.WriteString(confirmStyle.Render(fmt.Sprintf("\nKill process %d on port %s? (y/n)", p.PID, portsStr)))
			} else {
				sb.WriteString(confirmStyle.Render(fmt.Sprintf("\nKill process %d on ports %s? (y/n)", p.PID, portsStr)))
			}
		} else {
			sb.WriteString(confirmStyle.Render(fmt.Sprintf("\nKill %d selected processes? (y/n)", len(m.toKill))))
		}
	}

	// Status message (show for configured duration)
	if m.statusMessage != "" && time.Since(m.statusTime) < StatusDisplayDuration {
		sb.WriteByte('\n')
		sb.WriteString(statusStyle.Render(m.statusMessage))
	}

	// Search bar or Help
	if m.searching {
		sb.WriteByte('\n')
		sb.WriteString(searchStyle.Render("/" + m.searchQuery + "▌"))
	} else if m.searchQuery != "" {
		// Show filter indicator and modified help
		sb.WriteByte('\n')
		sb.WriteString(searchFilterStyle.Render(fmt.Sprintf("filter: %s", m.searchQuery)))
		help := "↑/k up • ↓/j down • space select • enter/d kill • / search • esc clear • q quit"
		sb.WriteByte('\n')
		sb.WriteString(helpStyle.Render(help))
	} else {
		help := "↑/k up • ↓/j down • space select • a select all • enter/d kill • / search • r refresh • s system ports • q quit"
		sb.WriteByte('\n')
		sb.WriteString(helpStyle.Render(help))
	}

	return sb.String()
}
