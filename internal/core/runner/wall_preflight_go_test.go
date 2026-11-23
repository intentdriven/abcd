package runner

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// preflightGo makes a preflight's Go folder for p, removed when the test ends.
func preflightGo(t *testing.T, p Profile) string {
	t.Helper()
	dir, err := MakePreflightGo(p)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = RemovePreflightGo(p, dir) })
	return dir
}

// TestAPreflightsBuildCacheAndGOPATHAreItsOwnAndItsModuleCacheThePersons: a
// stage's shell is told the person's Go caches; a preflight is told the
// person's module cache, a build cache and GOPATH in a folder made for it
// alone, beside the stage's home and outside it, and no proxy it did not
// carry. Each preflight's folder is a new one, a link left where one stood is
// removed and never followed, and the folder is gone once removed.
func TestAPreflightsBuildCacheAndGOPATHAreItsOwnAndItsModuleCacheThePersons(t *testing.T) {
	p := wallProfile(t)
	stubGo(t, 0o755, `printf '%s\n' /caches/build /caches/mod /caches/path`)
	t.Setenv("GOPROXY", "")
	_ = os.Unsetenv("GOPROXY")
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
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "kept"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(p.StageRoot, preflightGoPrefix+"left")); err != nil {
		t.Fatal(err)
	}
	first, err := MakePreflightGo(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(p.StageRoot, preflightGoPrefix+"left")); !os.IsNotExist(err) {
		t.Errorf("a link left where a preflight's folder stood is not removed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outside, "kept")); err != nil {
		t.Errorf("the removal of a left link followed it: %v", err)
	}
	env := PreflightEnv(p, first)
	for name, want := range map[string]string{"GOCACHE": filepath.Join(first, "build"), "GOPATH": filepath.Join(first, "path"), "GOMODCACHE": "/caches/mod"} {
		if got, _ := envValue(env, name); got != want {
			t.Errorf("a preflight's %s is %q, want %q", name, got, want)
		}
	}
	if !fsutil.PathWithin(first, p.StageRoot, false) || fsutil.PathWithin(first, p.ShellHome, false) {
		t.Errorf("a preflight's folder %s is not beside the stage's home %s", first, p.ShellHome)
	}
	if got, ok := envValue(env, "GOPROXY"); ok {
		t.Errorf("a preflight is told a GOPROXY it did not carry: %q", got)
	}
	if err := RemovePreflightGo(p, first); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(first); !os.IsNotExist(err) {
		t.Errorf("a removed preflight's folder stands: %v", err)
	}
	second := preflightGo(t, p)
	if second == first {
		t.Errorf("the next preflight's folder is the last one's: %s", second)
	}
	if err := RemovePreflightGo(p, filepath.Join(p.StageRoot, "home")); err == nil {
		t.Error("a folder that is not a preflight's is removed")
	}
}

// writeModuleProxy lays out, under proxy, a Go module proxy serving the module
// path at v1.0.0 with the files given (go.mod among them).
func writeModuleProxy(t *testing.T, proxy, path string, files map[string]string, withZip bool) {
	t.Helper()
	dir := filepath.Join(proxy, filepath.FromSlash(path), "@v")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"list":        "v1.0.0\n",
		"v1.0.0.info": `{"Version":"v1.0.0","Time":"2026-01-01T00:00:00Z"}`,
		"v1.0.0.mod":  files["go.mod"],
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if !withZip {
		return
	}
	f, err := os.Create(filepath.Join(dir, "v1.0.0.zip"))
	if err != nil {
		t.Fatal(err)
	}
	z := zip.NewWriter(f)
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		w, err := z.Create(path + "@v1.0.0/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(files[name])); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

// goModHash is the go.sum hash of a module's go.mod holding body: Go's h1, the
// SHA-256 of the file's own SHA-256 line.
func goModHash(body string) string {
	line := fmt.Sprintf("%x  go.mod\n", sha256.Sum256([]byte(body)))
	sum := sha256.Sum256([]byte(line))
	return "h1:" + base64.StdEncoding.EncodeToString(sum[:])
}

// goIn runs the real go in dir under env, failing the test on an error.
func goIn(t *testing.T, dir string, env []string, args ...string) string {
	t.Helper()
	cmd := exec.Command("go", args...)
	cmd.Dir, cmd.Env = dir, env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go %s in %s: %v\n%s", strings.Join(args, " "), dir, err, out)
	}
	return string(out)
}

// TestAPreflightBuildsFromThePersonsModuleCacheWithNoNetwork: under a
// preflight's environment, with the proxy it carries unreachable, the real go
// builds a module whose requirements the person's module cache holds: one a
// carried GOPRIVATE covers, which Go would otherwise fetch from its origin
// directly, and one go.sum does not list. `go mod verify` passes over that
// cache with no network and leaves go.mod and go.sum as they were; a module
// directory changed since Go unpacked it refuses the preflight, naming it; and
// a module the cache lacks altogether is left unverified, its metadata fetched
// through the proxy carried, while the modules the cache holds are still
// verified.
func TestAPreflightBuildsFromThePersonsModuleCacheWithNoNetwork(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("no go")
	}
	work := t.TempDir()
	proxy := filepath.Join(work, "proxy")
	writeModuleProxy(t, proxy, "private.invalid/mod", map[string]string{
		"go.mod": "module private.invalid/mod\n\ngo 1.21\n",
		"p.go":   "package mod\n\nconst Word = \"private\"\n",
	}, true)
	writeModuleProxy(t, proxy, "notsummed.invalid/lib", map[string]string{
		"go.mod": "module notsummed.invalid/lib\n\ngo 1.21\n",
		"l.go":   "package lib\n\nconst Word = \"unsummed\"\n",
	}, true)
	modcache := filepath.Join(work, "modcache")
	module := filepath.Join(work, "module")
	if err := os.MkdirAll(module, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(module, "go.mod"), []byte("module example.com/fixture\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// The person's module cache, filled from the local proxy alone.
	seed := append(os.Environ(), "GOMODCACHE="+modcache, "GOCACHE="+filepath.Join(work, "seedbuild"), "GOPATH="+filepath.Join(work, "seedpath"),
		"GOPROXY=file://"+filepath.ToSlash(proxy), "GONOSUMDB=*.invalid", "GOPRIVATE=", "GOFLAGS=-modcacherw", "GOTOOLCHAIN=local", "GOWORK=off", "GOENV=off", "GOTELEMETRY=off")
	goIn(t, module, seed, "get", "private.invalid/mod@v1.0.0")
	goIn(t, module, seed, "mod", "download", "notsummed.invalid/lib@v1.0.0")
	if err := os.WriteFile(filepath.Join(module, "go.mod"), []byte("module example.com/fixture\n\ngo 1.21\n\nrequire (\n\tnotsummed.invalid/lib v1.0.0\n\tprivate.invalid/mod v1.0.0\n)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(module, "main.go"), []byte("package main\n\nimport (\n\t\"notsummed.invalid/lib\"\n\t\"private.invalid/mod\"\n)\n\nfunc main() { println(mod.Word, lib.Word) }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sum, err := os.ReadFile(filepath.Join(module, "go.sum"))
	if err != nil || strings.Contains(string(sum), "notsummed.invalid") || !strings.Contains(string(sum), "private.invalid/mod v1.0.0 h1:") {
		t.Fatalf("premise: go.sum lists the private module and not the other: %v\n%s", err, sum)
	}
	gomod, err := os.ReadFile(filepath.Join(module, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}

	// The person's environment, as a drive's stage would carry it.
	t.Setenv("GOMODCACHE", modcache)
	t.Setenv("GOPROXY", "http://127.0.0.1:9")
	t.Setenv("GOPRIVATE", "private.invalid")
	t.Setenv("GONOSUMDB", "notsummed.invalid")
	t.Setenv("GOFLAGS", "")
	t.Setenv("GOTOOLCHAIN", "local")
	admittedGo(t)
	p := wallProfile(t)
	if _, err := WriteWallSettings(p, wallBin); err != nil {
		t.Fatal(err)
	}
	dir := preflightGo(t, p)
	env := append(PreflightEnv(p, dir), "GOWORK=off", "GOENV=off")
	if got, _ := envValue(env, "GOMODCACHE"); got != modcache {
		t.Fatalf("a preflight's module cache is %q, want the person's %q", got, modcache)
	}
	// The Go there starts its telemetry under the stage's home, which would race
	// this test's cleanup.
	goIn(t, module, env, "telemetry", "off")

	if _, err := VerifyModules(context.Background(), module, env); err != nil {
		t.Fatalf("the person's module cache, unchanged, does not verify: %v", err)
	}
	for name, was := range map[string][]byte{"go.mod": gomod, "go.sum": sum} {
		if now, err := os.ReadFile(filepath.Join(module, name)); err != nil || string(now) != string(was) {
			t.Fatalf("`go mod verify` changed %s: %v\n%s", name, err, now)
		}
	}
	goIn(t, module, withEnv(env, "GOFLAGS=-mod=mod"), "build", "-o", filepath.Join(work, "fixture"), ".")
	if err := os.WriteFile(filepath.Join(module, "go.sum"), sum, 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("a module the cache lacks", func(t *testing.T) {
		writeModuleProxy(t, proxy, "absent.invalid/gone", map[string]string{"go.mod": "module absent.invalid/gone\n\ngo 1.21\n"}, false)
		lacking := filepath.Join(work, "lacking")
		if err := os.MkdirAll(lacking, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(lacking, "go.mod"), []byte("module example.com/lacking\n\ngo 1.21\n\nrequire (\n\tabsent.invalid/gone v1.0.0\n\tprivate.invalid/mod v1.0.0\n)\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		goneMod := "module absent.invalid/gone\n\ngo 1.21\n"
		if err := os.WriteFile(filepath.Join(lacking, "go.sum"), append(slices.Clone(sum), []byte("absent.invalid/gone v1.0.0/go.mod "+goModHash(goneMod)+"\n")...), 0o644); err != nil {
			t.Fatal(err)
		}
		var mce *ModuleCacheError
		if _, err := VerifyModules(context.Background(), lacking, env); !errors.As(err, &mce) || len(mce.Modified) != 0 || mce.Changed != "" {
			t.Fatalf("a module neither the cache nor the proxy carried can serve refuses in go's words alone: %v", err)
		}
		reach := withEnv(env, "GOPROXY=file://"+filepath.ToSlash(proxy), "GONOSUMDB=*.invalid")
		if _, err := VerifyModules(context.Background(), lacking, reach); err != nil {
			t.Fatalf("a module the cache lacks, its metadata at the proxy carried, refuses: %v", err)
		}
		if _, err := os.Stat(filepath.Join(modcache, "absent.invalid", "gone@v1.0.0")); !os.IsNotExist(err) {
			t.Fatalf("the verification unpacked the module the cache lacked: %v", err)
		}
	})

	t.Run("a changed module directory", func(t *testing.T) {
		changed := filepath.Join(modcache, "private.invalid", "mod@v1.0.0", "p.go")
		if err := os.Chmod(changed, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(changed, []byte("package mod\n\nconst Word = \"changed\"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		for name, e := range map[string][]string{"offline": env, "its metadata fetched": withEnv(env, "GOPROXY=file://"+filepath.ToSlash(proxy), "GONOSUMDB=*.invalid")} {
			dir := module
			if name != "offline" {
				dir = filepath.Join(work, "lacking")
			}
			var mce *ModuleCacheError
			_, err := VerifyModules(context.Background(), dir, e)
			if !errors.As(err, &mce) || !slices.Contains(mce.Modified, "private.invalid/mod@v1.0.0") || !strings.Contains(err.Error(), "private.invalid/mod@v1.0.0") {
				t.Errorf("%s: a changed module directory does not refuse the preflight naming it: %v", name, err)
			}
		}
		if now, err := os.ReadFile(filepath.Join(module, "go.sum")); err != nil || string(now) != string(sum) {
			t.Fatalf("a refused verification changed go.sum: %v", err)
		}
	})
}

// TestAVerificationThatWritesTheModuleFilesIsRefusedAndPutBack: a go that
// rewrites go.sum while it verifies refuses the preflight, and go.sum is put
// back as it was; a directory with no go.mod is not verified.
func TestAVerificationThatWritesTheModuleFilesIsRefusedAndPutBack(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.sum"), []byte("as it was\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stubGo(t, 0o755, `printf 'rewritten\n' > go.sum; printf 'go.work\n' > go.work; echo 'all modules verified'`)
	var mce *ModuleCacheError
	if _, err := VerifyModules(context.Background(), dir, os.Environ()); !errors.As(err, &mce) || mce.Changed == "" {
		t.Fatalf("a verification that wrote the module files is not refused: %v", err)
	}
	if raw, err := os.ReadFile(filepath.Join(dir, "go.sum")); err != nil || string(raw) != "as it was\n" {
		t.Fatalf("go.sum is not put back: %q %v", raw, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "go.work")); !os.IsNotExist(err) {
		t.Fatalf("a go.work the verification wrote stands: %v", err)
	}
	if _, err := VerifyModules(context.Background(), t.TempDir(), os.Environ()); err != nil {
		t.Fatalf("a directory with no go.mod is verified: %v", err)
	}
}

// seededModule lays out, in a directory of its own, a module requiring
// private.invalid/mod with its go.sum, and a module cache holding that module,
// filled from a local proxy alone; it returns the module, the cache, and the
// environment a verification runs under, offline (GOPROXY=off) and in module
// mode (GOWORK=off).
func seededModule(t *testing.T) (module, modcache string, env []string) {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("no go")
	}
	work := t.TempDir()
	proxy := filepath.Join(work, "proxy")
	writeModuleProxy(t, proxy, "private.invalid/mod", map[string]string{
		"go.mod": "module private.invalid/mod\n\ngo 1.21\n",
		"p.go":   "package mod\n\nconst Word = \"private\"\n",
	}, true)
	module, modcache = filepath.Join(work, "module"), filepath.Join(work, "modcache")
	if err := os.MkdirAll(module, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"go.mod":  "module example.com/fixture\n\ngo 1.21\n",
		"main.go": "package main\n\nimport \"private.invalid/mod\"\n\nfunc main() { println(mod.Word) }\n",
	} {
		if err := os.WriteFile(filepath.Join(module, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	env = withEnv(os.Environ(), "GOMODCACHE="+modcache, "GOCACHE="+filepath.Join(work, "build"), "GOPATH="+filepath.Join(work, "path"),
		"GOPROXY=off", "GONOSUMDB=*.invalid", "GOPRIVATE=", "GOFLAGS=", "GOTOOLCHAIN=local", "GOWORK=off", "GOENV=off", "GOTELEMETRY=off")
	goIn(t, module, withEnv(env, "GOPROXY=file://"+filepath.ToSlash(proxy), "GOFLAGS=-modcacherw"), "get", "private.invalid/mod@v1.0.0")
	return module, modcache, env
}

// withoutEnv is env with no entry for key.
func withoutEnv(env []string, key string) []string {
	return slices.DeleteFunc(slices.Clone(env), func(kv string) bool { return strings.HasPrefix(kv, key+"=") })
}

// TestAVendoredBuildIsNotVerifiedAgainstTheModuleCache: a module whose build
// reads vendor/ (vendor/modules.txt, a go directive at 1.14 or later, and no
// -mod in GOFLAGS, or -mod=vendor) builds with an empty module cache and no
// network; its preflight is not stopped for the cache, and the note says the
// module cache was not verified. A GOFLAGS of -mod=mod reads the cache again,
// and is verified.
func TestAVendoredBuildIsNotVerifiedAgainstTheModuleCache(t *testing.T) {
	module, _, env := seededModule(t)
	goIn(t, module, env, "mod", "vendor")
	empty := withEnv(env, "GOMODCACHE="+filepath.Join(t.TempDir(), "empty"))
	for _, proxy := range []string{"off", "http://127.0.0.1:9"} {
		for _, flags := range []string{"", "-mod=vendor"} {
			e := withEnv(empty, "GOPROXY="+proxy, "GOFLAGS="+flags)
			goIn(t, module, e, "list", "-deps", ".")
			note, err := VerifyModules(context.Background(), module, e)
			if err != nil || !strings.Contains(note, "vendor") {
				t.Errorf("GOPROXY=%s GOFLAGS=%q: a vendored build is stopped for the module cache, or not said to be passed over: %q %v", proxy, flags, note, err)
			}
		}
	}
	var mce *ModuleCacheError
	if _, err := VerifyModules(context.Background(), module, withEnv(empty, "GOFLAGS=-mod=mod")); !errors.As(err, &mce) {
		t.Errorf("a build told -mod=mod reads the module cache, and is verified: %v", err)
	}
}

// TestAWorkspaceIsVerifiedWithoutAModFlag: with a go.work in force, the
// verification runs in workspace mode, which takes no -mod=mod, and still
// names a changed module; a checkout with no go.mod at its root has nothing
// verified, and the note says so.
func TestAWorkspaceIsVerifiedWithoutAModFlag(t *testing.T) {
	module, modcache, env := seededModule(t)
	if err := os.WriteFile(filepath.Join(module, "go.work"), []byte("go 1.21\n\nuse .\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ws := withoutEnv(env, "GOWORK")
	if note, err := VerifyModules(context.Background(), module, ws); err != nil {
		t.Fatalf("an unchanged workspace does not verify: %q %v", note, err)
	}
	changed := filepath.Join(modcache, "private.invalid", "mod@v1.0.0", "p.go")
	if err := os.Chmod(changed, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(changed, []byte("package mod\n\nconst Word = \"changed\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var mce *ModuleCacheError
	if _, err := VerifyModules(context.Background(), module, ws); !errors.As(err, &mce) || !slices.Contains(mce.Modified, "private.invalid/mod@v1.0.0") {
		t.Fatalf("a changed module in a workspace is not named: %v", err)
	}

	// What go is told: the person's GOFLAGS less any -mod in workspace mode,
	// -mod=mod in module mode.
	flags := filepath.Join(t.TempDir(), "flags")
	stubGo(t, 0o755, `if [ "$1" = env ]; then if [ "$GOWORK" = off ]; then echo off; else echo `+module+`/go.work; fi; exit 0; fi; printf '%s\n' "$GOFLAGS" >> `+flags+`; echo 'all modules verified'`)
	for _, e := range [][]string{withEnv(ws, "GOFLAGS=-mod=mod -p=1"), env} {
		if _, err := VerifyModules(context.Background(), module, e); err != nil {
			t.Fatal(err)
		}
	}
	if raw, _ := os.ReadFile(flags); string(raw) != "-p=1\n-mod=mod\n" {
		t.Errorf("go mod verify was told GOFLAGS %q, want no -mod in workspace mode and -mod=mod in module mode", raw)
	}

	none := t.TempDir()
	if err := os.WriteFile(filepath.Join(none, "go.work"), []byte("go 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if note, err := VerifyModules(context.Background(), none, env); err != nil || !strings.Contains(note, "no go.mod") {
		t.Errorf("a checkout with no root go.mod is not said to have nothing verified: %q %v", note, err)
	}
}
