package vintage

import (
	"runtime/debug"
	"testing"
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
