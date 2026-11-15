package vintage

import "os"

// ExecutableAtStart is the running executable's path and file as the process
// found them when it started, read once as this package is initialised, so a
// binary put in place at the same path afterwards (an update) is never taken
// for the one running. fi is nil when either could not be read.
func ExecutableAtStart() (path string, fi os.FileInfo) { return startPath, startInfo }

var startPath, startInfo = readExecutable()

// readExecutable is the executable's path and file now.
func readExecutable() (string, os.FileInfo) {
	exe, err := os.Executable()
	if err != nil {
		return "", nil
	}
	fi, err := os.Stat(exe)
	if err != nil {
		return exe, nil
	}
	return exe, fi
}
