package ahoy

import (
	"strings"
	"testing"
)

// TestColdCacheRemedySaysHowTheOneLinerVerifies: the remedy sends the operator
// to the README one-liner, so it says what that one-liner checks under the
// product thinker's ruling of 2026-10-12 (iss-2610120617521209): the signed
// build record through gh attestation verify when gh is present, and the
// checksum alone, with a warning, without it. A remedy that promised only the
// checksum would describe a weaker install than the one it points at, and one
// that promised provenance unconditionally would overstate a gh-less install.
func TestColdCacheRemedySaysHowTheOneLinerVerifies(t *testing.T) {
	flat := strings.Join(strings.Fields(coldCacheRemedy), " ")
	for _, want := range []string{
		"https://github.com/intentdriven/abcd#install",
		"gh attestation verify",
		"signed build record",
		"checksums.txt",
		"warns",
		"abcd ahoy install",
	} {
		if !strings.Contains(flat, want) {
			t.Errorf("the cold-cache remedy must say %q:\n%s", want, coldCacheRemedy)
		}
	}
}
