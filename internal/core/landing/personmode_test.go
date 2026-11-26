package landing

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/identity"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/statusline"
	"github.com/intentdriven/abcd/internal/gittest"
)

// A person's own entry for one project (itd-2611252049031692, step 2): read
// from ~/.abcd.noindex/facilitator.json under the home the account database
// names, selected by the identity record alone, matched on the default
// branch's root commit (and, for auto, the repository's name), and deciding
// only where the project leaves the mode to each contributor.

const (
	personName  = "Ada Person"
	personEmail = "ada@example.invalid"
	personForge = "owner/repo"
)

// personRepo is mergingRepo with an origin address naming personForge, git's
// identity set to the person's, and the person's identity record written in
// the test's HOME. It returns the repository, its root and its root commit.
func personRepo(t *testing.T, project string) (*gittest.Repo, string, string) {
	t.Helper()
	repo, root := mergingRepo(t, project)
	repo.Git("remote", "add", "origin", "https://forge.invalid/Owner/Repo.git")
	repo.Git("config", "user.name", personName)
	repo.Git("config", "user.email", personEmail)
	if err := identity.WriteRecord(os.Getenv("HOME"), identity.MachineRecordRef(), identity.Record{Name: personName, Email: personEmail}); err != nil {
		t.Fatal(err)
	}
	return repo, root, gittest.RootCommit(t, root)
}

// entry is one entry of the person's file, as the test writes it.
type entry struct {
	Email string `json:"email"`
	Root  string `json:"root"`
	Forge string `json:"forge"`
	Mode  string `json:"mode"`
	SetAt string `json:"set_at"`
}

// writeEntries writes the person's file in the test's HOME holding es.
func writeEntries(t *testing.T, es ...entry) string {
	t.Helper()
	for i := range es {
		if es[i].SetAt == "" {
			es[i].SetAt = "2026-11-26T01:00:00Z"
		}
	}
	raw, err := json.Marshal(map[string]any{"schema_version": 1, "entries": es})
	if err != nil {
		t.Fatal(err)
	}
	return writePersonFile(t, string(raw))
}

func writePersonFile(t *testing.T, body string) string {
	t.Helper()
	dir := abcdhome.Path(os.Getenv("HOME"))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "facilitator.json")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// readShown is the board's and the prompt line's reading of root.
func readShown(t *testing.T, root string) Facilitator {
	t.Helper()
	f, err := readDefaultFacilitatorContext(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// TestThePersonsEntryDecidesOnlyWhereTheProjectLeavesIt is the table over the
// project's value, the computer's file and the person's entry, each human,
// auto or absent (criteria 1 and 2): the project's value decides; with none,
// the entry; with neither, the computer's file; with none of them, human.
func TestThePersonsEntryDecidesOnlyWhereTheProjectLeavesIt(t *testing.T) {
	values := []string{ModeHuman, ModeAuto, ""}
	config := map[string]string{ModeHuman: humanConfig, ModeAuto: autoConfig, "": ""}
	for _, project := range values {
		for _, computer := range values {
			for _, own := range values {
				name := "project " + project + ", computer " + computer + ", entry " + own
				t.Run(name, func(t *testing.T) {
					_, root, sha := personRepo(t, config[project])
					if computer != "" {
						writeMachineConfig(t, config[computer])
					}
					if own != "" {
						writeEntries(t, entry{Email: personEmail, Root: sha, Forge: personForge, Mode: own})
					}
					f := readShown(t, root)
					mode, layer := ModeHuman, "bundled"
					switch {
					case project != "":
						mode, layer = project, "repo"
					case own != "":
						mode, layer = own, "person"
					case computer != "":
						mode, layer = computer, "machine"
					}
					if f.Mode != mode || f.Layer != layer {
						t.Fatalf("reading = %+v; want %s from the %s layer", f, mode, layer)
					}
					if layer == "person" {
						if want := "left to each contributor, and your own for this project sets " + own; f.Says() != want {
							t.Errorf("Says() = %q, want %q", f.Says(), want)
						}
						if f.merging().Whose != statusline.WhosePerson {
							t.Errorf("the row would say %q, want yours, here", f.merging().Whose)
						}
					}
				})
			}
		}
	}
}

// TestAnEntryIsSelectedByTheIdentityRecordAlone is the identity table
// (criterion 9): an entry is read where git would author and commit as the
// record's person; where git would act as another, or no record can be read,
// the mode is human and the reading says why. An entry for another person or
// another project is none.
func TestAnEntryIsSelectedByTheIdentityRecordAlone(t *testing.T) {
	for _, c := range []struct {
		name     string
		setup    func(t *testing.T, repo *gittest.Repo, sha string)
		mode     string
		layer    string
		says     string
		withheld bool
	}{
		{"the record's person, as git", func(t *testing.T, _ *gittest.Repo, sha string) {
			writeEntries(t, entry{Email: personEmail, Root: sha, Forge: personForge, Mode: ModeAuto})
		}, ModeAuto, "person", "", false},
		{"the record's person, in git's global settings alone", func(t *testing.T, repo *gittest.Repo, sha string) {
			// Most people set their identity once, in their own global
			// settings: the check asks git as the person's own git does.
			writeEntries(t, entry{Email: personEmail, Root: sha, Forge: personForge, Mode: ModeAuto})
			repo.Git("config", "--unset", "user.name")
			repo.Git("config", "--unset", "user.email")
			global := "[user]\n\tname = " + personName + "\n\temail = " + personEmail + "\n"
			if err := os.WriteFile(filepath.Join(os.Getenv("HOME"), ".gitconfig"), []byte(global), 0o600); err != nil {
				t.Fatal(err)
			}
		}, ModeAuto, "person", "", false},
		{"git would commit as another", func(t *testing.T, repo *gittest.Repo, sha string) {
			writeEntries(t, entry{Email: personEmail, Root: sha, Forge: personForge, Mode: ModeAuto})
			repo.Git("config", "user.email", "other@example.invalid")
		}, ModeHuman, "person", "other@example.invalid", true},
		{"no identity record", func(t *testing.T, _ *gittest.Repo, sha string) {
			writeEntries(t, entry{Email: personEmail, Root: sha, Forge: personForge, Mode: ModeAuto})
			if err := os.Remove(abcdhome.Path(os.Getenv("HOME"), "identity.json")); err != nil {
				t.Fatal(err)
			}
		}, ModeHuman, "person", "abcd ahoy --identity", true},
		{"a refused identity record", func(t *testing.T, _ *gittest.Repo, sha string) {
			writeEntries(t, entry{Email: personEmail, Root: sha, Forge: personForge, Mode: ModeAuto})
			if err := os.Chmod(abcdhome.Path(os.Getenv("HOME"), "identity.json"), 0o622); err != nil {
				t.Fatal(err)
			}
		}, ModeHuman, "person", "identity.json", true},
		{"an entry for another person", func(t *testing.T, _ *gittest.Repo, sha string) {
			writeEntries(t, entry{Email: "other@example.invalid", Root: sha, Forge: personForge, Mode: ModeAuto})
		}, ModeAuto, "machine", "", false},
		{"an entry for another project", func(t *testing.T, _ *gittest.Repo, _ string) {
			writeEntries(t, entry{Email: personEmail, Root: strings.Repeat("a", 40), Forge: personForge, Mode: ModeHuman})
		}, ModeAuto, "machine", "", false},
		{"no file", func(*testing.T, *gittest.Repo, string) {}, ModeAuto, "machine", "", false},
		{"a fork's auto is none", func(t *testing.T, _ *gittest.Repo, sha string) {
			writeEntries(t, entry{Email: personEmail, Root: sha, Forge: "someone/repo", Mode: ModeAuto})
		}, ModeAuto, "machine", "", false},
		{"a fork shares human", func(t *testing.T, _ *gittest.Repo, sha string) {
			writeEntries(t, entry{Email: personEmail, Root: sha, Forge: "someone/repo", Mode: ModeHuman})
		}, ModeHuman, "person", "", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			repo, root, sha := personRepo(t, "")
			writeMachineConfig(t, autoConfig)
			c.setup(t, repo, sha)
			f := readShown(t, root)
			if f.Mode != c.mode || f.Layer != c.layer || f.Withheld != c.withheld {
				t.Fatalf("reading = %+v; want %s from the %s layer, withheld %t", f, c.mode, c.layer, c.withheld)
			}
			if c.says != "" && !strings.Contains(f.Origin, c.says) {
				t.Errorf("origin = %q, want it to name %q", f.Origin, c.says)
			}
		})
	}
}

// TestARefusedPersonsFileReadsHumanOrIsSaidBesideTheProjects (criterion 14):
// where the project leaves it to each, a file that is there and refused reads
// human, naming the file in its tilde form; where the project has a value,
// that value stands and the refusal is said beside it.
func TestARefusedPersonsFileReadsHumanOrIsSaidBesideTheProjects(t *testing.T) {
	for name, damage := range map[string]func(t *testing.T, sha string){
		"not JSON":         func(t *testing.T, _ string) { writePersonFile(t, "auto") },
		"another schema":   func(t *testing.T, _ string) { writePersonFile(t, `{"schema_version":2,"entries":[]}`) },
		"an unknown field": func(t *testing.T, _ string) { writePersonFile(t, `{"schema_version":1,"entries":[],"by":"x"}`) },
		"a value that is not a mode": func(t *testing.T, sha string) {
			writeEntries(t, entry{Email: personEmail, Root: sha, Forge: personForge, Mode: "robot"})
		},
		"writable by others": func(t *testing.T, sha string) {
			p := writeEntries(t, entry{Email: personEmail, Root: sha, Forge: personForge, Mode: ModeAuto})
			if err := os.Chmod(p, 0o622); err != nil {
				t.Fatal(err)
			}
		},
		"a symlink": func(t *testing.T, sha string) {
			p := writeEntries(t, entry{Email: personEmail, Root: sha, Forge: personForge, Mode: ModeAuto})
			to := filepath.Join(t.TempDir(), "elsewhere")
			if err := os.Rename(p, to); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(to, p); err != nil {
				t.Fatal(err)
			}
		},
		"over its size": func(t *testing.T, _ string) {
			writePersonFile(t, `{"schema_version":1,"entries":[]}`+strings.Repeat(" ", 70<<10))
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, root, sha := personRepo(t, "")
			writeMachineConfig(t, autoConfig)
			damage(t, sha)
			f := readShown(t, root)
			if f.Mode != ModeHuman || f.Layer != "person" || !f.Withheld || !strings.Contains(f.Origin, abcdhome.Display("facilitator.json")) {
				t.Fatalf("reading = %+v; want human from the person layer naming the file", f)
			}

			_, root, sha = personRepo(t, autoConfig)
			damage(t, sha)
			f = readShown(t, root)
			if f.Mode != ModeAuto || f.Layer != "repo" || !strings.Contains(f.Note, abcdhome.Display("facilitator.json")) {
				t.Fatalf("reading = %+v; want the project's auto, with the refusal said beside it", f)
			}
		})
	}
}

// TestUnderAnotherHomeAPersonHasNoSetting (criteria 11 and 24): a person's
// entry and the computer's file are the ones in the home the account database
// names; under any other HOME, a hosted session's among them, neither is
// read, so the mode is human where the project leaves it to each, and the
// project's value where it has one.
func TestUnderAnotherHomeAPersonHasNoSetting(t *testing.T) {
	for _, c := range []struct {
		project, mode, layer string
	}{
		{"", ModeHuman, "machine"},
		{autoConfig, ModeAuto, "repo"},
	} {
		_, root, sha := personRepo(t, c.project)
		writeMachineConfig(t, autoConfig)
		writeEntries(t, entry{Email: personEmail, Root: sha, Forge: personForge, Mode: ModeAuto})
		other := t.TempDir()
		prev := AccountHome
		AccountHome = func() (string, error) { return other, nil }
		f := readShown(t, root)
		AccountHome = prev
		if f.Mode != c.mode || f.Layer != c.layer {
			t.Errorf("project %q under another HOME: reading = %+v; want %s from the %s layer", c.project, f, c.mode, c.layer)
		}
		if c.project == "" && (!f.HomeRefused || !strings.Contains(f.Origin, "HOME")) {
			t.Errorf("reading = %+v; want it to say HOME is not this account's", f)
		}
	}
}

// TestTheGateTakesAPersonsHumanAndHoldsBackTheirAuto: the merge gate reads a
// person's human for this project over the computer's auto; a person's own
// auto arms only a pull request their own account opened (decision 7), which
// the gate does not yet read, so it reads human and says so.
func TestTheGateTakesAPersonsHumanAndHoldsBackTheirAuto(t *testing.T) {
	repo, root, sha := personRepo(t, "")
	writeMachineConfig(t, autoConfig)
	writeEntries(t, entry{Email: personEmail, Root: sha, Forge: personForge, Mode: ModeHuman})
	roots := layered.Roots{Home: os.Getenv("HOME")}
	tip := repo.Git("rev-parse", "HEAD")
	f, err := readFacilitatorAt(roots, root, tip, "main")
	if err != nil || f.Mode != ModeHuman || f.Layer != "person" || f.Withheld {
		t.Fatalf("the gate reads %+v, %v; want the person's human", f, err)
	}

	writeMachineConfig(t, humanConfig)
	writeEntries(t, entry{Email: personEmail, Root: sha, Forge: personForge, Mode: ModeAuto})
	f, err = readFacilitatorAt(roots, root, tip, "main")
	if err != nil || f.Mode != ModeHuman || f.Layer != "person" || !f.Withheld || !strings.Contains(f.Origin, "opened") {
		t.Fatalf("the gate reads %+v, %v; want human, saying a person's own auto waits on who opened the pull request", f, err)
	}
	if AutoAt(roots, root, tip, "main") {
		t.Error("the gate's AutoAt took a person's own auto")
	}
}

// TestTheRowSaysYoursHere (criteria 15 and 27): the row names the person's own
// for this project as `yours, here`, matched on what the prompt hook kept with
// its reading (the root, the name, and whether git's identity was the
// record's), asking git nothing.
func TestTheRowSaysYoursHere(t *testing.T) {
	repo, root, sha := personRepo(t, "")
	writeMachineConfig(t, humanConfig)
	writeEntries(t, entry{Email: personEmail, Root: sha, Forge: personForge, Mode: ModeAuto})
	kept(t, root)
	calls := noGit(t)
	m, note := KeptMerging(context.Background(), root)
	if m.State != statusline.MergingAuto || m.Whose != statusline.WhosePerson || note != "" {
		t.Fatalf("the row reads %+v, note %q; want auto, yours, here", m, note)
	}
	if n := calls(); n != 0 {
		t.Errorf("the row started %d gits for the mode", n)
	}
	t.Setenv("PATH", strings.SplitN(os.Getenv("PATH"), string(os.PathListSeparator), 2)[1])
	repo.Git("config", "user.email", "other@example.invalid")
	if _, err := AutoForSessions(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	if m, _ := KeptMerging(context.Background(), root); m.State != statusline.MergingHuman || m.Whose != statusline.WhosePerson {
		t.Errorf("with git acting as another the row reads %+v; want human, yours, here", m)
	}
}
