package main

import "fmt"

// truncate truncates a string to maxLen, padding with spaces if shorter
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return fmt.Sprintf("%-*s", maxLen, s)
	}
	return s[:maxLen-1] + "…"
}

// formatPorts formats a list of ports for display with truncation.
// maxWidth is the maximum character width for the output.
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
