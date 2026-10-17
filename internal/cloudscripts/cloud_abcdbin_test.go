package cloudscripts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// scripts/cloud/abcd-bin.sh, which gives the cloud scripts an abcd built from
// this checkout (spc-2610120941441282, "The brief"). ABCD_BIN is honoured only
// when it names the binary the cloud setup builds in this checkout's local
// tier, .abcd/.work.local/cloud/bin/abcd, both sides resolved with pwd -P:
// on any other machine, or pointing anywhere else, a variable left set is no
// way to run another abcd than this checkout's, and abcd is built.

const abcdBinScript = "scripts/cloud/abcd-bin.sh"

func TestAbcdBinIgnoresAForeignABCDBIN(t *testing.T) {
	c := newBriefCase(t)
	foreign := ""
	for _, kv := range c.env {
		if strings.HasPrefix(kv, "FAKE_ABCD=") {
			foreign = strings.TrimPrefix(kv, "FAKE_ABCD=")
		}
	}
	if foreign == "" {
		t.Fatal("the fixture carries no fake abcd")
	}
	run := func(t *testing.T, bin string) (path string, built int) {
		t.Helper()
		c.resetCalls()
		dir := t.TempDir()
		out, errOut, code := c.scriptSplit([]string{"ABCD_BIN=" + bin}, abcdBinScript, dir)
		if code != 0 {
			t.Fatalf("abcd-bin.sh with ABCD_BIN=%s exited %d\n%s%s", bin, code, out, errOut)
		}
		path = strings.TrimSpace(out)
		if n := strings.Count(c.callLog(), "go build -o "); n > 0 {
			if path != filepath.Join(dir, "abcd") {
				t.Errorf("abcd was built, but the path printed is %q, not the build's %q", path, filepath.Join(dir, "abcd"))
			}
			return path, n
		}
		return path, 0
	}

	t.Run("an executable outside the checkout", func(t *testing.T) {
		if path, built := run(t, foreign); built != 1 || path == foreign {
			t.Errorf("ABCD_BIN=%s was used (printed %q, built %d times); want it ignored and abcd built once", foreign, path, built)
		}
	})
	t.Run("an executable elsewhere in the checkout", func(t *testing.T) {
		other := filepath.Join(c.dir, ".abcd", ".work.local", "abcd")
		if err := os.MkdirAll(filepath.Dir(other), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(other, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		if path, built := run(t, other); built != 1 || path == other {
			t.Errorf("ABCD_BIN=%s was used (printed %q, built %d times); want it ignored", other, path, built)
		}
	})
	t.Run("a symlink in the setup's place", func(t *testing.T) {
		link := filepath.Join(c.dir, filepath.FromSlash(localAbcdRel))
		if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
			t.Fatal(err)
		}
		_ = os.Remove(link)
		if err := os.Symlink(foreign, link); err != nil {
			t.Fatal(err)
		}
		defer os.Remove(link)
		if path, built := run(t, link); built != 1 || path == link {
			t.Errorf("a symlink at %s was used (printed %q, built %d times); want it ignored", localAbcdRel, path, built)
		}
	})
	t.Run("the setup's binary", func(t *testing.T) {
		local := c.localAbcd(foreign)
		if path, built := run(t, local); built != 0 || path != local {
			t.Errorf("ABCD_BIN naming the setup's binary printed %q and built %d times; want it printed as is", path, built)
		}
		dotted := filepath.Join(c.dir, ".abcd", ".work.local", "cloud", "bin", "..", "bin", "abcd")
		if path, built := run(t, dotted); built != 0 || path != dotted {
			t.Errorf("ABCD_BIN naming the setup's binary by another spelling printed %q and built %d times", path, built)
		}
	})
}
