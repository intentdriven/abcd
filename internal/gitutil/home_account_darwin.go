//go:build darwin && !osusergo

package gitutil

// accountFromOSUser: on darwin, os/user resolves a uid through libSystem's
// getpwuid_r, with or without cgo, and never from the environment.
const accountFromOSUser = true
