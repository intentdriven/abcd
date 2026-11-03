package launch

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/gittest"
)

// thisRepoRender is one render of this repository's own payload, shared by the
// tests that only read what it wrote. The render is a pure function of the
// tree and its request, and no test in this package writes to this
// repository, so each of those tests saw the same payload; under -race every
// render of this repository cost about 45 seconds, nearly all of it the
// payload's secret scan, so the scan the render's pre-check made of the
// payload is kept as well, for the test that reads the scan itself. The
// committed manifests are read before the render,
// so a test that holds the render to leaving them unchanged compares against
// the bytes as they stood before it ran, whichever test ran it.
var thisRepoRender struct {
	once   sync.Once
	dir    string // the scratch directory holding the payload; TestMain removes it
	dest   string
	before map[string][]byte
	scan   scanner.ScanResult
	err    error
}

// thisRepoManifests are the committed manifests adr-19 keeps version-absent.
var thisRepoManifests = []string{".claude-plugin/plugin.json", ".claude-plugin/marketplace.json"}

// renderThisRepo returns that shared render's payload directory and the
// committed manifests' bytes read before it was made.
func renderThisRepo(t *testing.T) (dest string, before map[string][]byte) {
	t.Helper()
	root := repoRootForTest(t)
	r := &thisRepoRender
	r.once.Do(func() {
		r.before = make(map[string][]byte, len(thisRepoManifests))
		for _, rel := range thisRepoManifests {
			data, err := os.ReadFile(filepath.Join(root, rel))
			if err != nil {
				r.err = fmt.Errorf("read %s: %w", rel, err)
				return
			}
			r.before[rel] = data
		}
		if r.dir, r.err = os.MkdirTemp("", "abcd-launch-render-"); r.err != nil {
			return
		}
		r.dest = filepath.Join(r.dir, "payload")
		// The render scans exactly once: the pre-check's scan of the bundle
		// ResolveBundle(root, nil) returns, the input the scan test reads.
		scans, prev := 0, scanBundle
		scanBundle = func(repoRoot string, bundle Bundle) scanner.ScanResult {
			scans++
			r.scan = prev(repoRoot, bundle)
			return r.scan
		}
		defer func() { scanBundle = prev }()
		_, r.err = RenderPayload(PayloadRenderRequest{
			RepoRoot: root, Dest: r.dest, Version: "9.9.9", Entry: sampleEntry(), Dirty: DirtySkip,
		})
		if r.err == nil && scans != 1 {
			r.err = fmt.Errorf("the render scanned its payload %d times, want once", scans)
		}
	})
	if r.err != nil {
		t.Fatalf("render this repository's payload: %v", r.err)
	}
	return r.dest, r.before
}

// scanThisRepo returns the scan that shared render's pre-check made of this
// repository's payload.
func scanThisRepo(t *testing.T) scanner.ScanResult {
	t.Helper()
	renderThisRepo(t)
	return thisRepoRender.scan
}

// TestMain runs the package as a neutral caller (gittest.NeutralCaller): its
// scans judge the payload, so they must not read whoever runs the suite into
// it. Under an account named for a word the payload holds, a cloud
// container's `abcd`, every scan of a clean tree hard-failed on that word as
// the caller's username. A test that sets HOME itself still does so.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "launch-caller-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "launch: a neutral caller's home:", err)
		os.Exit(1)
	}
	home, err := gittest.NeutralHomeIn(dir)
	if err == nil {
		err = os.Setenv("HOME", home)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "launch: a neutral caller's home:", err)
		_ = os.RemoveAll(dir)
		os.Exit(1)
	}
	code := m.Run()
	if thisRepoRender.dir != "" {
		_ = os.RemoveAll(thisRepoRender.dir)
	}
	_ = os.RemoveAll(dir)
	os.Exit(code)
}
