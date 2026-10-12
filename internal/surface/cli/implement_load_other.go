//go:build !unix

package cli

import "os"

// signalledBy reports no signal: outside unix a process ends with an exit code
// alone, and the caller keeps its fallback for a non-positive one.
func signalledBy(*os.ProcessState) (int, bool) { return 0, false }
