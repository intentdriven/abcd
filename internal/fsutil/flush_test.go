package fsutil

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// TestSkipFlushGateTruthTable pins the gate: the flush is skipped only when the
// binary is a test binary AND the opt-in is exactly "1". Neither half alone
// disables durability.
func TestSkipFlushGateTruthTable(t *testing.T) {
	for _, c := range []struct {
		testBinary bool
		optIn      string
		want       bool
	}{
		{false, "", false},
		{false, "1", false},
		{false, "true", false},
		{true, "", false},
		{true, "0", false},
		{true, "true", false},
		{true, " 1", false},
		{true, "1", true},
	} {
		if got := skipFlush(c.testBinary, c.optIn); got != c.want {
			t.Errorf("skipFlush(testBinary=%v, %s=%q) = %v, want %v", c.testBinary, SkipFlushEnv, c.optIn, got, c.want)
		}
	}
}

// countFlushes swaps the flush seam for a counter for the rest of the test.
func countFlushes(t *testing.T) *int {
	t.Helper()
	n := 0
	orig := syncFile
	syncFile = func(f *os.File) error {
		n++
		return orig(f)
	}
	t.Cleanup(func() { syncFile = orig })
	return &n
}

// TestEveryDurableWriteFlushesUnlessTheTestOptedIn is the durability test the
// skip must not reach: with the opt-in cleared explicitly, every durable write
// flushes the file and its parent directory; with it set, this test binary
// skips both.
func TestEveryDurableWriteFlushesUnlessTheTestOptedIn(t *testing.T) {
	writes := map[string]func(t *testing.T, dir string) error{
		"WriteFileAtomic": func(t *testing.T, dir string) error {
			return WriteFileAtomic(filepath.Join(dir, "a"), []byte("x"), 0o644)
		},
		"WriteFileAtomicInRoot": func(t *testing.T, dir string) error {
			root, err := os.OpenRoot(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			return WriteFileAtomicInRoot(root, "a", []byte("x"), 0o644)
		},
		"CreateExclusiveIn": func(t *testing.T, dir string) error {
			root, err := os.OpenRoot(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			return CreateExclusiveIn(root, "a", []byte("x"), 0o644)
		},
	}
	for name, write := range writes {
		for _, c := range []struct {
			optIn string
			want  int
		}{{"", 2}, {"1", 0}} {
			t.Run(name+"/"+SkipFlushEnv+"="+c.optIn, func(t *testing.T) {
				t.Setenv(SkipFlushEnv, c.optIn)
				n := countFlushes(t)
				if err := write(t, t.TempDir()); err != nil {
					t.Fatalf("%s: %v", name, err)
				}
				if *n != c.want {
					t.Errorf("%s with %s=%q flushed %d time(s), want %d (the file and its parent directory, or none)",
						name, SkipFlushEnv, c.optIn, *n, c.want)
				}
			})
		}
	}
}

// flushOutcome reports whether Flush reached the flush, observed by its error:
// a flush of a closed file fails with os.ErrClosed, a skipped one returns nil.
func flushOutcome(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "flush-*")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	switch err := Flush(f); {
	case errors.Is(err, os.ErrClosed):
		return "flushed"
	case err == nil:
		return "skipped"
	default:
		t.Fatalf("Flush: %v", err)
		return ""
	}
}

// TestAShippedBinaryFlushesWithTheOptInSet proves the opt-in cannot reach a
// shipped binary: the probe under testdata is built with go build — a binary
// like the released one, not a test binary — and run with the opt-in set, and
// it still flushes. The same observation inside this test binary skips, so the
// difference is testing.Testing and nothing else.
func TestAShippedBinaryFlushesWithTheOptInSet(t *testing.T) {
	t.Setenv(SkipFlushEnv, "1")
	if got := flushOutcome(t); got != "skipped" {
		t.Fatalf("in this test binary with %s=1, Flush %s; want skipped", SkipFlushEnv, got)
	}

	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go unavailable")
	}
	bin := filepath.Join(t.TempDir(), "flushprobe")
	build := exec.Command("go", "build", "-o", bin, "./testdata/flushprobe")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build ./testdata/flushprobe: %v\n%s", err, out)
	}
	run := exec.Command(bin)
	run.Env = append(os.Environ(), SkipFlushEnv+"=1")
	out, err := run.Output()
	if err != nil {
		t.Fatalf("flushprobe: %v\n%s", err, out)
	}
	if got := strings.TrimSpace(string(out)); got != "flushed" {
		t.Fatalf("a go-built binary with %s=1 in its environment reports %q; want flushed, "+
			"because the opt-in must never disable durability outside a test binary", SkipFlushEnv, got)
	}
}

// flushCallRe matches a direct flush: a Sync() method call, an fsync syscall,
// or the macOS full-flush fcntl.
var flushCallRe = regexp.MustCompile(`\.Sync\(\)|\bFsync\(|F_FULLFSYNC`)

// TestEveryFlushGoesThroughTheGate holds the gate to one place: no non-test Go
// file in this module flushes except through Flush, whose only Sync is the
// method value in syncFile. A direct call elsewhere would flush in every test
// run, and a copy of the gate would be a second thing to keep out of shipped
// binaries.
func TestEveryFlushGoesThroughTheGate(t *testing.T) {
	root := filepath.Join("..", "..")
	var offenders []string
	for _, top := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, top), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for i, line := range strings.Split(string(data), "\n") {
				if !strings.HasPrefix(strings.TrimSpace(line), "//") && flushCallRe.MatchString(line) {
					offenders = append(offenders, filepath.ToSlash(path)+":"+strconv.Itoa(i+1)+": "+strings.TrimSpace(line))
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", top, err)
		}
	}
	if len(offenders) > 0 {
		t.Fatalf("flushes that bypass fsutil.Flush (route them through it):\n  %s", strings.Join(offenders, "\n  "))
	}
}
