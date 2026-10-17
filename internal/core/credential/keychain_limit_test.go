package credential

import (
	"os"
	"strings"
	"testing"
)

// The macOS keychain is driven through `security -i`, which reads one command
// line of at most 4096 bytes: measured on a real Mac on 2026-10-12, a value
// whose add line is 4096 bytes round-trips and one whose line is 4098 is
// refused (iss-2609281654467661). abcd refuses a value that would pass that
// line before anything is written, naming the limit, and stores one that fits.
func TestKeychainRefusesAValueLongerThanItsCommandLineBeforeWriting(t *testing.T) {
	dir := withFakeKeychain(t, keychainMacOS)
	name := "abcd-limit-test-xxxxxxxx"
	err := keychainStore(name, strings.Repeat("a", 2002))
	if err == nil || !strings.Contains(err.Error(), "4096") {
		t.Fatalf("a value past the line limit must be refused naming it, got %v", err)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Fatalf("the refusal must come before anything is written, found %d item(s)", len(entries))
	}
	if err := keychainStore(name, strings.Repeat("a", 2001)); err != nil {
		t.Fatalf("a value whose line is exactly 4096 bytes is stored: %v", err)
	}
}
