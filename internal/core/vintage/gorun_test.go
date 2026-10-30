package vintage

import (
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdrepo"
)

// TestGoRunModuleOf: only a development build of the module's cmd/abcd, with
// no revision stamped, run from where go run runs it (its work directory's
// link, or the build cache's copy), is a go run build.
func TestGoRunModuleOf(t *testing.T) {
	const mod = "github.com/intentdriven/abcd"
	goRunExe := "/var/folders/xy/T/go-build1234/b001/exe/abcd"
	cacheID := "4e99" + strings.Repeat("0123456789abcdef", 3) + "0123456789ab"
	stamped := []debug.BuildSetting{{Key: "vcs.revision", Value: "0123456789abcdef0123456789abcdef01234567"}}
	for _, tc := range []struct {
		name                   string
		pkg, main, version, ex string
		settings               []debug.BuildSetting
		want                   bool
	}{
		{"go run of the checkout", mod + "/cmd/abcd", mod, "(devel)", goRunExe, nil, true},
		{"a built binary on disk", mod + "/cmd/abcd", mod, "(devel)", "/Users/x/.local/bin/abcd", nil, false},
		{"scripts/abcd.sh's build, which CheckoutBuild reports", mod + "/cmd/abcd", mod, "(devel)", "/Users/x/src/abcd/" + abcdrepo.CheckoutBuildRel, nil, false},
		{"a stamped build", mod + "/cmd/abcd", mod, "(devel)", goRunExe, stamped, false},
		{"a released module version", mod + "/cmd/abcd", mod, "v0.14.0", goRunExe, nil, false},
		{"a go test binary", mod + "/internal/core/ahoy.test", mod, "(devel)", goRunExe, nil, false},
		{"another module's command", "example.com/other/cmd/abcd", "example.com/other", "(devel)", goRunExe, nil, true},
		{"go run under GOTMPDIR", mod + "/cmd/abcd", mod, "(devel)", "/Users/x/gotmp/go-build98/b042/exe/abcd", nil, true},
		{"go run's cached executable, on a cache hit", mod + "/cmd/abcd", mod, "(devel)", "/Users/x/Library/Caches/go-build/4e/" + cacheID + "-d/abcd", nil, true},
		{"go run's cached executable under a GOCACHE of its own", mod + "/cmd/abcd", mod, "(devel)", "/srv/cache/4e/" + cacheID + "-d/abcd", nil, true},
		{"a cache-shaped path with a short id", mod + "/cmd/abcd", mod, "(devel)", "/Users/x/Library/Caches/go-build/4e/4e99-d/abcd", nil, false},
		{"a binary beside the cache entries, not in one", mod + "/cmd/abcd", mod, "(devel)", "/Users/x/Library/Caches/go-build/4e/abcd", nil, false},
		{"a local-tier binary in a checkout named go-build...", mod + "/cmd/abcd", mod, "(devel)", "/Users/x/wt/go-build-cache/" + abcdrepo.CheckoutBuildRel, nil, false},
		{"a binary in a go-build directory but not where go run links", mod + "/cmd/abcd", mod, "(devel)", "/var/folders/xy/T/go-build1234/abcd", nil, false},
		{"a binary in a directory only named like a work directory", mod + "/cmd/abcd", mod, "(devel)", "/Users/x/go-build-tools/b001/exe/abcd", nil, false},
	} {
		got, ok := goRunModuleOf(tc.pkg, tc.main, tc.version, tc.settings, tc.ex)
		if ok != tc.want {
			t.Errorf("%s: ok = %v, want %v", tc.name, ok, tc.want)
		}
		if ok && got != tc.main {
			t.Errorf("%s: module = %q, want %q", tc.name, got, tc.main)
		}
	}
}

// TestTheMarkIsTakenOutOfTheEnvironment: the mark vouches only for the binary
// the script ran, so the binary reads it and removes it, and nothing it starts
// can inherit it (iss-2610241240133009).
func TestTheMarkIsTakenOutOfTheEnvironment(t *testing.T) {
	t.Setenv(abcdrepo.CheckoutBuildEnv, "/x/"+abcdrepo.CheckoutBuildRel)
	if got := takeEnv(abcdrepo.CheckoutBuildEnv); got != "/x/"+abcdrepo.CheckoutBuildRel {
		t.Errorf("takeEnv = %q, want the mark", got)
	}
	if v, set := os.LookupEnv(abcdrepo.CheckoutBuildEnv); set {
		t.Errorf("the mark is still in the environment a child would inherit: %q", v)
	}
}

// TestCheckoutBuildOf: the build scripts/abcd.sh has just made is an unstamped
// development build of the module's cmd/abcd at a checkout's local tier, and
// only while the script's mark names that very file. A copy left there and run
// directly carries no mark, so it is not one (iss-2610241240133009).
func TestCheckoutBuildOf(t *testing.T) {
	const mod = "github.com/intentdriven/abcd"
	binary := func(rel string) string {
		p := filepath.Join(t.TempDir(), filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("binary"), 0o755); err != nil {
			t.Fatal(err)
		}
		return p
	}
	exe := binary(abcdrepo.CheckoutBuildRel)
	other := binary(abcdrepo.CheckoutBuildRel)
	link := filepath.Join(t.TempDir(), "abcd")
	if err := os.Symlink(exe, link); err != nil {
		t.Fatal(err)
	}
	elsewhere := binary("bin/abcd")
	stamped := []debug.BuildSetting{{Key: "vcs.revision", Value: "0123456789abcdef0123456789abcdef01234567"}}
	for _, tc := range []struct {
		name                      string
		pkg, version, exe, marked string
		settings                  []debug.BuildSetting
		want                      bool
	}{
		{"the script's run of its fresh build", mod + "/cmd/abcd", "(devel)", exe, exe, nil, true},
		{"the mark spelled through a symlink", mod + "/cmd/abcd", "(devel)", exe, link, nil, true},
		{"a copy run directly, with no mark", mod + "/cmd/abcd", "(devel)", exe, "", nil, false},
		{"a mark naming another checkout's build", mod + "/cmd/abcd", "(devel)", exe, other, nil, false},
		{"a mark naming a file that does not exist", mod + "/cmd/abcd", "(devel)", exe, exe + ".gone", nil, false},
		{"a marked binary outside the local tier", mod + "/cmd/abcd", "(devel)", elsewhere, elsewhere, nil, false},
		{"a stamped build in the local tier", mod + "/cmd/abcd", "(devel)", exe, exe, stamped, false},
		{"a release-versioned build in the local tier", mod + "/cmd/abcd", "v0.15.2-0.20261027120000-0123456789ab", exe, exe, nil, false},
		{"a go test binary", mod + "/internal/core/vintage.test", "(devel)", exe, exe, nil, false},
	} {
		got, ok := checkoutBuildOf(tc.pkg, mod, tc.version, tc.settings, tc.exe, tc.marked)
		if ok != tc.want {
			t.Errorf("%s: ok = %v, want %v", tc.name, ok, tc.want)
		}
		if ok && got != mod {
			t.Errorf("%s: module = %q, want %q", tc.name, got, mod)
		}
	}
}
