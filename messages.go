package main

import "time"

// TUI messages for the Elm architecture

// tickMsg is sent periodically to trigger auto-refresh
type tickMsg time.Time

// refreshMsg contains the updated process list or an error
type refreshMsg struct {
	processes []Process
	err       error
}

// killResultMsg reports the result of a kill operation
type killResultMsg struct {
	success   bool
	pid       int
	port      int
	remaining int // how many left to kill
}
