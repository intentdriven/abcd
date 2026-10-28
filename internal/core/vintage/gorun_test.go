package vintage

import (
	"os"
	"path/filepath"
	"runtime/debug"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdrepo"
)

// TestGoRunModuleOf: only a development build of the module's cmd/abcd, with
// no revision stamped, run from a go-build work directory, is a go run build.
func TestGoRunModuleOf(t *testing.T) {
	const mod = "github.com/intentdriven/abcd"
	goRunExe := "/var/folders/xy/T/go-build1234/b001/exe/abcd"
	stamped := []debug.BuildSetting{{Key: "vcs.revision", Value: "0123456789abcdef0123456789abcdef01234567"}}
	for _, tc := range []struct {
		name                   string
		pkg, main, version, ex string
		settings               []debug.BuildSetting
		want                   bool
	}{
		{"go run of the checkout", mod + "/cmd/abcd", mod, "(devel)", goRunExe, nil, true},
		{"a built binary on disk", mod + "/cmd/abcd", mod, "(devel)", "/Users/x/.local/bin/abcd", nil, false},
		{"scripts/abcd's build, which CheckoutBuild reports", mod + "/cmd/abcd", mod, "(devel)", "/Users/x/src/abcd/" + abcdrepo.CheckoutBuildRel, nil, false},
		{"a stamped build", mod + "/cmd/abcd", mod, "(devel)", goRunExe, stamped, false},
		{"a released module version", mod + "/cmd/abcd", mod, "v0.14.0", goRunExe, nil, false},
		{"a go test binary", mod + "/internal/core/ahoy.test", mod, "(devel)", goRunExe, nil, false},
		{"another module's command", "example.com/other/cmd/abcd", "example.com/other", "(devel)", goRunExe, nil, true},
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

// TestCheckoutBuildOf: the build scripts/abcd has just made is an unstamped
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
