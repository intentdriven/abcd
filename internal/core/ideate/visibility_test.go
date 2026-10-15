package ideate

// visibility_test.go pins iss-2610090740139804: every verdict is private or
// public by the person's answer, and the answer decides the one store it is
// written to. A public verdict is committed under the research store keyed on
// the repository's root commit, with its decision-log pointer; a private one
// lands owner-only in ~/.abcd.noindex/ideas/<root-sha>/ and the repository is
// left exactly as it was.

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/homeignore"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// testRootSHA keys the fixture stores. It is a well-formed object name, not a
// real commit: the store's key is checked for its shape, never resolved.
const testRootSHA = "0123456789abcdef0123456789abcdef01234567"

// publicStore is the store every pre-existing record test writes to: the
// verdict is committed, as it always was, now under the root-sha folder.
var publicStore = Store{Visibility: VisibilityPublic, RootSHA: testRootSHA}

// TestRecordPublicLandsUnderTheRootSHA proves a public verdict lands at
// .abcd/development/research/ideas/<root-sha>/<date>-ideate-<slug>.md and that
// the decision-log pointer names that path.
func TestRecordPublicLandsUnderTheRootSHA(t *testing.T) {
	root := seedRepo(t)
	res, err := Record(root, "a-public-idea", publicStore, encode(t, validPayload()), at)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	// The literal path, not one built from the constants, so a moved constant
	// cannot carry the test along with it.
	want := ".abcd/development/research/ideas/" + testRootSHA + "/2026-07-28-ideate-a-public-idea.md"
	if res.Path != want {
		t.Errorf("Path = %q, want %q", res.Path, want)
	}
	if res.Visibility != VisibilityPublic {
		t.Errorf("Visibility = %q, want public", res.Visibility)
	}
	if res.DecisionPath != DecisionsRelDir {
		t.Errorf("DecisionPath = %q, want %q", res.DecisionPath, DecisionsRelDir)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(want))); err != nil {
		t.Errorf("the public record is not on disk: %v", err)
	}
	if log := readFile(t, root, DecisionsRelDir); !strings.Contains(log, want) {
		t.Errorf("the decision log does not point at %s:\n%s", want, log)
	}
}

// TestRecordPrivateLandsInTheHomeStoreOnly proves a private verdict lands in
// the person's home store, owner-only, and that the repository is untouched:
// no record in the research store and no line in the committed decision log.
func TestRecordPrivateLandsInTheHomeStoreOnly(t *testing.T) {
	root := seedRepo(t)
	home := t.TempDir()
	logBefore := readFile(t, root, DecisionsRelDir)

	store := Store{Visibility: VisibilityPrivate, RootSHA: testRootSHA, Home: home}
	res, err := Record(root, "a-private-idea", store, encode(t, validPayload()), at)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	wantShown := abcdhome.Display("ideas", testRootSHA, "2026-07-28-ideate-a-private-idea.md")
	if res.Path != wantShown {
		t.Errorf("Path = %q, want %q (home-relative, never an absolute home path)", res.Path, wantShown)
	}
	if res.Visibility != VisibilityPrivate {
		t.Errorf("Visibility = %q, want private", res.Visibility)
	}
	if res.DecisionPath != "" {
		t.Errorf("DecisionPath = %q; a private verdict writes no pointer", res.DecisionPath)
	}

	dir := abcdhome.Path(home, "ideas", testRootSHA)
	file := filepath.Join(dir, "2026-07-28-ideate-a-private-idea.md")
	fi, err := os.Stat(file)
	if err != nil {
		t.Fatalf("the private record is not on disk: %v", err)
	}
	if mode := fi.Mode().Perm(); mode != 0o600 {
		t.Errorf("private record mode = %o, want 600", mode)
	}
	di, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if mode := di.Mode().Perm(); mode != 0o700 {
		t.Errorf("private store mode = %o, want 700", mode)
	}
	body, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "# Ideate verdict — a-private-idea") {
		t.Errorf("the private record is not the rendered verdict:\n%s", body)
	}

	if got := readFile(t, root, DecisionsRelDir); got != logBefore {
		t.Errorf("a private verdict changed the committed decision log:\n%s", got)
	}
	assertNothingWritten(t, root)
}

// TestRecordRefusesWithoutAnAnswer proves the verb never writes without the
// person's answer: a store with no visibility, or one outside the enum, is
// refused before anything is written, in either store.
func TestRecordRefusesWithoutAnAnswer(t *testing.T) {
	for _, vis := range []Visibility{"", "secret", "Public"} {
		t.Run(string(vis), func(t *testing.T) {
			root := seedRepo(t)
			home := t.TempDir()
			store := Store{Visibility: vis, RootSHA: testRootSHA, Home: home}
			_, err := Record(root, "an-unanswered-idea", store, encode(t, validPayload()), at)
			var ve *VisibilityError
			if !errors.As(err, &ve) {
				t.Fatalf("err = %v, want a *VisibilityError", err)
			}
			assertNothingWritten(t, root)
			if _, err := os.Stat(abcdhome.Path(home)); !os.IsNotExist(err) {
				t.Errorf("a refused run laid the home store: %v", err)
			}
		})
	}
}

// TestRecordRefusesAMalformedRootSHA proves the store key is held to the shape
// of a git object name before it reaches a path, in both stores.
func TestRecordRefusesAMalformedRootSHA(t *testing.T) {
	for _, vis := range []Visibility{VisibilityPublic, VisibilityPrivate} {
		for _, sha := range []string{"", "../../etc", strings.ToUpper(testRootSHA), testRootSHA[:39]} {
			t.Run(string(vis)+"/"+sha, func(t *testing.T) {
				root := seedRepo(t)
				home := t.TempDir()
				store := Store{Visibility: vis, RootSHA: sha, Home: home}
				if _, err := Record(root, "a-keyless-idea", store, encode(t, validPayload()), at); err == nil {
					t.Fatal("a malformed root sha was accepted")
				}
				assertNothingWritten(t, root)
				if _, err := os.Stat(abcdhome.Path(home)); !os.IsNotExist(err) {
					t.Errorf("a refused run laid the home store: %v", err)
				}
			})
		}
	}
}

// TestRecordPrivateRefusesAHomeInsideAWorkingTree proves a private verdict is
// never written where a commit could carry it: a home that lies inside a git
// working tree (a HOME a repository's settings planted) is refused.
func TestRecordPrivateRefusesAHomeInsideAWorkingTree(t *testing.T) {
	root := seedRepo(t)
	tree := t.TempDir()
	if err := os.Mkdir(filepath.Join(tree, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(tree, "planted-home")
	if err := os.Mkdir(home, 0o755); err != nil {
		t.Fatal(err)
	}
	store := Store{Visibility: VisibilityPrivate, RootSHA: testRootSHA, Home: home}
	if _, err := Record(root, "a-planted-idea", store, encode(t, validPayload()), at); err == nil {
		t.Fatal("a private verdict was written into a home inside a working tree")
	}
	if _, err := os.Stat(abcdhome.Path(home)); !os.IsNotExist(err) {
		t.Errorf("the refused run laid a store inside the working tree: %v", err)
	}
}

// TestRecordPrivateRefusesAnOverwrite proves the private store keeps the
// no-overwrite guarantee the public one has.
func TestRecordPrivateRefusesAnOverwrite(t *testing.T) {
	root := seedRepo(t)
	store := Store{Visibility: VisibilityPrivate, RootSHA: testRootSHA, Home: t.TempDir()}
	if _, err := Record(root, "a-repeated-idea", store, encode(t, validPayload()), at); err != nil {
		t.Fatalf("first Record: %v", err)
	}
	_, err := Record(root, "a-repeated-idea", store, encode(t, validPayload()), at)
	var ee *ExistsError
	if !errors.As(err, &ee) {
		t.Fatalf("second Record err = %v, want an *ExistsError", err)
	}
	if strings.Contains(ee.Path, store.Home) {
		t.Errorf("the refusal echoes the absolute home path: %s", ee.Path)
	}
}

// TestRecordPrivateRedactsLikeThePublicStore proves a private verdict passes the
// same redactor: a private note is a file on disk a person may publish by hand,
// so a secret span in the verdict never reaches it.
func TestRecordPrivateRedactsLikeThePublicStore(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := seedRepo(t)
	p := validPayload()
	p["idea"] = "the gauntlet should read the token " + fakeIdeateToken
	store := Store{Visibility: VisibilityPrivate, RootSHA: testRootSHA, Home: home}
	res, err := Record(root, "a-leaky-private-idea", store, encode(t, p), at)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if res.Redactions == 0 {
		t.Error("the private verdict reports no redaction")
	}
	body, err := os.ReadFile(abcdhome.Path(home, "ideas", testRootSHA, "2026-07-28-ideate-a-leaky-private-idea.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), fakeIdeateToken) {
		t.Error("the private record carries the raw secret span")
	}
}

// TestRecordPrivateRefusesAnUnresolvedHome proves an empty or relative home is
// refused before anything is written, rather than resolved against wherever the
// verb happens to run.
func TestRecordPrivateRefusesAnUnresolvedHome(t *testing.T) {
	for _, home := range []string{"", "relative/home"} {
		t.Run(home, func(t *testing.T) {
			root := seedRepo(t)
			t.Chdir(t.TempDir())
			store := Store{Visibility: VisibilityPrivate, RootSHA: testRootSHA, Home: home}
			if _, err := Record(root, "a-homeless-idea", store, encode(t, validPayload()), at); err == nil {
				t.Fatal("an unresolved home was accepted")
			}
			if home != "" {
				if _, err := os.Stat(abcdhome.Path(home)); !os.IsNotExist(err) {
					t.Errorf("a refused run laid a store under the relative home: %v", err)
				}
			}
		})
	}
}

// TestRecordPrivateRefusesASymlinkedLevel proves the home-scope walk refuses a
// level of the private store symlinked elsewhere, rather than writing through it.
func TestRecordPrivateRefusesASymlinkedLevel(t *testing.T) {
	root := seedRepo(t)
	home := t.TempDir()
	elsewhere := t.TempDir()
	if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(elsewhere, abcdhome.Path(home, "ideas")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	store := Store{Visibility: VisibilityPrivate, RootSHA: testRootSHA, Home: home}
	if _, err := Record(root, "a-redirected-idea", store, encode(t, validPayload()), at); err == nil {
		t.Fatal("a symlinked level of the private store was written through")
	}
	entries, err := os.ReadDir(elsewhere)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		t.Errorf("the refused run wrote %s at the link's target", e.Name())
	}
}

// TestRecordPrivateWritesIntoTheAccountsOwnTrackedHome proves the dotfiles case
// itd-2610140858048004 settles: a home that is the account's own git working
// tree is not refused, because abcd's home ignores itself there; the refusal
// is kept for a HOME planted inside some other checkout.
func TestRecordPrivateWritesIntoTheAccountsOwnTrackedHome(t *testing.T) {
	root := seedRepo(t)
	home := t.TempDir()
	if err := os.Mkdir(filepath.Join(home, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	restore := gitutil.SwapAccountHomeForTest(func() (string, error) { return home, nil })
	defer restore()
	store := Store{Visibility: VisibilityPrivate, RootSHA: testRootSHA, Home: home}
	if _, err := Record(root, "a-dotfiles-idea", store, encode(t, validPayload()), at); err != nil {
		t.Fatalf("a private verdict in the account's own tracked home was refused: %v", err)
	}
	if _, err := os.Stat(abcdhome.Path(home, "ideas", testRootSHA, "2026-07-28-ideate-a-dotfiles-idea.md")); err != nil {
		t.Errorf("the private record is not on disk: %v", err)
	}
}

// TestRecordPrivateLaysTheIgnoreFileFirst proves the private store does not
// rest on the front door: the verb lays abcd's home ignore file itself before
// the verdict is written, so a run whose front-door lay was skipped or silent
// still leaves the verdict where a `git add -A` in the home passes it by.
func TestRecordPrivateLaysTheIgnoreFileFirst(t *testing.T) {
	root := seedRepo(t)
	home := t.TempDir()
	store := Store{Visibility: VisibilityPrivate, RootSHA: testRootSHA, Home: home}
	if _, err := Record(root, "an-ignored-idea", store, encode(t, validPayload()), at); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if _, err := os.Stat(abcdhome.Path(home, homeignore.IgnoreName)); err != nil {
		t.Errorf("the private verdict was written with no ignore file laid: %v", err)
	}
}

// TestRecordPrivateRefusesWhenTheIgnoreFileCannotBeLaid proves a home the
// ignore file cannot be laid in takes no private verdict: the verb refuses and
// writes nothing, rather than leave a verdict a home repository would stage.
func TestRecordPrivateRefusesWhenTheIgnoreFileCannotBeLaid(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes through a read-only directory")
	}
	root := seedRepo(t)
	home := t.TempDir()
	dir := abcdhome.Path(home)
	if err := os.MkdirAll(filepath.Join(dir, "ideas", testRootSHA), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	store := Store{Visibility: VisibilityPrivate, RootSHA: testRootSHA, Home: home}
	if _, err := Record(root, "an-unguarded-idea", store, encode(t, validPayload()), at); err == nil {
		t.Fatal("a private verdict was recorded where the ignore file could not be laid")
	}
	entries, err := os.ReadDir(filepath.Join(dir, "ideas", testRootSHA))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		t.Errorf("the refused run wrote %s", e.Name())
	}
}
