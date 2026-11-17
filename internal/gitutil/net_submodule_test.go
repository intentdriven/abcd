package gitutil_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestAFetchAbcdMakesGoesIntoNoSubmodule: git's default for a fetch is on
// demand, so a fetch that brings a changed submodule commit runs `git fetch`
// inside each checked-out submodule, under that submodule's own settings
// (git-config(1), fetch.recurseSubmodules). A fetch abcd makes through a
// networked runner carries fetch.recurseSubmodules=false and
// submodule.recurse=false on its command line, so the transport the
// submodule's own settings name is not started, even where the tree's
// .gitmodules asks for recursion; the superproject's refs still arrive.
func TestAFetchAbcdMakesGoesIntoNoSubmodule(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX transport command")
	}
	up := gittest.NewRepo(t)
	up.Write("a.txt", "a\n")
	up.Commit("seed")
	up.AddSubmodule("sub")
	down := gittest.NewRepo(t)
	down.Git("remote", "add", "origin", up.Root())
	down.Git("fetch", "-q", "origin")
	down.Git("checkout", "-q", "-B", "main", "origin/main")
	down.Git("-c", "protocol.file.allow=always", "submodule", "--quiet", "update", "--init")
	down.Git("config", "--file", ".gitmodules", "submodule.sub.fetchRecurseSubmodules", "true")
	marker := filepath.Join(t.TempDir(), "submodule-transport-ran")
	subConfig := filepath.Join(down.Root(), ".git", "modules", "sub", "config")
	down.Git("config", "--file", subConfig, "remote.origin.url", "ext::sh -c touch% "+marker)
	down.Git("config", "--file", subConfig, "protocol.ext.allow", "always")
	up.Git("-C", "sub", "commit", "-q", "--allow-empty", "-m", "sub moves")
	up.Git("commit", "-q", "-a", "-m", "bump the submodule")
	want := up.Git("rev-parse", "HEAD")

	for _, run := range []struct {
		name  string
		fetch func() error
	}{
		{"Net", func() error { _, err := gitutil.Net(down.Root(), "fetch", "origin"); return err }},
		{"NetInputContext", func() error {
			_, err := gitutil.NetInputContext(context.Background(), down.Root(), "+refs/heads/main:refs/remotes/origin/main\n", "fetch", "--stdin", "origin")
			return err
		}},
	} {
		down.Git("update-ref", "refs/remotes/origin/main", "HEAD")
		if err := run.fetch(); err != nil {
			t.Fatalf("%s fetch: %v", run.name, err)
		}
		if _, err := os.Stat(marker); err == nil {
			t.Fatalf("%s: the fetch went into the submodule and started the transport its own settings name", run.name)
		}
		if got := down.Git("rev-parse", "refs/remotes/origin/main"); got != want {
			t.Fatalf("%s: origin/main = %s, want %s: the superproject's ref did not arrive", run.name, got, want)
		}
	}
}
