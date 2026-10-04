package credential

import (
	"sync/atomic"
	"testing"
)

// Seams for the external test package (guide_test.go), which drives the
// guided connect of internal/core/oracle against this package's fake
// keychain. They exist only in this package's test binary.

// KeychainMacOS is the fake keychain's macOS kind.
const KeychainMacOS = keychainMacOS

// WithFakeKeychain points the keychain home at the fake for one test and
// returns the directory its items are kept in, one file per item.
func WithFakeKeychain(t *testing.T, kind string) string { return withFakeKeychain(t, kind) }

// CountKeychainReaches wraps the keychain's locator, already pointed at the
// fake, so every reach for the keychain tool (a lookup or a store) is
// counted.
func CountKeychainReaches(t *testing.T) *atomic.Int32 {
	t.Helper()
	var n atomic.Int32
	inner := locateKeychain
	locateKeychain = func() (keychainTool, error) {
		n.Add(1)
		return inner()
	}
	t.Cleanup(func() { locateKeychain = inner })
	return &n
}
