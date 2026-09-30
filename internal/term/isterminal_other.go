//go:build !darwin && !linux

package term

// isTerminalFd answers false on a platform abcd does not build for (the
// release targets are macOS and Linux): without a termios probe the check
// fails closed, so a consent gate declines as having no terminal to ask at
// rather than asking a question no one may be there to answer.
func isTerminalFd(uintptr) bool { return false }
