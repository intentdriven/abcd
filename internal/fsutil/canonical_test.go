package fsutil

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// nonCanonicalPrimitiveRe matches a private redefinition of a durable-write or
// real-dir primitive — the exact names iss-32 consolidates. The canonical home
// is internal/fsutil (exported WriteFileAtomic / IsRealDir / EnsureRealDir); any
// lowercase redefinition elsewhere is a divergent copy.
//
// ensureRealDir joins the list in iss-2609091128479544. The principle names
// directory validation and forbids a third copy, and three had accumulated —
// lifeboat's, intent's, and history's, added in the release that introduced
// this line — precisely because the name that the copies actually used was not
// one this regex looked for. "Create it, then prove it is a real directory" is
// the same primitive whether the caller spells the predicate or the whole
// two-step; the two-step is the one that can be got wrong, so it is the one
// worth naming here.
var nonCanonicalPrimitiveRe = regexp.MustCompile(`func\s+(writeFileAtomic|durableWrite|isRealDir|ensureRealDir|createExclusiveIn)\b`)

// TestNoNonCanonicalAtomicWritePrimitives is the one-canonical-primitive
// detector: no package under internal/ (other than fsutil) may declare its own
// named atomic-write or real-dir primitive. It matches top-level function
// declarations by name (not inline temp+rename sequences), which is what the
// consolidation removes. It walks the internal/ tree from this package's
// directory; before the consolidation it flags four copies (ahoy
// marker.go/store.go, capture roots.go, memory writer.go).
func TestNoNonCanonicalAtomicWritePrimitives(t *testing.T) {
	internalRoot := filepath.Join("..") // internal/fsutil -> internal/
	var offenders []string
	err := filepath.WalkDir(internalRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// The canonical home is exempt; it defines the real thing.
			if filepath.Base(path) == "fsutil" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, m := range nonCanonicalPrimitiveRe.FindAllStringSubmatch(string(data), -1) {
			offenders = append(offenders, path+": func "+m[1])
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk internal/: %v", err)
	}
	if len(offenders) > 0 {
		t.Fatalf("non-canonical atomic-write/real-dir primitives (route through internal/fsutil):\n  %s",
			strings.Join(offenders, "\n  "))
	}
}

// TestNoInlineAtomicWriteSequences is the second half of the one-canonical-
// primitive detector (iss-79): the name-based check above cannot see a durable
// write open-coded inline rather than in a named func — exactly how memory's
// storeOriginal escaped the iss-32 consolidation. An inline atomic write is an
// exclusive temp create (os.O_EXCL) whose temp is then renamed onto the target
// (os.Rename); their co-occurrence in one non-fsutil file is the signature.
// (capture's allocator uses syscall.O_EXCL for a reservation with no rename, so
// it is correctly not matched — the os-package constant plus os.Rename is what
// marks a hand-rolled WriteFileAtomic.) The fix routes such a site through
// fsutil.WriteFileAtomic, which removes os.O_EXCL from the file.
//
// Two idioms are flagged (iss-82 broadened the check to the second): the strong
// os.O_EXCL temp + os.Rename form, and the weaker os.CreateTemp + os.Rename form
// (no fsync) that rules/inject.go SaveState open-coded. Either co-occurrence in
// one non-fsutil file is a hand-rolled durable write; route it through
// fsutil.WriteFileAtomic, which removes the temp+rename pair from the file.
func TestNoInlineAtomicWriteSequences(t *testing.T) {
	internalRoot := filepath.Join("..")
	var offenders []string
	err := filepath.WalkDir(internalRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if filepath.Base(path) == "fsutil" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		src := string(data)
		if strings.Contains(src, "os.O_EXCL") && strings.Contains(src, "os.Rename(") {
			offenders = append(offenders, path+": inline os.O_EXCL temp + os.Rename")
		}
		if strings.Contains(src, "os.CreateTemp(") && strings.Contains(src, "os.Rename(") {
			offenders = append(offenders, path+": inline os.CreateTemp + os.Rename")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk internal/: %v", err)
	}
	if len(offenders) > 0 {
		t.Fatalf("inline atomic-write sequences (route through fsutil.WriteFileAtomic):\n  %s",
			strings.Join(offenders, "\n  "))
	}
}
