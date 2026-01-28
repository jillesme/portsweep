package main

import (
	"os/exec"
	"strconv"
	"strings"
)

// Process represents a process listening on a port
type Process struct {
	PID     int
	Port    int
	Name    string
	User    string
	Command string
}

// GetListeningPorts returns all processes listening on TCP ports
func GetListeningPorts() ([]Process, error) {
	// Run lsof to get listening TCP ports
	// -iTCP: only TCP connections
	// -sTCP:LISTEN: only listening sockets
	// -n: no hostname resolution
	// -P: no port name resolution
	cmd := exec.Command("lsof", "-iTCP", "-sTCP:LISTEN", "-n", "-P")
	output, err := cmd.Output()
	if err != nil {
		// lsof returns exit code 1 if no results, which is fine
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return []Process{}, nil
		}
		return nil, err
	}

	return parseLsofOutput(string(output))
}

// parseLsofOutput parses the lsof output into Process structs
func parseLsofOutput(output string) ([]Process, error) {
	lines := strings.Split(output, "\n")
	processes := make([]Process, 0)
	seen := make(map[int]bool) // Track PIDs we've already added

	for i, line := range lines {
		// Skip header line
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}

		// lsof output format:
		// COMMAND PID USER FD TYPE DEVICE SIZE/OFF NODE NAME
		// node    123 user 22u IPv4 ...    0t0      TCP  *:3000 (LISTEN)

		name := fields[0]
		pidStr := fields[1]
		user := fields[2]
		nameField := fields[len(fields)-1]

		// Handle "(LISTEN)" suffix
		if nameField == "(LISTEN)" && len(fields) >= 10 {
			nameField = fields[len(fields)-2]
		}

		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}

		// Skip if we've already seen this PID (lsof can show multiple entries per process)
		if seen[pid] {
			continue
		}

		// Parse port from name field (e.g., "*:3000" or "127.0.0.1:8080")
		port := parsePort(nameField)
		if port == 0 {
			continue
		}

		// Get full command line
		command := getFullCommand(pid)

		processes = append(processes, Process{
			PID:     pid,
			Port:    port,
			Name:    name,
			User:    user,
			Command: command,
		})
		seen[pid] = true
	}

	return processes, nil
}

// parsePort extracts the port number from a lsof NAME field
func parsePort(nameField string) int {
	// Handle formats like "*:3000", "127.0.0.1:8080", "[::1]:3000"
	parts := strings.Split(nameField, ":")
	if len(parts) < 2 {
		return 0
	}

	portStr := parts[len(parts)-1]
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0
	}

	return port
}

// getFullCommand gets the full command line for a PID
func getFullCommand(pid int) string {
	cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "command=")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}

// KillProcess sends SIGTERM to a process
func KillProcess(pid int) error {
	cmd := exec.Command("kill", strconv.Itoa(pid))
	return cmd.Run()
}
