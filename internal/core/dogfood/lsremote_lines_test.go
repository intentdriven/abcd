package dogfood

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestTheBuildersCheckReadsTheBranchsOwnLine: the builder's ls-remote, over
// its own ssh transport (an ssh on PATH that serves a local bare repository
// in the forge's place), reads the reviewed branch's tip from that branch's
// own line however many branches whose names end in refs/heads/main the
// remote lists before it, more than the whole listing the check once kept.
func TestTheBuildersCheckReadsTheBranchsOwnLine(t *testing.T) {
	up := realDir(t)
	git(t, up, "init", "-q")
	git(t, up, "commit", "-q", "--allow-empty", "-m", "c0")
	tip := git(t, up, "rev-parse", "HEAD")
	bare := filepath.Join(realDir(t), "abcd.git")
	gittest.InitBare(t, bare)
	git(t, up, "push", "-q", bare, "HEAD:refs/heads/main")
	pad := strings.Repeat("x", 200)
	var in strings.Builder
	for n := 0; n < 400; n++ {
		fmt.Fprintf(&in, "create refs/heads/a/%d/%s/refs/heads/main %s\n", n, pad, tip)
	}
	cmd := exec.Command("git", "-C", bare, "update-ref", "--stdin")
	cmd.Env, cmd.Stdin = gittest.Env(t), strings.NewReader(in.String())
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("update-ref: %v\n%s", err, out)
	}
	if listed := git(t, up, "ls-remote", bare, "refs/heads/main"); len(listed) <= 64<<10 {
		t.Fatalf("the listing runs to %d bytes; want more than the 64 KiB the check once kept", len(listed))
	}
	// The fake ssh runs the command git asks the far side to run (the last
	// argument, git-upload-pack '<path>'), here.
	bin := realDir(t)
	script := "#!/bin/sh\nfor last; do :; done\nexec sh -c \"git ${last#git-}\"\n"
	if err := os.WriteFile(filepath.Join(bin, "ssh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	got, err := lsRemoteGit(context.Background(), realDir(t), "ssh://forge.example.invalid"+bare, "main")
	if err != nil || got != tip {
		t.Fatalf("lsRemoteGit = %q, %v; want main's tip %s", got, err, tip)
	}
}
