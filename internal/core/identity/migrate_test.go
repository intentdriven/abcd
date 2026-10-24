package identity

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/provenance"
	"github.com/intentdriven/abcd/internal/gittest"
)

// committedRepo is gitRepo with one commit, so the repository has a root
// commit to key its own record on, and the identity it commits under also in
// the environment (isolate blanks GIT_AUTHOR_*, which git reads as empty).
func committedRepo(t *testing.T, name, email string) string {
	t.Helper()
	dir := gitRepo(t, name, email)
	t.Setenv("GIT_AUTHOR_NAME", name)
	t.Setenv("GIT_AUTHOR_EMAIL", email)
	runGitT(t, dir, "commit", "--allow-empty", "-m", "root", "--no-verify")
	return dir
}

// Criterion 4, on the owner's machine: the pin moves to the owner's own record
// for this repository, its production mode moves to the repository's own
// committed file, and the pin leaves the working tree.
func TestMigration_OwnerMovesThePin(t *testing.T) {
	isolate(t)
	dir := committedRepo(t, "A Maintainer", "maintainer@example.com")
	writeRepoPin(t, dir, `{"name":"A Maintainer","email":"maintainer@example.com","production_mode":"dictated-and-formatted"}`)
	home := recordHome(t)
	plan, found, err := PlanMigration(dir, home)
	if err != nil || !found {
		t.Fatalf("PlanMigration: found=%v err=%v", found, err)
	}
	if !plan.Owner || plan.Blocker != "" || plan.Target.Scope != ScopeRepository {
		t.Fatalf("want an owner plan for the repository's own record, got %+v", plan)
	}
	res, err := ApplyMigration(dir, home, plan)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Recorded || !res.ProductionModeMoved || !res.Removed {
		t.Fatalf("migration did not do all three: %+v", res)
	}
	if _, err := os.Stat(filepath.Join(dir, LegacyPinRelPath)); !os.IsNotExist(err) {
		t.Fatalf("the pin is still in the repository: %v", err)
	}
	m, err := provenance.DeclaredMode(dir)
	if err != nil || m != provenance.ModeDictatedAndFormatted {
		t.Fatalf("the production mode was not kept: %q %v", m, err)
	}
	got, err := Check(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusOK || got.RecordAt != plan.Target {
		t.Fatalf("after the move the owner's commits are not checked against their record: %v at %+v", got.Status, got.RecordAt)
	}
}

// Criterion 4, on anyone else's machine: the pin is not the person's, so it
// is not copied to their machine and abcd does not move it.
func TestMigration_NonOwnerIsNotGivenThePin(t *testing.T) {
	isolate(t)
	dir := committedRepo(t, "Second Contributor", "second@example.com")
	writeRepoPin(t, dir, `{"name":"A Maintainer","email":"maintainer@example.com"}`)
	home := recordHome(t)
	plan, found, err := PlanMigration(dir, home)
	if err != nil || !found {
		t.Fatalf("PlanMigration: found=%v err=%v", found, err)
	}
	if plan.Owner {
		t.Fatalf("a contributor was taken for the pin's owner: %+v", plan)
	}
	if _, err := ApplyMigration(dir, home, plan); err == nil {
		t.Fatal("ApplyMigration must refuse a person who is not the owner")
	}
	if _, _, found, err := LoadRecord(home, gittestRoot(t, dir)); found || err != nil {
		t.Fatalf("the maintainer's identity reached a contributor's machine: found=%v err=%v", found, err)
	}
	if _, err := os.Stat(filepath.Join(dir, LegacyPinRelPath)); err != nil {
		t.Fatalf("the pin was removed on a contributor's machine: %v", err)
	}
}

// A pin holding a member the migration cannot carry is left in place: removing
// it would drop a setting (criterion 4: keeps the repository's other settings).
func TestMigration_UnknownMemberBlocks(t *testing.T) {
	isolate(t)
	dir := committedRepo(t, "A Maintainer", "maintainer@example.com")
	writeRepoPin(t, dir, `{"name":"A Maintainer","email":"maintainer@example.com","signing_key":"x"}`)
	home := recordHome(t)
	plan, _, err := PlanMigration(dir, home)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Blocker == "" {
		t.Fatalf("want a blocker naming signing_key, got %+v", plan)
	}
	if _, err := ApplyMigration(dir, home, plan); err == nil {
		t.Fatal("ApplyMigration must refuse a blocked plan")
	}
	if _, err := os.Stat(filepath.Join(dir, LegacyPinRelPath)); err != nil {
		t.Fatalf("a blocked pin was removed: %v", err)
	}
}

// The owner's own record wins: a different record already at the target is
// kept, and the pin still leaves the repository.
func TestMigration_KeepsADifferentRecord(t *testing.T) {
	isolate(t)
	dir := committedRepo(t, "A Maintainer", "maintainer@example.com")
	writeRepoPin(t, dir, `{"name":"A Maintainer","email":"maintainer@example.com"}`)
	home := recordHome(t)
	ref, err := RepoRecordRef(gittestRoot(t, dir))
	if err != nil {
		t.Fatal(err)
	}
	writeHomeFile(t, ref.Rel, `{"name":"A Maintainer","email":"maintainer@work.example"}`)
	plan, _, err := PlanMigration(dir, home)
	if err != nil {
		t.Fatal(err)
	}
	res, err := ApplyMigration(dir, home, plan)
	if err != nil {
		t.Fatal(err)
	}
	if !res.KeptDifferent || res.Recorded || !res.Removed {
		t.Fatalf("want the existing record kept and the pin removed: %+v", res)
	}
	got, _, _, _ := LoadRecord(home, gittestRoot(t, dir))
	if got.Email != "maintainer@work.example" {
		t.Fatalf("the person's own record was overwritten: %+v", got)
	}
}

// ProposeRecord offers the person's usual git identity, the global one first,
// and never a machine identity (criterion 1).
func TestProposeRecord(t *testing.T) {
	isolate(t)
	dir := gitRepo(t, "Repo Local", "local@example.com")
	p, ok, err := ProposeRecord(dir)
	if err != nil || !ok || p.Identity.Name != "Repo Local" {
		t.Fatalf("with no global identity, want the repository's: %+v ok=%v err=%v", p, ok, err)
	}
	machine := gitRepo(t, "Claude", "noreply@anthropic.com")
	if _, ok, err := ProposeRecord(machine); ok || err != nil {
		t.Fatalf("a machine identity was proposed: ok=%v err=%v", ok, err)
	}
}

func gittestRoot(t *testing.T, dir string) string {
	t.Helper()
	return gittest.RootCommit(t, dir)
}

// The committed pin is read through the guarded reader: a symlinked pin, even
// one pointing at a well-formed pin inside the working tree, or a FIFO, is
// refused rather than followed or opened.
func TestLoadLegacyPin_ReadsThroughTheGuardedReader(t *testing.T) {
	isolate(t)
	dir := gitRepo(t, "A Maintainer", "maintainer@example.com")
	real := filepath.Join(dir, "elsewhere.json")
	if err := os.WriteFile(real, []byte(`{"name":"A Maintainer","email":"maintainer@example.com"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".abcd", "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	pin := filepath.Join(dir, filepath.FromSlash(LegacyPinRelPath))
	// A relative target inside the working tree, which an os.Root alone follows.
	if err := os.Symlink(filepath.Join("..", "..", filepath.Base(real)), pin); err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadLegacyPin(dir); err == nil {
		t.Fatal("a symlinked pin was followed")
	}
	if err := os.Remove(pin); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pin, []byte(`{"name":"A Maintainer","email":"maintainer@example.com","pad":"`+strings.Repeat("x", 70<<10)+`"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadLegacyPin(dir); err == nil {
		t.Fatal("a pin over 64 KiB was read")
	}
}
