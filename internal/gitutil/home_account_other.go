//go:build !darwin || osusergo

package gitutil

// accountFromOSUser: off darwin, or with the osusergo tag, os/user's pure-Go
// Current can answer the home from $HOME, so the passwd file is read instead.
const accountFromOSUser = false
