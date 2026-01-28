package main

import "testing"

func TestFormatCommand(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// .bin symlinks (npm/pnpm/yarn)
		{
			name:     "node_modules .bin vite",
			input:    "node /Users/jilles/Code/cf-question-agent/node_modules/.bin/vite",
			expected: "vite (cf-question-agent)",
		},
		{
			name:     "node_modules .bin wrangler",
			input:    "node /Users/jilles/Code/my-worker/node_modules/.bin/wrangler dev",
			expected: "wrangler (my-worker)",
		},
		{
			name:     "node_modules .bin without project",
			input:    "node /tmp/node_modules/.bin/tsc",
			expected: "tsc",
		},

		// pnpm packages
		{
			name:     "pnpm scoped package with project",
			input:    "node /Users/jilles/Code/goal-shark-api/node_modules/.pnpm/@cloudflare+workerd@1.2.3/node_modules/@cloudflare/workerd/bin/workerd",
			expected: "workerd (goal-shark-api)",
		},
		{
			name:     "pnpm regular package with project",
			input:    "node /Users/jilles/Code/my-project/node_modules/.pnpm/vite@5.0.0/node_modules/vite/bin/vite.js",
			expected: "vite (my-project)",
		},

		// npm packages
		{
			name:     "npm scoped package",
			input:    "node /Users/jilles/Code/my-app/node_modules/@cloudflare/workers-sdk/bin/wrangler.js",
			expected: "workers-sdk (my-app)",
		},
		{
			name:     "npm regular package",
			input:    "node /Users/jilles/Code/my-app/node_modules/vite/bin/vite.js",
			expected: "vite (my-app)",
		},

		// Homebrew
		{
			name:     "homebrew package",
			input:    "/opt/homebrew/Cellar/opencode/1.0.220/libexec/lib/node_modules/opencode/bin/opencode.js",
			expected: "opencode",
		},
		{
			name:     "homebrew intel mac",
			input:    "/usr/local/Cellar/node/20.0.0/bin/node",
			expected: "node",
		},

		// .app bundles
		{
			name:     "app bundle Spotify",
			input:    "/Applications/Spotify.app/Contents/MacOS/Spotify",
			expected: "Spotify",
		},
		{
			name:     "app bundle Raycast",
			input:    "/Applications/Raycast.app/Contents/MacOS/Raycast_UPDATED_VERSION",
			expected: "Raycast",
		},

		// Project paths
		{
			name:     "node in Code directory",
			input:    "node /Users/jilles/Code/cf-question-agent/src/index.js",
			expected: "node (cf-question-agent)",
		},
		{
			name:     "node in Cloudflare directory",
			input:    "node /Users/jilles/Cloudflare/workers-sdk/packages/wrangler/bin/wrangler.js",
			expected: "node (workers-sdk)",
		},

		// System binaries
		{
			name:     "system binary rapportd",
			input:    "/usr/libexec/rapportd",
			expected: "rapportd",
		},
		{
			name:     "system binary with args",
			input:    "/usr/bin/ssh-agent -l",
			expected: "ssh-agent",
		},

		// Fallback cases
		{
			name:     "simple node command",
			input:    "node server.js",
			expected: "node (server)",
		},
		{
			name:     "already clean command with parens",
			input:    "node (npx remotion studio)",
			expected: "node", // Don't add extra parens when already formatted
		},
		{
			name:     "python script",
			input:    "python3 /path/to/script.py",
			expected: "python3 (script)",
		},

		// Edge cases
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "simple executable",
			input:    "nginx",
			expected: "nginx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatCommand(tt.input)
			if result != tt.expected {
				t.Errorf("formatCommand(%q)\n  got:      %q\n  expected: %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractPnpmPackage(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"node_modules/.pnpm/@cloudflare+workerd@1.2.3/node_modules", "workerd"},
		{"node_modules/.pnpm/vite@5.0.0/node_modules", "vite"},
		{"node_modules/.pnpm/@types+node@20.0.0/node_modules", "node"},
		{"/some/path", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractPnpmPackage(tt.input)
			if result != tt.expected {
				t.Errorf("extractPnpmPackage(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractNpmPackage(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"node_modules/vite/bin/vite.js", "vite"},
		{"node_modules/@cloudflare/workers-sdk/bin/wrangler.js", "workers-sdk"},
		{"/some/path", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractNpmPackage(tt.input)
			if result != tt.expected {
				t.Errorf("extractNpmPackage(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractProjectName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/Users/jilles/Code/my-project/src/index.js", "my-project"},
		{"/Users/jilles/Projects/webapp/server.js", "webapp"},
		{"/Users/jilles/Cloudflare/workers-sdk/packages", "workers-sdk"},
		{"/usr/bin/node", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractProjectName(tt.input)
			if result != tt.expected {
				t.Errorf("extractProjectName(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}
