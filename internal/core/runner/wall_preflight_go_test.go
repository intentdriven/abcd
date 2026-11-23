package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// TestAPreflightsGoCachesAreItsOwn: a stage's shell is told the person's Go
// caches, and a preflight is told none of them: its build cache, module cache
// and GOPATH lie beside the stage's files, outside the stage's home, and a
// module reaches its module cache through the person's module downloads read
// as a proxy, ahead of the proxy it would use otherwise.
func TestAPreflightsGoCachesAreItsOwn(t *testing.T) {
	for name, carried := range map[string]struct{ set, want string }{
		"no proxy of its own": {want: "https://proxy.golang.org,direct"},
		"a proxy of its own":  {set: "https://proxy.example.invalid,off", want: "https://proxy.example.invalid,off"},
	} {
		t.Run(name, func(t *testing.T) {
			p := wallProfile(t)
			stubGo(t, 0o755, `printf '%s\n' /caches/build /caches/mod /caches/path`)
			if carried.set == "" {
				t.Setenv("GOPROXY", "")
				_ = os.Unsetenv("GOPROXY")
			} else {
				t.Setenv("GOPROXY", carried.set)
			}
			if _, err := WriteWallSettings(p, wallBin); err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(p.EnvFile)
			if err != nil {
				t.Fatal(err)
			}
			for _, kv := range []string{"GOCACHE='/caches/build'", "GOMODCACHE='/caches/mod'", "GOPATH='/caches/path'"} {
				if !strings.Contains(string(raw), "export "+kv+"\n") {
					t.Errorf("a stage's shell is not told %s:\n%s", kv, raw)
				}
			}
			env := PreflightEnv(p)
			for _, name := range []string{"GOCACHE", "GOMODCACHE", "GOPATH"} {
				got, ok := envValue(env, name)
				if !ok || !filepath.IsAbs(got) || strings.HasPrefix(got, "/caches/") ||
					!fsutil.PathWithin(got, p.StageRoot, false) || fsutil.PathWithin(got, p.ShellHome, false) {
					t.Errorf("a preflight's %s is %q, want its own beside the stage's files %s and outside its home", name, got, p.StageRoot)
				}
			}
			proxy, _ := envValue(env, "GOPROXY")
			if want := "file:///caches/mod/cache/download," + carried.want; proxy != want {
				t.Errorf("a preflight's GOPROXY is %q, want %q", proxy, want)
			}
		})
	}
}

// TestAPreflightWithNoModuleCacheNamedKeepsItsProxy: where the person's Go names
// no module cache, a preflight still keeps caches of its own and the proxy it
// carried, with no download directory read ahead of it.
func TestAPreflightWithNoModuleCacheNamedKeepsItsProxy(t *testing.T) {
	p := wallProfile(t)
	stubGo(t, 0o755, `printf '%s\n' /caches/build off /caches/path`)
	t.Setenv("GOPROXY", "https://proxy.example.invalid")
	if _, err := WriteWallSettings(p, wallBin); err != nil {
		t.Fatal(err)
	}
	env := PreflightEnv(p)
	if got, _ := envValue(env, "GOMODCACHE"); !fsutil.PathWithin(got, p.StageRoot, false) {
		t.Errorf("a preflight's module cache is %q, want its own", got)
	}
	if got, _ := envValue(env, "GOPROXY"); got != "https://proxy.example.invalid" {
		t.Errorf("a preflight's GOPROXY is %q, want the one it carried", got)
	}
}
