package launch

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// binaryNameSpelling is the body of BinaryNamePattern as it would be written
// out by hand in a regexp literal.
const binaryNameSpelling = `abcd(-[a-z0-9]+-[a-z0-9]+)?`

// TestBinaryNamePatternIsSpelledOnce holds the build-name recogniser to one
// spelling: every Go file outside this package's own declaration that wants to
// know whether a file is an abcd binary builds on BinaryNamePattern, so a new
// build name is taught in one place. A second hand-written copy is how the
// status-line recursion guard came to miss `abcd-darwin-arm64`.
func TestBinaryNamePatternIsSpelledOnce(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	var offenders []string
	for _, top := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, top), func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				return nil
			}
			raw, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			if !strings.Contains(string(raw), binaryNameSpelling) {
				return nil
			}
			rel, _ := filepath.Rel(root, p)
			if filepath.ToSlash(rel) != "internal/core/launch/bundle.go" {
				offenders = append(offenders, filepath.ToSlash(rel))
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(offenders) > 0 {
		t.Errorf("the abcd build-name pattern is spelled outside launch.BinaryNamePattern in %v; build on the exported pattern instead", offenders)
	}
}
