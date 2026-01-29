package main

import (
	"path/filepath"
	"regexp"
	"strings"
)

// Pre-compiled regular expressions for better performance
var (
	binSymlinkRegex  = regexp.MustCompile(`node_modules/\.bin/([^/\s]+)`)
	homebrewRegex    = regexp.MustCompile(`/(?:opt/homebrew|usr/local)/Cellar/([^/]+)/`)
	appBundleRegex   = regexp.MustCompile(`/([^/]+)\.app/Contents/`)
	pnpmPackageRegex = regexp.MustCompile(`node_modules/\.pnpm/([^/]+)`)
	npmPackageRegex  = regexp.MustCompile(`node_modules/([^/]+(?:/[^/]+)?)`)
)

// CommandFormatter is an interface for formatting command strings.
// Implement this interface to add custom formatting for specific applications.
type CommandFormatter interface {
	// Name returns the formatter name (for debugging/logging)
	Name() string

	// CanFormat returns true if this formatter can handle the given command
	CanFormat(cmd string) bool

	// Format returns the formatted command string
	Format(cmd string) string
}

// formatCommand applies all registered formatters to produce a readable command string.
// It tries each formatter in order and returns the first successful format.
func formatCommand(cmd string) string {
	if cmd == "" {
		return ""
	}

	// Try each formatter in priority order
	for _, formatter := range registeredFormatters {
		if formatter.CanFormat(cmd) {
			result := formatter.Format(cmd)
			if result != "" {
				return result
			}
		}
	}

	// Fallback: extract base executable name
	return fallbackFormat(cmd)
}

// registeredFormatters holds all active formatters in priority order.
// Formatters earlier in the list take precedence.
var registeredFormatters = []CommandFormatter{
	&BinSymlinkFormatter{}, // Must come before pnpm/npm formatters
	&PnpmFormatter{},
	&NpmFormatter{},
	&HomebrewFormatter{},
	&AppBundleFormatter{},
	&ProjectFormatter{},
	&SystemBinaryFormatter{},
}

// RegisterFormatter adds a custom formatter to the beginning of the list (highest priority).
// This allows extending the formatting behavior at runtime.
func RegisterFormatter(f CommandFormatter) {
	registeredFormatters = append([]CommandFormatter{f}, registeredFormatters...)
}

// =============================================================================
// Built-in Formatters
// =============================================================================

// BinSymlinkFormatter handles commands running from node_modules/.bin/ symlinks.
// These are typically created by npm/pnpm/yarn when installing packages with binaries.
type BinSymlinkFormatter struct{}

func (f *BinSymlinkFormatter) Name() string { return "bin-symlink" }

func (f *BinSymlinkFormatter) CanFormat(cmd string) bool {
	return strings.Contains(cmd, "node_modules/.bin/")
}

func (f *BinSymlinkFormatter) Format(cmd string) string {
	// Extract the binary name from node_modules/.bin/<binary>
	// Example: node /path/to/project/node_modules/.bin/vite -> vite (project)
	matches := binSymlinkRegex.FindStringSubmatch(cmd)

	var binName string
	if len(matches) >= 2 {
		binName = matches[1]
	}

	projectName := extractProjectName(cmd)

	if binName != "" {
		if projectName != "" {
			return binName + " (" + projectName + ")"
		}
		return binName
	}

	return ""
}

// PnpmFormatter handles commands running from pnpm's node_modules structure.
type PnpmFormatter struct{}

func (f *PnpmFormatter) Name() string { return "pnpm" }

func (f *PnpmFormatter) CanFormat(cmd string) bool {
	return strings.Contains(cmd, "node_modules/.pnpm/")
}

func (f *PnpmFormatter) Format(cmd string) string {
	executable := extractExecutable(cmd)
	pkgName := extractPnpmPackage(cmd)
	projectName := extractProjectName(cmd)

	if pkgName != "" {
		if projectName != "" {
			return pkgName + " (" + projectName + ")"
		}
		return pkgName
	}

	if projectName != "" && executable != "" {
		return executable + " (" + projectName + ")"
	}

	return ""
}

// NpmFormatter handles commands running from npm's node_modules structure.
type NpmFormatter struct{}

func (f *NpmFormatter) Name() string { return "npm" }

func (f *NpmFormatter) CanFormat(cmd string) bool {
	// Match node_modules but NOT .pnpm (pnpm formatter handles that)
	return strings.Contains(cmd, "node_modules/") && !strings.Contains(cmd, "node_modules/.pnpm/")
}

func (f *NpmFormatter) Format(cmd string) string {
	executable := extractExecutable(cmd)
	pkgName := extractNpmPackage(cmd)
	projectName := extractProjectName(cmd)

	if pkgName != "" {
		if projectName != "" {
			return pkgName + " (" + projectName + ")"
		}
		return pkgName
	}

	if projectName != "" && executable != "" {
		return executable + " (" + projectName + ")"
	}

	return ""
}

// HomebrewFormatter handles commands installed via Homebrew.
type HomebrewFormatter struct{}

func (f *HomebrewFormatter) Name() string { return "homebrew" }

func (f *HomebrewFormatter) CanFormat(cmd string) bool {
	return strings.Contains(cmd, "/opt/homebrew/Cellar/") ||
		strings.Contains(cmd, "/usr/local/Cellar/")
}

func (f *HomebrewFormatter) Format(cmd string) string {
	// Pattern: /opt/homebrew/Cellar/<package>/<version>/...
	// or /usr/local/Cellar/<package>/<version>/...
	matches := homebrewRegex.FindStringSubmatch(cmd)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// AppBundleFormatter handles macOS .app bundles.
type AppBundleFormatter struct{}

func (f *AppBundleFormatter) Name() string { return "app-bundle" }

func (f *AppBundleFormatter) CanFormat(cmd string) bool {
	return strings.Contains(cmd, ".app/Contents/")
}

func (f *AppBundleFormatter) Format(cmd string) string {
	// Pattern: /path/to/Name.app/Contents/...
	matches := appBundleRegex.FindStringSubmatch(cmd)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// ProjectFormatter handles commands running from common project directories.
type ProjectFormatter struct{}

func (f *ProjectFormatter) Name() string { return "project" }

// projectDirs contains common project directory indicators.
// These patterns are used to extract project names from command paths.
var projectDirs = []string{
	"/Code/",
	"/Projects/",
	"/Developer/",
	"/Sites/",
	"/src/",
	"/repos/",
	"/git/",
	"/workspace/",
	"/Cloudflare/", // Common for Cloudflare employees
	"/OSS/",        // Open source projects
}

func (f *ProjectFormatter) CanFormat(cmd string) bool {
	for _, dir := range projectDirs {
		if strings.Contains(cmd, dir) {
			return true
		}
	}
	return false
}

func (f *ProjectFormatter) Format(cmd string) string {
	executable := extractExecutable(cmd)
	projectName := extractProjectName(cmd)

	if projectName != "" && executable != "" {
		return executable + " (" + projectName + ")"
	}

	return ""
}

// SystemBinaryFormatter handles system binaries (simplifies to just the binary name).
type SystemBinaryFormatter struct{}

func (f *SystemBinaryFormatter) Name() string { return "system" }

// systemPaths contains common system binary paths.
var systemPaths = []string{
	"/usr/bin/",
	"/usr/sbin/",
	"/usr/libexec/",
	"/bin/",
	"/sbin/",
	"/System/",
}

func (f *SystemBinaryFormatter) CanFormat(cmd string) bool {
	for _, path := range systemPaths {
		if strings.HasPrefix(cmd, path) {
			return true
		}
	}
	return false
}

func (f *SystemBinaryFormatter) Format(cmd string) string {
	return extractExecutable(cmd)
}

// =============================================================================
// Helper Functions
// =============================================================================

// extractExecutable returns the base name of the executable from a command string.
func extractExecutable(cmd string) string {
	if cmd == "" {
		return ""
	}

	// Split by spaces to get executable path
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return ""
	}

	execPath := parts[0]
	return filepath.Base(execPath)
}

// extractProjectName tries to find a project name from common directory patterns.
func extractProjectName(path string) string {
	// Try each project directory pattern
	for _, dir := range projectDirs {
		if idx := strings.Index(path, dir); idx != -1 {
			// Extract the project name (first directory after the pattern)
			remaining := path[idx+len(dir):]
			parts := strings.Split(remaining, "/")
			if len(parts) > 0 && parts[0] != "" {
				return parts[0]
			}
		}
	}
	return ""
}

// extractPnpmPackage extracts the package name from a pnpm node_modules path.
// Example: node_modules/.pnpm/@cloudflare+workerd@1.2.3/... -> workerd
// Example: node_modules/.pnpm/vite@5.0.0/... -> vite
func extractPnpmPackage(path string) string {
	matches := pnpmPackageRegex.FindStringSubmatch(path)
	if len(matches) < 2 {
		return ""
	}

	pkgPart := matches[1]

	// Handle scoped packages: @scope+name@version -> name
	// Handle regular packages: name@version -> name
	if strings.HasPrefix(pkgPart, "@") {
		// Scoped package: @cloudflare+workerd@1.2.3
		// Find the + which separates scope from name in pnpm
		plusIdx := strings.Index(pkgPart, "+")
		if plusIdx != -1 {
			// Get everything after the +
			afterPlus := pkgPart[plusIdx+1:]
			// Remove version (@x.y.z)
			atIdx := strings.Index(afterPlus, "@")
			if atIdx != -1 {
				return afterPlus[:atIdx]
			}
			return afterPlus
		}
	}

	// Regular package: vite@5.0.0
	atIdx := strings.Index(pkgPart, "@")
	if atIdx != -1 {
		return pkgPart[:atIdx]
	}

	return pkgPart
}

// extractNpmPackage extracts the package name from an npm node_modules path.
// Example: node_modules/vite/bin/vite.js -> vite
// Example: node_modules/@cloudflare/workerd/... -> workerd
func extractNpmPackage(path string) string {
	matches := npmPackageRegex.FindStringSubmatch(path)
	if len(matches) < 2 {
		return ""
	}

	pkgPart := matches[1]

	// Handle scoped packages: @scope/name -> name
	if strings.HasPrefix(pkgPart, "@") {
		parts := strings.Split(pkgPart, "/")
		if len(parts) >= 2 {
			return parts[1]
		}
	}

	// Regular package - might have trailing path
	parts := strings.Split(pkgPart, "/")
	return parts[0]
}

// fallbackFormat provides a simple fallback when no formatter matches.
func fallbackFormat(cmd string) string {
	executable := extractExecutable(cmd)
	if executable == "" {
		// If we can't extract executable, show truncated command
		if len(cmd) > 30 {
			return cmd[:27] + "..."
		}
		return cmd
	}

	// For node/python/ruby, try to show something meaningful from args
	parts := strings.Fields(cmd)
	if len(parts) > 1 && isScriptRunner(executable) {
		// Get the script argument
		arg := parts[1]
		// Skip if arg looks like a flag or already has parens (already formatted)
		if !strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "(") {
			argBase := filepath.Base(arg)
			// Remove common extensions
			argBase = strings.TrimSuffix(argBase, ".js")
			argBase = strings.TrimSuffix(argBase, ".ts")
			argBase = strings.TrimSuffix(argBase, ".py")
			argBase = strings.TrimSuffix(argBase, ".rb")

			if argBase != "" && argBase != executable {
				return executable + " (" + argBase + ")"
			}
		}
	}

	return executable
}

// scriptRunners is a set of executables that typically run scripts.
var scriptRunners = map[string]bool{
	"node":    true,
	"python":  true,
	"python3": true,
	"ruby":    true,
	"perl":    true,
	"php":     true,
}

// isScriptRunner returns true if the executable typically runs scripts.
func isScriptRunner(exec string) bool {
	return scriptRunners[exec]
}
