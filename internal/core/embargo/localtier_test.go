package embargo

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The store's local tier holds only files abcd writes; a store whose index
// tracks anything there, under any spelling of its case, or whose disk spells
// a name on the state's path another way, is refused.

// commitLocalTier commits st at rel, a path in the local tier under some
// spelling, in the fixture's checkout.
func commitLocalTier(t *testing.T, f fixture, rel string, st State) {
	t.Helper()
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	f.repo.Write(rel, string(data)+"\n")
	f.repo.Git("add", "-f", "--", rel)
	f.repo.Git("commit", "-q", "-m", "add a file")
}

// TestStoreLocalTierHoldsOnlyWhatAbcdWrites: a store made from a tree that
// tracks a file in the local tier, under the exact spelling or another case,
// holds in its state only what intake records, and its index tracks nothing
// there.
func TestStoreLocalTierHoldsOnlyWhatAbcdWrites(t *testing.T) {
	for _, rel := range []string{StateRel, ".abcd/.WORK.LOCAL/embargo/state.json", ".abcd/.Work.Local/EMBARGO/State.json"} {
		t.Run(rel, func(t *testing.T) {
			f, g := intakeFixture(t)
			at := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
			commitLocalTier(t, f, rel, State{SchemaVersion: SchemaVersion, Repo: "example/example", Advisories: []Advisory{{
				GHSAID: "GHSA-xxxx-xxxx-xxxx", IssueID: "iss-2610120000000001", Title: "a sample report",
				Severity: "major", ForgeState: "triage", PublishedAt: &at,
			}}})
			g.answer("GET", listPath, "triage", 0, page(t,
				advisoryJSON("GHSA-xxxx-xxxx-xxxx", "triage", "high", "a sample report", "the sample body")))
			res := runIntake(t, f)
			if len(res.Minted) != 1 || len(res.Held) != 0 {
				t.Fatalf("intake minted %+v and held %q; want the advisory minted", res.Minted, res.Held)
			}
			s := Store{Root: res.Store, Checkout: f.repo.Root()}
			st, err := s.Verify()
			if err != nil {
				t.Fatal(err)
			}
			if len(st.Advisories) != 1 || st.Advisories[0].PublishedAt != nil || st.Advisories[0].IssueID != res.Minted[0].IssueID {
				t.Fatalf("the store's state %+v; want only what intake recorded", st.Advisories)
			}
			if out, err := storeGit(t, f, s.Root, "ls-files", "--", ":(icase).abcd/.work.local"); err != nil || out != "" {
				t.Fatalf("the store's index tracks %q in its local tier (%v); want nothing", out, err)
			}
			cr, err := Check(CheckRequest{Home: f.home, RepoRoot: f.repo.Root(), Files: []string{"-"},
				Stdin: strings.NewReader("a fix for GHSA-xxxx-xxxx-xxxx\n")})
			if err != nil {
				t.Fatal(err)
			}
			if cr.Clean() {
				t.Fatal("the outbound check let the advisory's id through")
			}
		})
	}
}

// TestStoreStateNamesOriginsRepository: a store made from a tree that tracks
// a state file in the local tier names the repository origin names.
func TestStoreStateNamesOriginsRepository(t *testing.T) {
	f, _ := intakeFixture(t)
	commitLocalTier(t, f, StateRel, State{SchemaVersion: SchemaVersion, Repo: "example/other", Advisories: []Advisory{}})
	res := runIntake(t, f)
	if !res.StoreCreated || res.Repo != "example/example" {
		t.Fatalf("intake %+v; want the store made for origin's repository", res)
	}
}

// TestVerifyRefusesATrackedLocalTier: a store whose index tracks a file in
// its local tier, under the exact spelling or another case, is refused.
func TestVerifyRefusesATrackedLocalTier(t *testing.T) {
	for _, rel := range []string{StateRel, ".abcd/.WORK.LOCAL/embargo/notes.txt", ".ABCD/.work.local/x"} {
		t.Run(rel, func(t *testing.T) {
			f := newFixture(t)
			s, _, err := openStore(f.home, f.repo.Root())
			if err != nil {
				t.Fatal(err)
			}
			if err := s.SaveState(emptyState()); err != nil {
				t.Fatal(err)
			}
			if _, err := s.Verify(); err != nil {
				t.Fatalf("an untracked state was refused: %v", err)
			}
			p := filepath.Join(s.Root, filepath.FromSlash(rel))
			if rel != StateRel {
				if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, []byte("x\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if out, err := storeGit(t, f, s.Root, "add", "-f", "--", rel); err != nil {
				t.Fatalf("add: %v %s", err, out)
			}
			_, err = s.Verify()
			mustRefuse(t, err, "tracks a file in its local tier")
		})
	}
}

// TestVerifyRefusesAFolderSpelledAnotherWay: a folder on the state's path
// whose name differs from it only by case, a Kelvin sign for a K included, is
// refused, and the remedy renames that folder in place, moving nothing out of
// the store.
func TestVerifyRefusesAFolderSpelledAnotherWay(t *testing.T) {
	for rel, found := range map[string]string{
		".abcd/.Work.Local/embargo":      ".Work.Local",
		".abcd/.work.local/Embargo":      "Embargo",
		".abcd/.wor\u212a.local/embargo": ".wor\u212a.local",
	} {
		t.Run(rel, func(t *testing.T) {
			f := newFixture(t)
			s, _, err := openStore(f.home, f.repo.Root())
			if err != nil {
				t.Fatal(err)
			}
			dir := filepath.Join(s.Root, filepath.FromSlash(rel))
			if err := os.MkdirAll(dir, 0o700); err != nil {
				t.Fatal(err)
			}
			data, _ := json.Marshal(emptyState())
			if err := os.WriteFile(filepath.Join(dir, "state.json"), data, 0o600); err != nil {
				t.Fatal(err)
			}
			_, err = s.Verify()
			mustRefuse(t, err, "differs from")
			exact := map[string]string{".Work.Local": ".work.local", "Embargo": "embargo", ".wor\u212a.local": ".work.local"}[found]
			if want := fmt.Sprintf("rename %q to %s where it stands", found, exact); !strings.Contains(err.Error(), want) ||
				!strings.Contains(err.Error(), "moving nothing out of the store") {
				t.Fatalf("refusal %q; want the remedy to say %q and to move nothing out of the store", err, want)
			}
		})
	}
}

// TestVerifyRefusesALinkedStateFolder: a folder on the state's path that is a
// link is refused, so the state is never read from wherever it points.
func TestVerifyRefusesALinkedStateFolder(t *testing.T) {
	f := newFixture(t)
	s, _, err := openStore(f.home, f.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	elsewhere := filepath.Join(s.Root, "elsewhere")
	if err := os.MkdirAll(elsewhere, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := (Store{Root: elsewhere}).SaveState(emptyState()); err != nil {
		t.Fatal(err)
	}
	local := filepath.Join(s.Root, ".abcd", ".work.local")
	if err := os.MkdirAll(local, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(elsewhere, ".abcd", ".work.local", "embargo"), filepath.Join(local, "embargo")); err != nil {
		t.Fatal(err)
	}
	_, err = s.Verify()
	mustRefuse(t, err, "not a real directory")
}

// TestCheckRefusesAnotherStoreWithATrackedLocalTier: another project's store
// under the same home is held to the same rules before its state is read:
// a tracked local tier, or a folder on the state's path spelled another way,
// refuses the check.
func TestCheckRefusesAnotherStoreWithATrackedLocalTier(t *testing.T) {
	t.Run("tracked", func(t *testing.T) {
		f := newFixture(t)
		other := unrelatedStore(t, f.home, emptyState())
		if out, err := storeGit(t, f, other.Root, "add", "-f", "--", StateRel); err != nil {
			t.Fatalf("add: %v %s", err, out)
		}
		_, err := checkStdin(f.home, f.repo.Root(), "a text\n")
		mustRefuse(t, err, "tracks a file in its local tier")
	})
	t.Run("spelled another way", func(t *testing.T) {
		f := newFixture(t)
		other := unrelatedStore(t, f.home, emptyState())
		if err := os.RemoveAll(filepath.Join(other.Root, ".abcd", ".work.local")); err != nil {
			t.Fatal(err)
		}
		dir := filepath.Join(other.Root, ".abcd", ".Work.Local", "embargo")
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		data, _ := json.Marshal(emptyState())
		if err := os.WriteFile(filepath.Join(dir, "state.json"), data, 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := checkStdin(f.home, f.repo.Root(), "a text\n")
		mustRefuse(t, err, "differs from")
	})
}
