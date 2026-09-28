package scanner

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

// TestNameGuardHooksReadTheScannersJSONLayers holds the pre-commit name guard's
// decode bound to this package's (iss-2609280944560197). The guard is shell and
// awk, so it cannot import maxJSONDecodeLayers; it declares the number once, as
// decode_layers, and this test is what keeps the two one number. A guard that
// reads fewer layers than the scanner lets through a name the store-before-commit
// redactors would have read. Both copies are held: abcd's own hook and the one
// ahoy scaffolds into a managed repository.
func TestNameGuardHooksReadTheScannersJSONLayers(t *testing.T) {
	decl := regexp.MustCompile(`(?m)^decode_layers=([0-9]+)$`)
	for _, rel := range []string{
		"../../../.githooks/pre-commit",
		"../../core/ahoy/defaults/pre-commit",
	} {
		src, err := os.ReadFile(filepath.FromSlash(rel))
		if err != nil {
			t.Fatalf("read the name guard: %v", err)
		}
		m := decl.FindAllSubmatch(src, -1)
		if len(m) != 1 {
			t.Errorf("%s: want exactly one decode_layers=N declaration, found %d", rel, len(m))
			continue
		}
		n, err := strconv.Atoi(string(m[0][1]))
		if err != nil {
			t.Fatalf("%s: decode_layers: %v", rel, err)
		}
		if n != maxJSONDecodeLayers {
			t.Errorf("%s: decode_layers=%d, the scanner reads %d JSON layers (maxJSONDecodeLayers)", rel, n, maxJSONDecodeLayers)
		}
	}
}
