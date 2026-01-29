package main

import (
	"testing"
)

func TestParseLsofOutput(t *testing.T) {
	// Mock command lookup that returns predictable results
	mockCommandLookup := func(pid int) string {
		switch pid {
		case 123:
			return "node /Users/test/project/server.js"
		case 456:
			return "/usr/bin/python3 app.py"
		case 789:
			return "nginx: master process"
		default:
			return ""
		}
	}

	tests := []struct {
		name     string
		input    string
		expected []Process
	}{
		{
			name: "single process single port",
			input: `COMMAND   PID   USER   FD   TYPE     DEVICE SIZE/OFF NODE NAME
node      123   user   22u  IPv4 0x123456      0t0  TCP *:3000 (LISTEN)`,
			expected: []Process{
				{PID: 123, Ports: []int{3000}, Name: "node", User: "user", Command: "node /Users/test/project/server.js"},
			},
		},
		{
			name: "single process multiple ports",
			input: `COMMAND   PID   USER   FD   TYPE     DEVICE SIZE/OFF NODE NAME
node      123   user   22u  IPv4 0x123456      0t0  TCP *:3000 (LISTEN)
node      123   user   23u  IPv4 0x123457      0t0  TCP *:3001 (LISTEN)
node      123   user   24u  IPv4 0x123458      0t0  TCP *:8080 (LISTEN)`,
			expected: []Process{
				{PID: 123, Ports: []int{3000, 3001, 8080}, Name: "node", User: "user", Command: "node /Users/test/project/server.js"},
			},
		},
		{
			name: "multiple processes",
			input: `COMMAND   PID   USER   FD   TYPE     DEVICE SIZE/OFF NODE NAME
node      123   user   22u  IPv4 0x123456      0t0  TCP *:3000 (LISTEN)
python3   456   root   5u   IPv4 0x789012      0t0  TCP 127.0.0.1:8000 (LISTEN)`,
			expected: []Process{
				{PID: 123, Ports: []int{3000}, Name: "node", User: "user", Command: "node /Users/test/project/server.js"},
				{PID: 456, Ports: []int{8000}, Name: "python3", User: "root", Command: "/usr/bin/python3 app.py"},
			},
		},
		{
			name: "IPv6 address",
			input: `COMMAND   PID   USER   FD   TYPE     DEVICE SIZE/OFF NODE NAME
node      123   user   22u  IPv6 0x123456      0t0  TCP [::1]:3000 (LISTEN)`,
			expected: []Process{
				{PID: 123, Ports: []int{3000}, Name: "node", User: "user", Command: "node /Users/test/project/server.js"},
			},
		},
		{
			name: "deduplication across interfaces",
			input: `COMMAND   PID   USER   FD   TYPE     DEVICE SIZE/OFF NODE NAME
node      123   user   22u  IPv4 0x123456      0t0  TCP *:3000 (LISTEN)
node      123   user   23u  IPv6 0x123457      0t0  TCP [::]:3000 (LISTEN)`,
			expected: []Process{
				{PID: 123, Ports: []int{3000}, Name: "node", User: "user", Command: "node /Users/test/project/server.js"},
			},
		},
		{
			name: "specific IP binding",
			input: `COMMAND   PID   USER   FD   TYPE     DEVICE SIZE/OFF NODE NAME
nginx     789   www    10u  IPv4 0xabcdef      0t0  TCP 192.168.1.100:80 (LISTEN)`,
			expected: []Process{
				{PID: 789, Ports: []int{80}, Name: "nginx", User: "www", Command: "nginx: master process"},
			},
		},
		{
			name:     "empty output",
			input:    "",
			expected: []Process{},
		},
		{
			name: "header only",
			input: `COMMAND   PID   USER   FD   TYPE     DEVICE SIZE/OFF NODE NAME
`,
			expected: []Process{},
		},
		{
			name: "malformed line (too few fields)",
			input: `COMMAND   PID   USER   FD   TYPE     DEVICE SIZE/OFF NODE NAME
incomplete line here`,
			expected: []Process{},
		},
		{
			name: "ports are sorted ascending",
			input: `COMMAND   PID   USER   FD   TYPE     DEVICE SIZE/OFF NODE NAME
node      123   user   22u  IPv4 0x123456      0t0  TCP *:9000 (LISTEN)
node      123   user   23u  IPv4 0x123457      0t0  TCP *:3000 (LISTEN)
node      123   user   24u  IPv4 0x123458      0t0  TCP *:5000 (LISTEN)`,
			expected: []Process{
				{PID: 123, Ports: []int{3000, 5000, 9000}, Name: "node", User: "user", Command: "node /Users/test/project/server.js"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseLsofOutput(tt.input, mockCommandLookup)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d processes, got %d", len(tt.expected), len(result))
			}

			// Create a map for easier comparison (order is not guaranteed from map iteration)
			resultMap := make(map[int]Process)
			for _, p := range result {
				resultMap[p.PID] = p
			}

			for _, exp := range tt.expected {
				got, exists := resultMap[exp.PID]
				if !exists {
					t.Errorf("expected process with PID %d not found", exp.PID)
					continue
				}

				if got.Name != exp.Name {
					t.Errorf("PID %d: expected Name %q, got %q", exp.PID, exp.Name, got.Name)
				}
				if got.User != exp.User {
					t.Errorf("PID %d: expected User %q, got %q", exp.PID, exp.User, got.User)
				}
				if got.Command != exp.Command {
					t.Errorf("PID %d: expected Command %q, got %q", exp.PID, exp.Command, got.Command)
				}

				if len(got.Ports) != len(exp.Ports) {
					t.Errorf("PID %d: expected %d ports, got %d", exp.PID, len(exp.Ports), len(got.Ports))
					continue
				}
				for i, port := range exp.Ports {
					if got.Ports[i] != port {
						t.Errorf("PID %d: expected port[%d]=%d, got %d", exp.PID, i, port, got.Ports[i])
					}
				}
			}
		})
	}
}

func TestParsePort(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"wildcard IPv4", "*:3000", 3000},
		{"localhost", "127.0.0.1:8080", 8080},
		{"specific IP", "192.168.1.1:443", 443},
		{"IPv6 localhost", "[::1]:3000", 3000},
		{"IPv6 wildcard", "[::]:8080", 8080},
		{"high port", "*:65535", 65535},
		{"low port", "*:22", 22},
		{"no colon", "3000", 0},
		{"empty string", "", 0},
		{"invalid port", "*:abc", 0},
		{"only colon", ":", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parsePort(tt.input)
			if result != tt.expected {
				t.Errorf("parsePort(%q) = %d, expected %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestProcessLowestPort(t *testing.T) {
	tests := []struct {
		name     string
		ports    []int
		expected int
	}{
		{"single port", []int{3000}, 3000},
		{"multiple ports sorted", []int{80, 443, 8080}, 80},
		{"empty ports", []int{}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Process{Ports: tt.ports}
			result := p.LowestPort()
			if result != tt.expected {
				t.Errorf("LowestPort() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

// MockScanner implements PortScanner for testing
type MockScanner struct {
	Processes []Process
	Err       error
}

func (m *MockScanner) GetListeningPorts() ([]Process, error) {
	return m.Processes, m.Err
}

// MockKiller implements ProcessKiller for testing
type MockKiller struct {
	KilledPIDs []int
	Err        error
}

func (m *MockKiller) Kill(pid int) error {
	if m.Err != nil {
		return m.Err
	}
	m.KilledPIDs = append(m.KilledPIDs, pid)
	return nil
}

func TestMockScanner(t *testing.T) {
	// Verify our mock implements the interface correctly
	scanner := &MockScanner{
		Processes: []Process{
			{PID: 123, Ports: []int{3000}, Name: "node"},
		},
	}

	var _ PortScanner = scanner // Compile-time interface check

	procs, err := scanner.GetListeningPorts()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(procs) != 1 {
		t.Fatalf("expected 1 process, got %d", len(procs))
	}
	if procs[0].PID != 123 {
		t.Errorf("expected PID 123, got %d", procs[0].PID)
	}
}

func TestMockKiller(t *testing.T) {
	// Verify our mock implements the interface correctly
	killer := &MockKiller{}

	var _ ProcessKiller = killer // Compile-time interface check

	err := killer.Kill(123)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(killer.KilledPIDs) != 1 || killer.KilledPIDs[0] != 123 {
		t.Errorf("expected KilledPIDs=[123], got %v", killer.KilledPIDs)
	}
}
