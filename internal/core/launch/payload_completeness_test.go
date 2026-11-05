package launch

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// pluginSurfaceDirs are the directory names an agent harness auto-discovers as
// plugin surfaces. Each one present on disk must be either shipped (named in the
// launch-payload includes) or deliberately withheld (surfaceExcludeReasons), so
// a newly-added surface directory can never be silently dropped from the
// published bundle (iss-77: agents/ and hooks/ were both).
var pluginSurfaceDirs = []string{"commands", "agents", "hooks", "skills"}

// surfaceExcludeReasons names any plugin-surface directory that is deliberately
// NOT shipped, each with its reason. A surface present on disk that is neither in
// the payload includes nor here fails TestBundleShipsEveryPluginSurface — forcing
// an explicit include-or-exclude decision rather than a silent drop.
var surfaceExcludeReasons = map[string]string{}

// repoRootForTest returns the abcd-cli repo root, found by walking up from the
// test's working directory (its package directory) to go.mod, so the
// completeness check runs against the committed launch-payload config and the
// real surface directories rather than a synthetic fixture.
func repoRootForTest(t *testing.T) string {
	t.Helper()
	root, err := fsutil.ModuleRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// TestBundleShipsEveryPluginSurface is the bundle-completeness detector: every
// auto-discovered plugin-surface directory present on disk must be covered by the
// launch-payload includes or explicitly excluded with a reason, so the published
// plugin can never ship missing one of its surfaces (iss-77).
func TestBundleShipsEveryPluginSurface(t *testing.T) {
	root := repoRootForTest(t)
	includes, err := LoadIncludes(root)
	if err != nil {
		t.Fatalf("load committed launch-payload includes: %v", err)
	}
	included := make(map[string]struct{}, len(includes))
	for _, inc := range includes {
		included[inc] = struct{}{}
	}
	for _, dir := range pluginSurfaceDirs {
		info, err := os.Stat(filepath.Join(root, dir))
		if err != nil || !info.IsDir() {
			continue // not present on disk → nothing to ship for this surface
		}
		if _, ok := included[dir]; ok {
			continue
		}
		if reason, ok := surfaceExcludeReasons[dir]; ok {
			if reason == "" {
				t.Errorf("plugin surface %q is excluded but its reason is empty", dir)
			}
			continue
		}
		t.Errorf("plugin-surface directory %q exists on disk but is neither in the launch-payload includes nor in surfaceExcludeReasons — the published plugin would silently ship without it", dir)
	}
}
