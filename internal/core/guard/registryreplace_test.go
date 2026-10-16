package guard

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestRegistryAtReadsObjectsAsStored: the registry at the run's base is read
// from the objects as git stores them, never through a refs/replace/* ref,
// which would stand another blob in for the committed .abcd/guard.json (or
// another commit in for the base) and so change the registry the run is
// judged against.
func TestRegistryAtReadsObjectsAsStored(t *testing.T) {
	repo, base := baseRepo(t, repoBlocker)
	blob := strings.TrimSpace(repo.Git("rev-parse", base+":"+RepoRelPath))
	other := filepath.Join(t.TempDir(), "guard.json")
	if err := os.WriteFile(other, []byte(killSwitch), 0o600); err != nil {
		t.Fatal(err)
	}
	stand := strings.TrimSpace(repo.Git("hash-object", "-w", other))
	repo.Git("replace", blob, stand)
	if out, err := exec.Command("git", "-C", repo.Root(), "cat-file", "blob", base+":"+RepoRelPath).Output(); err != nil || !strings.Contains(string(out), `"disabled":true`) {
		t.Fatalf("the replacement is not in place (%v): %s", err, out)
	}
	r := registryAt(repo.Root(), base)
	if r.Disabled {
		t.Fatal("the registry at the base was read through a replacement blob: it is switched off")
	}
	if _, ok := r.Entries["no-make-clean"]; !ok {
		t.Error("the registry at the base lacks the committed entry")
	}

	// A replacement of the base commit itself, by a commit whose tree holds
	// the kill switch.
	repo2, base2 := baseRepo(t, repoBlocker)
	repo2.Write(RepoRelPath, killSwitch)
	repo2.Commit("off")
	off := strings.TrimSpace(repo2.Git("rev-parse", "HEAD"))
	repo2.Git("replace", "-f", base2, off)
	if r := registryAt(repo2.Root(), base2); r.Disabled {
		t.Fatal("the registry at the base was read through a replacement commit: it is switched off")
	}
}
