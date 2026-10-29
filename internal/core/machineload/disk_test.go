package machineload

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestNearestExisting: a directory not yet created is read on the volume of
// its nearest existing ancestor.
func TestNearestExisting(t *testing.T) {
	dir := t.TempDir()
	if got := NearestExisting(filepath.Join(dir, "worktrees", "abc", "lane")); got != dir {
		t.Fatalf("NearestExisting = %q, want %q", got, dir)
	}
	if got := NearestExisting(dir); got != dir {
		t.Fatalf("NearestExisting(existing) = %q, want %q", got, dir)
	}
}

// TestReadVolumeReadsThisMachine reads the real volume under a temporary
// directory, and one not yet created under it reads the same volume.
func TestReadVolumeReadsThisMachine(t *testing.T) {
	dir := t.TempDir()
	v, err := ReadVolume(dir)
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		if !errors.Is(err, ErrUnsupported) {
			t.Fatalf("err = %v, want ErrUnsupported", err)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if v.Total == 0 || v.Free > v.Total || v.ID == "" || v.Path != dir {
		t.Fatalf("volume = %+v", v)
	}
	if runtime.GOOS == "darwin" && v.Mount == "" {
		t.Fatalf("no mount point on macOS: %+v", v)
	}
	missing, err := ReadVolume(filepath.Join(dir, "not", "yet"))
	if err != nil || missing.ID != v.ID || missing.Path != dir {
		t.Fatalf("missing = %+v, %v; want volume %s at %s", missing, err, v.ID, dir)
	}
}

// cacheFixture lays out a Go build cache: n subdirectories, each holding one
// file of size bytes, plus the cache's own top-level files and a stray
// directory that is not one of its subdirectories.
func cacheFixture(t *testing.T, n int, size int) string {
	t.Helper()
	dir := t.TempDir()
	for i := range n {
		sub := filepath.Join(dir, strings.ToLower(hex2(i)))
		if err := os.Mkdir(sub, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(sub, "entry-d"), make([]byte, size), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []string{"README", "trim.txt"} {
		if err := os.WriteFile(filepath.Join(dir, f), make([]byte, 1000), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "fuzz", "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func hex2(i int) string {
	const digits = "0123456789abcdef"
	return string([]byte{digits[i>>4], digits[i&15]})
}

// TestGoCacheSizeExact: a cache walked in full within its budget is measured
// exactly, counting only its subdirectories' entries.
func TestGoCacheSizeExact(t *testing.T) {
	dir := cacheFixture(t, 256, 100)
	got, err := GoCacheSize(dir, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if got.Bytes != 25600 || got.Sampled != 256 || got.Subdirs != 256 || got.Estimated() {
		t.Fatalf("size = %+v, want 25600 bytes from all 256", got)
	}
}

// TestGoCacheSizeEstimates: a walk that runs out of budget scales what it
// walked by the subdirectories it did not, and always walks at least one.
func TestGoCacheSizeEstimates(t *testing.T) {
	dir := cacheFixture(t, 256, 100)
	t0 := time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC)
	calls := 0
	// The clock moves a second at each look: the deadline (start plus 3s)
	// passes after the third subdirectory.
	now := func() time.Time { calls++; return t0.Add(time.Duration(calls-1) * time.Second) }
	got, err := goCacheSize(dir, 3*time.Second, now)
	if err != nil {
		t.Fatal(err)
	}
	if got.Sampled != 3 || got.Subdirs != 256 || got.Bytes != 25600 || !got.Estimated() {
		t.Fatalf("size = %+v, want an estimate of 25600 from 3 of 256", got)
	}
	got, err = goCacheSize(dir, 0, now)
	if err != nil || got.Sampled != 1 || got.Bytes != 25600 {
		t.Fatalf("zero budget = %+v, %v; want one subdirectory walked", got, err)
	}
	if _, err := GoCacheSize(filepath.Join(dir, "absent"), time.Second); err == nil {
		t.Fatal("an absent cache measured")
	}
}

// TestGoCacheDir: `go env GOCACHE` wins; without a go command the variable,
// then go's own default; off, relative or absent is no cache.
func TestGoCacheDir(t *testing.T) {
	dir := t.TempDir()
	user := t.TempDir()
	if err := os.Mkdir(filepath.Join(user, "go-build"), 0o755); err != nil {
		t.Fatal(err)
	}
	noGo := func() (string, error) { return "", errors.New("go: not found") }
	env := func(v string) func(string) string { return func(string) string { return v } }
	userCache := func() (string, error) { return user, nil }
	for name, c := range map[string]struct {
		goEnv  func() (string, error)
		getenv func(string) string
		want   string
	}{
		"go env":             {func() (string, error) { return dir, nil }, env(""), dir},
		"variable":           {noGo, env(dir), dir},
		"default":            {noGo, env(""), filepath.Join(user, "go-build")},
		"off":                {func() (string, error) { return "off", nil }, env(""), ""},
		"relative":           {func() (string, error) { return "cache", nil }, env(""), ""},
		"absent":             {func() (string, error) { return filepath.Join(dir, "none"), nil }, env(""), ""},
		"go env empty falls": {func() (string, error) { return "", nil }, env(dir), dir},
	} {
		if got := goCacheDir(c.goEnv, c.getenv, userCache); got != c.want {
			t.Errorf("%s: goCacheDir = %q, want %q", name, got, c.want)
		}
	}
}
