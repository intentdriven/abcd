package guard

import (
	"errors"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestTheCommittedRegistryIsReadPastACommitGraph: a commit-graph file altered
// to claim, for HEAD, a tree that carries a kill switch, which a revision walk
// then reads as HEAD's tree. The committed registry is read as stored, so an
// uncommitted kill switch is still a weakening, refused, and FileAtCommit
// finds no file at HEAD.
func TestTheCommittedRegistryIsReadPastACommitGraph(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Write("README", "x\n")
	repo.Commit("seed")
	head := repo.Git("rev-parse", "HEAD")
	repo.Write(RepoRelPath, killSwitch)
	repo.Git("add", "-A")
	forged := repo.Git("write-tree")
	repo.Git("reset", "-q")
	repo.Git("commit-graph", "write", "--reachable")
	gittest.ForgeCommitGraphTree(t, repo.Git("rev-parse", "--absolute-git-dir"), head, forged)
	if got := repo.Git("-c", "core.commitGraph=true", "log", "-1", "--format=%T"); got != forged {
		t.Fatalf("the altered commit-graph does not take: HEAD's tree reads as %s", got)
	}

	r, err := Load(repo.Root())
	if !errors.Is(err, ErrUncommittedOverride) {
		t.Fatalf("Load error = %v, want %v: HEAD stores no kill switch", err, ErrUncommittedOverride)
	}
	if r.Disabled {
		t.Fatal("a kill switch only the commit-graph claims disabled the guard")
	}
	if _, ok, err := FileAtCommit(repo.Root(), head, RepoRelPath, maxGuardFileBytes); err != nil || ok {
		t.Fatalf("FileAtCommit = %v, %v; want no file at the stored HEAD", ok, err)
	}
}
