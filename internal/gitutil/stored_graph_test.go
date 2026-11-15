package gitutil_test

import (
	"slices"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestTheStoredReadReadsNoCommitGraph: a commit-graph file is a cache git takes a
// commit's tree and parents from in place of the commit, so one altered by hand
// can claim another tree for HEAD. A read as stored takes the commit's own,
// whether or not the environment turns git's test switch for a graph on.
func TestTheStoredReadReadsNoCommitGraph(t *testing.T) {
	for name, env := range map[string]string{"plain": "", "test switch turned on": "1"} {
		t.Run(name, func(t *testing.T) {
			r := gittest.NewRepo(t)
			r.Write("a.txt", "stored\n")
			r.Commit("one")
			head, tree := r.Git("rev-parse", "HEAD"), r.Git("rev-parse", "HEAD^{tree}")
			r.Write("a.txt", "forged\n")
			r.Git("add", "-A")
			forged := r.Git("write-tree")
			r.Git("reset", "-q", "--hard")
			r.Git("commit-graph", "write", "--reachable")
			gittest.ForgeCommitGraphTree(t, r.Git("rev-parse", "--absolute-git-dir"), head, forged)
			if got := r.Git("-c", "core.commitGraph=true", "log", "-1", "--format=%T"); got != forged {
				t.Fatalf("the altered commit-graph does not take: HEAD reads as tree %s", got)
			}
			if env != "" {
				t.Setenv("GIT_TEST_COMMIT_GRAPH", env)
			}
			got, err := gitutil.RunCappedAsStored(r.Root(), 4096, "log", "-1", "--format=%T", "HEAD")
			if err != nil {
				t.Fatal(err)
			}
			if got != tree {
				t.Errorf("read as stored, HEAD's tree is %s, want the stored %s (the graph claims %s)", got, tree, forged)
			}
		})
	}
}

// TestAsStoredEnvKeepsTheCallersConfig: the commit-graph is turned off as the
// last entry of GIT_CONFIG_PARAMETERS, after whatever entries the caller's
// environment carries, which stay; the caller's slice is left as it was.
func TestAsStoredEnvKeepsTheCallersConfig(t *testing.T) {
	in := []string{"A=1", "GIT_CONFIG_PARAMETERS='user.name'='Quinn'", "GIT_TEST_COMMIT_GRAPH=1", "B=2"}
	before := slices.Clone(in)
	out := gitutil.AsStoredEnv(in)
	if !slices.Equal(in, before) {
		t.Errorf("AsStoredEnv changed its argument: %q", in)
	}
	want := []string{
		"A=1", "B=2",
		"GIT_NO_REPLACE_OBJECTS=1",
		"GIT_GRAFT_FILE=/dev/null/none",
		"GIT_CONFIG_PARAMETERS='user.name'='Quinn' 'core.commitgraph=false'",
		"GIT_TEST_COMMIT_GRAPH=0",
	}
	if !slices.Equal(out, want) {
		t.Errorf("AsStoredEnv = %q\nwant %q", out, want)
	}
	if got := gitutil.AsStoredEnv(nil); !slices.Contains(got, "GIT_CONFIG_PARAMETERS='core.commitgraph=false'") {
		t.Errorf("AsStoredEnv(nil) = %q, want the one entry alone", got)
	}
}
