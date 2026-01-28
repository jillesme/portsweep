package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// version is set at build time via ldflags
var version = "dev"

func main() {
	var initialFilter string

	// Handle flags and arguments
	if len(os.Args) > 1 {
		arg := os.Args[1]
		if arg == "-v" || arg == "--version" || arg == "version" {
			fmt.Println("portsweep", version)
			return
		}
		if arg == "-h" || arg == "--help" || arg == "help" {
			printHelp()
			return
		}
		// Treat as filter argument (port number or process name)
		initialFilter = arg
	}

	p := tea.NewProgram(NewModel(initialFilter), tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running portsweep: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`portsweep - TUI for managing processes listening on ports

Usage:
  portsweep [flags]
  portsweep <port>      Kill process on specific port (e.g., portsweep 3000)
  portsweep <name>      Kill processes matching name (e.g., portsweep node)

Arguments:
  <port>        Port number to match (exact match)
  <name>        Process name or command to match (case-insensitive)

Flags:
  -h, --help      Show this help message
  -v, --version   Show version

Keybindings:
  ↑/k          Move up
  ↓/j          Move down
  space/tab    Select/deselect process
  a            Select all
  enter/d      Kill selected process(es)
  r            Refresh
  s            Toggle system ports (<1024)
  /            Search/filter processes
  q            Quit`)
}
