//go:build live

package runner

// NewClaudeForLive is the claude runner for a route, for the live check in
// package runner_test, which imports the loop's bundled classes and so cannot
// sit in this package: the loop imports the runner.
func NewClaudeForLive(route string) Runner { return newClaude(route) }
