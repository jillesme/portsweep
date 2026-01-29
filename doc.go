// Package main implements portsweep, a TUI application for managing processes
// listening on TCP ports.
//
// portsweep provides an interactive terminal interface to:
//   - View all processes listening on TCP ports
//   - Select and kill multiple processes at once
//   - Filter by port number or process name
//   - Search processes interactively
//
// The application uses the Bubbletea framework with the Elm architecture pattern
// for state management.
//
// # Architecture
//
// The codebase is organized into the following components:
//
//   - model.go: Core TUI model with Init, Update, and View methods
//   - ports.go: Process discovery (PortScanner interface) and killing (ProcessKiller interface)
//   - formatter.go: Smart command string formatting with extensible CommandFormatter interface
//   - styles.go: Lipgloss styles for terminal rendering
//   - keys.go: Key bindings configuration
//   - messages.go: TUI message types for the Elm architecture
//   - helpers.go: Utility functions for string formatting
//
// # Extensibility
//
// Custom command formatters can be registered using RegisterFormatter:
//
//	RegisterFormatter(&MyCustomFormatter{})
//
// The PortScanner and ProcessKiller interfaces allow for custom implementations
// and easier testing through dependency injection.
package main
