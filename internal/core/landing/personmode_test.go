package landing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
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
	// personForge is the repository's whole address as an entry names it:
	// the host, lower-cased, and the full path, as origin's address spells
	// them, with .git dropped.
	personForge = "forge.invalid/Owner/Repo"
)

// personRepo is mergingRepo with an origin address naming personForge, git's
// identity set to the person's, and the person's identity record written in
// the test's HOME. It returns the repository, its root and its root commit.
func personRepo(t *testing.T, project string) (*gittest.Repo, string, string) {
	t.Helper()
	repo, root := mergingRepo(t, project)
	// Nothing in the test's own environment chooses git's identity, its
	// grafts or its shallow boundary: each case that wants one sets it.
	for _, kv := range os.Environ() {
		k, _, _ := strings.Cut(kv, "=")
		if strings.HasPrefix(k, "GIT_AUTHOR_") || strings.HasPrefix(k, "GIT_COMMITTER_") || slices.Contains(identityVars, k) {
			t.Setenv(k, "")
			os.Unsetenv(k)
		}
	}
	repo.Git("remote", "add", "origin", "https://forge.invalid/Owner/Repo.git")
	repo.Git("config", "user.name", personName)
	repo.Git("config", "user.email", personEmail)
	if err := identity.WriteRecord(os.Getenv("HOME"), identity.MachineRecordRef(), identity.Record{Name: personName, Email: personEmail}); err != nil {
		t.Fatal(err)
	}
	return repo, root, gittest.RootCommit(t, root)
}

// identityVars are the variables besides GIT_AUTHOR_* and GIT_COMMITTER_*
// that personRepo clears: what chooses git's identity, grafts or shallow
// boundary.
var identityVars = []string{"GIT_GRAFT_FILE", "GIT_SHALLOW_FILE", "GIT_CONFIG_GLOBAL", "GIT_CONFIG_SYSTEM", "XDG_CONFIG_HOME", "EMAIL"}

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
			writeEntries(t, entry{Email: personEmail, Root: sha, Forge: "forge.invalid/someone/Repo", Mode: ModeAuto})
		}, ModeAuto, "machine", "", false},
		{"a fork shares human", func(t *testing.T, _ *gittest.Repo, sha string) {
			writeEntries(t, entry{Email: personEmail, Root: sha, Forge: "forge.invalid/someone/Repo", Mode: ModeHuman})
		}, ModeHuman, "person", "", false},
		{"the committer differs alone", func(t *testing.T, repo *gittest.Repo, sha string) {
			writeEntries(t, entry{Email: personEmail, Root: sha, Forge: personForge, Mode: ModeAuto})
			repo.Git("config", "committer.email", "other@example.invalid")
		}, ModeHuman, "person", "commit here as", true},
		{"the name differs alone", func(t *testing.T, repo *gittest.Repo, sha string) {
			writeEntries(t, entry{Email: personEmail, Root: sha, Forge: personForge, Mode: ModeAuto})
			repo.Git("config", "user.name", "Someone Else")
		}, ModeHuman, "person", "Someone Else", true},
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
		row                  statusline.Merging
	}{
		{"", ModeHuman, foreignHomeLayer, statusline.Merging{State: statusline.MergingHuman, Whose: statusline.WhoseNotYourHome}},
		{autoConfig, ModeAuto, "repo", statusline.Merging{State: statusline.MergingAuto, Whose: statusline.WhoseProject}},
	} {
		repo, root, sha := personRepo(t, c.project)
		writeMachineConfig(t, autoConfig)
		writeEntries(t, entry{Email: personEmail, Root: sha, Forge: personForge, Mode: ModeAuto})
		other := t.TempDir()
		prev := AccountHome
		AccountHome = func() (string, error) { return other, nil }
		f := readShown(t, root)
		gate, gateErr := readFacilitatorAt(layered.Roots{Home: os.Getenv("HOME")}, root, repo.Git("rev-parse", "HEAD"), "main")
		kept(t, root)
		row, _ := KeptMerging(context.Background(), root)
		AccountHome = prev
		if f.Mode != c.mode || f.Layer != c.layer {
			t.Errorf("project %q under another HOME: reading = %+v; want %s from the %s layer", c.project, f, c.mode, c.layer)
		}
		if gateErr != nil || gate.Mode != c.mode || gate.Layer != c.layer {
			t.Errorf("project %q under another HOME: the gate reads %+v, %v; want %s from the %s layer", c.project, gate, gateErr, c.mode, c.layer)
		}
		if row != c.row {
			t.Errorf("project %q under another HOME: the row reads %+v, want %+v", c.project, row, c.row)
		}
		if c.project != "" {
			continue
		}
		// Not read on purpose, said in its own words: never as a file that
		// could not be read.
		if f.HomeRefused || !strings.Contains(f.Origin, "HOME") {
			t.Errorf("reading = %+v; want it to say HOME is not this account's, and not that a file was refused", f)
		}
		if says := f.Says(); !strings.Contains(says, "HOME is not your account's own") || strings.Contains(says, "could not") {
			t.Errorf("Says() = %q; want it to say HOME is not your account's own", says)
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

// mergeDatedRoot merges a new parentless commit dated at epoch into the
// current branch and returns it, so the branch holds one more first commit.
func mergeDatedRoot(t *testing.T, repo *gittest.Repo, epoch int64) string {
	t.Helper()
	date := "@" + strconv.FormatInt(epoch, 10) + " +0000"
	git := func(args ...string) string {
		cmd := exec.Command("git", append([]string{"-C", repo.Root()}, args...)...)
		cmd.Env = append(repo.Env(), "GIT_AUTHOR_NAME=Fixture", "GIT_AUTHOR_EMAIL=fixture@example.invalid",
			"GIT_COMMITTER_NAME=Fixture", "GIT_COMMITTER_EMAIL=fixture@example.invalid",
			"GIT_AUTHOR_DATE="+date, "GIT_COMMITTER_DATE="+date)
		var stderr strings.Builder
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, stderr.String())
		}
		return strings.TrimSpace(string(out))
	}
	tree := git("hash-object", "-w", "-t", "tree", os.DevNull)
	other := git("commit-tree", tree, "-m", "an unrelated first commit")
	head := git("rev-parse", "HEAD")
	merge := git("commit-tree", head+"^{tree}", "-p", head, "-p", other, "-m", "merge an unrelated history")
	git("update-ref", "refs/heads/main", merge)
	return other
}

// TestTheProjectsFirstCommitsMustBeEstablished: where the person's file holds
// entries, the project's first commits are every parentless commit reachable
// from the default branch's tip. Where they cannot be established, the mode
// is human from the person layer, saying why, and never the computer's file.
// A human entry matches any of them; an auto entry only the one first commit
// of a project that has exactly one.
func TestTheProjectsFirstCommitsMustBeEstablished(t *testing.T) {
	for _, c := range []struct {
		name     string
		setup    func(t *testing.T, repo *gittest.Repo, root, sha string) string
		mode     string
		layer    string
		withheld bool
		says     string
	}{
		{"a clone of depth one", func(t *testing.T, repo *gittest.Repo, root, sha string) string {
			writeEntries(t, entry{Email: personEmail, Root: sha, Forge: personForge, Mode: ModeAuto})
			repo.Write("next.txt", "next")
			repo.Commit("next")
			dst := filepath.Join(t.TempDir(), "clone")
			repo.Git("clone", "-q", "--depth", "1", "file://"+root, dst)
			repo.Git("-C", dst, "remote", "set-url", "origin", "https://forge.invalid/Owner/Repo.git")
			repo.Git("-C", dst, "config", "user.name", personName)
			repo.Git("-C", dst, "config", "user.email", personEmail)
			return dst
		}, ModeHuman, "person", true, "shallow"},
		{"a grafted boundary", func(t *testing.T, repo *gittest.Repo, root, sha string) string {
			writeEntries(t, entry{Email: personEmail, Root: sha, Forge: personForge, Mode: ModeAuto})
			repo.Write("next.txt", "next")
			repo.Commit("next")
			if err := os.WriteFile(filepath.Join(root, ".git", "info", "grafts"), []byte(repo.Git("rev-parse", "HEAD")+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			return root
		}, ModeHuman, "person", true, "graft"},
		{"a graft file the environment names", func(t *testing.T, repo *gittest.Repo, root, sha string) string {
			writeEntries(t, entry{Email: personEmail, Root: sha, Forge: personForge, Mode: ModeAuto})
			repo.Write("next.txt", "next")
			repo.Commit("next")
			p := filepath.Join(t.TempDir(), "grafts")
			if err := os.WriteFile(p, []byte(repo.Git("rev-parse", "HEAD")+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("GIT_GRAFT_FILE", p)
			return root
		}, ModeHuman, "person", true, "GIT_GRAFT_FILE"},
		{"a shallow file the environment names", func(t *testing.T, repo *gittest.Repo, root, sha string) string {
			writeEntries(t, entry{Email: personEmail, Root: sha, Forge: personForge, Mode: ModeAuto})
			repo.Write("next.txt", "next")
			repo.Commit("next")
			p := filepath.Join(t.TempDir(), "shallow")
			if err := os.WriteFile(p, []byte(repo.Git("rev-parse", "HEAD")+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("GIT_SHALLOW_FILE", p)
			return root
		}, ModeHuman, "person", true, "GIT_SHALLOW_FILE"},
		{"an auto entry for an unrelated first commit dated 1971", func(t *testing.T, repo *gittest.Repo, root, _ string) string {
			old := mergeDatedRoot(t, repo, 31536000)
			writeEntries(t, entry{Email: personEmail, Root: old, Forge: personForge, Mode: ModeAuto})
			return root
		}, ModeHuman, "bundled", false, ""},
		{"an auto entry for the project's own first commit, beside an unrelated one", func(t *testing.T, repo *gittest.Repo, root, sha string) string {
			mergeDatedRoot(t, repo, 31536000)
			writeEntries(t, entry{Email: personEmail, Root: sha, Forge: personForge, Mode: ModeAuto})
			return root
		}, ModeHuman, "bundled", false, ""},
		{"a human entry for the project's own first commit, beside an unrelated one dated 1971", func(t *testing.T, repo *gittest.Repo, root, sha string) string {
			writeMachineConfig(t, autoConfig)
			mergeDatedRoot(t, repo, 31536000)
			writeEntries(t, entry{Email: personEmail, Root: sha, Forge: personForge, Mode: ModeHuman})
			return root
		}, ModeHuman, "person", false, ""},
		{"a human entry for an unrelated first commit dated 1971", func(t *testing.T, repo *gittest.Repo, root, _ string) string {
			old := mergeDatedRoot(t, repo, 31536000)
			writeEntries(t, entry{Email: personEmail, Root: old, Forge: personForge, Mode: ModeHuman})
			return root
		}, ModeHuman, "person", false, ""},
		{"the walk past its cap", func(t *testing.T, repo *gittest.Repo, root, sha string) string {
			writeEntries(t, entry{Email: personEmail, Root: sha, Forge: personForge, Mode: ModeAuto})
			mergeDatedRoot(t, repo, 31536000)
			prev := maxRootBytes
			maxRootBytes = 50
			t.Cleanup(func() { maxRootBytes = prev })
			return root
		}, ModeHuman, "person", true, "first commit"},
		{"git failing in the walk", func(t *testing.T, _ *gittest.Repo, root, sha string) string {
			writeEntries(t, entry{Email: personEmail, Root: sha, Forge: personForge, Mode: ModeAuto})
			swapListRoots(t, func(context.Context, string, string) (string, error) { return "", errors.New("exit status 128") })
			return root
		}, ModeHuman, "person", true, "exit status 128"},
		{"git naming no first commit", func(t *testing.T, _ *gittest.Repo, root, sha string) string {
			writeEntries(t, entry{Email: personEmail, Root: sha, Forge: personForge, Mode: ModeAuto})
			swapListRoots(t, func(context.Context, string, string) (string, error) { return "", nil })
			return root
		}, ModeHuman, "person", true, "first commit"},
		{"a hand-built commit-graph giving the tip no parents", func(t *testing.T, repo *gittest.Repo, root, sha string) string {
			writeMachineConfig(t, autoConfig)
			writeEntries(t, entry{Email: personEmail, Root: sha, Forge: personForge, Mode: ModeHuman})
			repo.Write("next.txt", "next")
			repo.Commit("next")
			repo.Git("commit-graph", "write", "--reachable")
			gittest.ForgeCommitGraphRoot(t, filepath.Join(root, ".git"), repo.Git("rev-parse", "HEAD"))
			return root
		}, ModeHuman, "person", false, ""},
		{"grafts written between the checks and the walk", func(t *testing.T, repo *gittest.Repo, root, sha string) string {
			writeMachineConfig(t, autoConfig)
			writeEntries(t, entry{Email: personEmail, Root: sha, Forge: personForge, Mode: ModeHuman})
			repo.Write("next.txt", "next")
			repo.Commit("next")
			walk := listRoots
			swapListRoots(t, func(ctx context.Context, dir, tip string) (string, error) {
				if err := os.WriteFile(filepath.Join(root, ".git", "info", "grafts"), []byte(tip+"\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				return walk(ctx, dir, tip)
			})
			return root
		}, ModeHuman, "person", false, ""},
		{"a shallow file written between the checks and the walk", func(t *testing.T, repo *gittest.Repo, root, sha string) string {
			writeMachineConfig(t, autoConfig)
			writeEntries(t, entry{Email: personEmail, Root: sha, Forge: personForge, Mode: ModeHuman})
			repo.Write("next.txt", "next")
			repo.Commit("next")
			walk := listRoots
			swapListRoots(t, func(ctx context.Context, dir, tip string) (string, error) {
				if err := os.WriteFile(filepath.Join(root, ".git", "shallow"), []byte(tip+"\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				return walk(ctx, dir, tip)
			})
			return root
		}, ModeHuman, "person", false, ""},
		{"no entries: the walk is not asked", func(t *testing.T, _ *gittest.Repo, root, _ string) string {
			swapListRoots(t, func(context.Context, string, string) (string, error) { return "", errors.New("asked") })
			return root
		}, ModeHuman, "bundled", false, ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			repo, root, sha := personRepo(t, "")
			read := c.setup(t, repo, root, sha)
			f := readShown(t, read)
			if f.Mode != c.mode || f.Layer != c.layer || f.Withheld != c.withheld {
				t.Fatalf("reading = %+v; want %s from the %s layer, withheld %t", f, c.mode, c.layer, c.withheld)
			}
			if c.says != "" && !strings.Contains(f.Origin, c.says) {
				t.Errorf("origin = %q, want it to name %q", f.Origin, c.says)
			}
		})
	}
}

// swapListRoots substitutes the root walk for one test.
func swapListRoots(t *testing.T, fn func(ctx context.Context, repo, tip string) (string, error)) {
	t.Helper()
	prev := listRoots
	listRoots = fn
	t.Cleanup(func() { listRoots = prev })
}

// TestADeadlineInTheRootWalkIsAnError: a walk the deadline ends is not a
// reading. Every reader returns the deadline's error, so the prompt line sends
// nothing, the board says the mode was not read, and the gate does not arm.
func TestADeadlineInTheRootWalkIsAnError(t *testing.T) {
	repo, root, sha := personRepo(t, "")
	writeMachineConfig(t, autoConfig)
	writeEntries(t, entry{Email: personEmail, Root: sha, Forge: personForge, Mode: ModeAuto})
	swapListRoots(t, func(context.Context, string, string) (string, error) {
		return "", fmt.Errorf("git rev-list: %w", context.DeadlineExceeded)
	})
	if f, err := AutoForSessions(context.Background(), root); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("AutoForSessions = %+v, %v; want the deadline's error", f, err)
	}
	if f, err := readDefaultFacilitatorContext(context.Background(), root); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("the board's reading = %+v, %v; want the deadline's error", f, err)
	}
	roots := layered.Roots{Home: os.Getenv("HOME")}
	tip := repo.Git("rev-parse", "HEAD")
	if f, err := readFacilitatorAt(roots, root, tip, "main"); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("the gate reads %+v, %v; want the deadline's error", f, err)
	}
	if AutoAt(roots, root, tip, "main") {
		t.Error("the gate armed on a walk the deadline ended")
	}
}

// TestARepositorysNameIsItsWholeAddress: an auto entry matches only the
// repository's whole address, the host lower-cased and the full path, read
// alike from git's https, ssh and scp spellings with .git and a trailing
// slash dropped.
func TestARepositorysNameIsItsWholeAddress(t *testing.T) {
	for _, c := range []struct {
		name, origin, forge, layer string
	}{
		{"https", "https://forge.invalid/Owner/Repo.git", personForge, "person"},
		{"scp form, host in capitals", "git@FORGE.invalid:Owner/Repo.git", personForge, "person"},
		{"ssh url", "ssh://git@forge.invalid/Owner/Repo", personForge, "person"},
		{"a trailing slash", "https://forge.invalid/Owner/Repo/", personForge, "person"},
		{"the same owner and name on another host", "https://other.invalid/Owner/Repo.git", personForge, "machine"},
		{"an entry naming no host, on another host", "https://other.invalid/owner/repo.git", "owner/repo", "machine"},
		{"a deeper path ending in the owner and name", "https://forge.invalid/group/Owner/Repo.git", personForge, "machine"},
		{"an entry naming no host, on a deeper path", "https://forge.invalid/group/owner/repo.git", "owner/repo", "machine"},
		{"the path in another case", "https://forge.invalid/owner/repo.git", personForge, "machine"},
		{"no host", "file:///srv/Owner/Repo.git", personForge, "machine"},
		{"http, not https", "http://forge.invalid/Owner/Repo.git", personForge, "machine"},
		{"git's own protocol", "git://forge.invalid/Owner/Repo.git", personForge, "machine"},
		{"a % in the path", "https://forge.invalid/Owner/Re%70o.git", personForge, "machine"},
	} {
		t.Run(c.name, func(t *testing.T) {
			repo, root, sha := personRepo(t, "")
			writeMachineConfig(t, autoConfig)
			repo.Git("remote", "set-url", "origin", c.origin)
			writeEntries(t, entry{Email: personEmail, Root: sha, Forge: c.forge, Mode: ModeAuto})
			if f := readShown(t, root); f.Mode != ModeAuto || f.Layer != c.layer {
				t.Fatalf("origin %s, entry %s: reading = %+v; want auto from the %s layer", c.origin, c.forge, f, c.layer)
			}
		})
	}
}

// TestAnAutoEntryWaitsWhileTheEnvironmentSetsGitsIdentity: while any
// GIT_AUTHOR_* or GIT_COMMITTER_* variable is set, an auto entry is not used,
// live or on the row; a human entry still is.
func TestAnAutoEntryWaitsWhileTheEnvironmentSetsGitsIdentity(t *testing.T) {
	_, root, sha := personRepo(t, "")
	writeMachineConfig(t, autoConfig)
	writeEntries(t, entry{Email: personEmail, Root: sha, Forge: personForge, Mode: ModeAuto})
	t.Setenv("GIT_AUTHOR_NAME", personName)
	if f := readShown(t, root); f.Mode != ModeHuman || f.Layer != "person" || !f.Withheld || !strings.Contains(f.Origin, "environment") {
		t.Fatalf("reading = %+v; want human from the person layer, saying the environment sets git's identity", f)
	}
	kept(t, root)
	if m, _ := KeptMerging(context.Background(), root); m.State != statusline.MergingHuman || m.Whose != statusline.WhosePerson {
		t.Errorf("the row reads %+v; want human, yours, here", m)
	}

	writeEntries(t, entry{Email: personEmail, Root: sha, Forge: personForge, Mode: ModeHuman})
	os.Unsetenv("GIT_AUTHOR_NAME")
	t.Setenv("GIT_COMMITTER_EMAIL", personEmail)
	if f := readShown(t, root); f.Mode != ModeHuman || f.Layer != "person" || f.Withheld {
		t.Fatalf("reading = %+v; want the person's own human", f)
	}
}

// TestTheProjectsValueIsReadAsStored: the gate, the board and the prompt line
// read the project's committed facilitator.mode from the tip as the object
// store holds it, so a refs/replace entry over the file's blob, the tip's
// tree or the tip itself is not read in its place.
func TestTheProjectsValueIsReadAsStored(t *testing.T) {
	for _, c := range []struct {
		name    string
		project string
		replace func(t *testing.T, repo *gittest.Repo, tip string)
		layer   string
	}{
		{"over the tip's config blob", humanConfig, func(t *testing.T, repo *gittest.Repo, tip string) {
			p := filepath.Join(t.TempDir(), "auto.json")
			if err := os.WriteFile(p, []byte(autoConfig), 0o600); err != nil {
				t.Fatal(err)
			}
			repo.Git("replace", repo.Git("rev-parse", tip+":.abcd/config.json"), repo.Git("hash-object", "-w", p))
		}, "repo"},
		{"over the tip's tree, which holds no file", "", func(t *testing.T, repo *gittest.Repo, tip string) {
			repo.Write(".abcd/config.json", autoConfig)
			repo.Commit("auto")
			other := repo.Git("rev-parse", "HEAD^{tree}")
			repo.Git("reset", "-q", "--hard", tip)
			repo.Git("replace", "-f", repo.Git("rev-parse", tip+"^{tree}"), other)
		}, "bundled"},
		{"over the tip itself, which holds no file", "", func(t *testing.T, repo *gittest.Repo, tip string) {
			repo.Write(".abcd/config.json", autoConfig)
			repo.Commit("auto")
			other := repo.Git("rev-parse", "HEAD")
			repo.Git("reset", "-q", "--hard", tip)
			repo.Git("replace", "-f", tip, other)
		}, "bundled"},
	} {
		t.Run(c.name, func(t *testing.T) {
			repo, root := mergingRepo(t, c.project)
			tip := repo.Git("rev-parse", "HEAD")
			c.replace(t, repo, tip)
			if got := repo.Git("cat-file", "-p", tip+":.abcd/config.json"); !strings.Contains(got, ModeAuto) {
				t.Fatalf("the fixture's replace ref does not present auto: %q", got)
			}
			gate, err := readFacilitatorAt(layered.Roots{Home: os.Getenv("HOME")}, root, tip, "main")
			if err != nil || gate.Mode != ModeHuman || gate.Layer != c.layer {
				t.Errorf("the gate reads %+v, %v; want human from the %s layer, as stored", gate, err, c.layer)
			}
			if shown := readShown(t, root); shown.Mode != ModeHuman || shown.Layer != c.layer {
				t.Errorf("the board reads %+v; want human from the %s layer, as stored", shown, c.layer)
			}
			if f, err := AutoForSessions(context.Background(), root); err != nil || f.Mode == ModeAuto {
				t.Errorf("the prompt line reads %+v, %v; want not auto", f, err)
			}
		})
	}
}

// TestAMisspeltEntryIsSaidPlainly: an entry whose forge is not a repository's
// whole address, whose root is not a commit's full object name in lower
// case, or that sets auto and names no address, refuses the file with a plain
// reason.
func TestAMisspeltEntryIsSaidPlainly(t *testing.T) {
	for _, c := range []struct {
		name string
		e    func(sha string) entry
		says string
	}{
		{"a forge spelt as an address git reads", func(sha string) entry {
			return entry{Email: personEmail, Root: sha, Forge: "https://forge.invalid/Owner/Repo", Mode: ModeHuman}
		}, "not a repository's whole address"},
		{"a root in upper case", func(sha string) entry {
			return entry{Email: personEmail, Root: strings.ToUpper(sha), Forge: personForge, Mode: ModeHuman}
		}, "lower-case"},
		{"auto with no forge", func(sha string) entry {
			return entry{Email: personEmail, Root: sha, Mode: ModeAuto}
		}, "names no address"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, root, sha := personRepo(t, "")
			writeMachineConfig(t, autoConfig)
			writeEntries(t, c.e(sha))
			f := readShown(t, root)
			if f.Mode != ModeHuman || f.Layer != "person" || !f.Withheld || !strings.Contains(f.Origin, c.says) {
				t.Fatalf("reading = %+v; want human from the person layer, saying %q", f, c.says)
			}
		})
	}
}

// TestTheAddressIsTheOneGitWouldFetch: the repository's address is origin's
// as git resolves it, the person's own url.*.insteadOf applied; an origin
// with more than one url names no address, so no auto entry matches.
func TestTheAddressIsTheOneGitWouldFetch(t *testing.T) {
	t.Run("insteadOf in the person's own settings", func(t *testing.T) {
		repo, root, sha := personRepo(t, "")
		writeMachineConfig(t, humanConfig)
		repo.Git("remote", "set-url", "origin", "fi:Owner/Repo")
		global := "[url \"https://forge.invalid/\"]\n\tinsteadOf = fi:\n"
		if err := os.WriteFile(filepath.Join(os.Getenv("HOME"), ".gitconfig"), []byte(global), 0o600); err != nil {
			t.Fatal(err)
		}
		writeEntries(t, entry{Email: personEmail, Root: sha, Forge: personForge, Mode: ModeAuto})
		if f := readShown(t, root); f.Mode != ModeAuto || f.Layer != "person" {
			t.Fatalf("reading = %+v; want the person's auto at the address git would fetch", f)
		}
	})
	t.Run("a second url", func(t *testing.T) {
		repo, root, sha := personRepo(t, "")
		writeMachineConfig(t, humanConfig)
		repo.Git("config", "--add", "remote.origin.url", "https://forge.invalid/Owner/Repo.git")
		writeEntries(t, entry{Email: personEmail, Root: sha, Forge: personForge, Mode: ModeAuto})
		if f := readShown(t, root); f.Mode != ModeHuman || f.Layer != "machine" {
			t.Fatalf("reading = %+v; want no auto entry matched under two urls", f)
		}
	})
}

// TestAnAutoEntryWaitsWhileTheEnvironmentLocatesGitsSettings: while
// GIT_CONFIG_GLOBAL, GIT_CONFIG_SYSTEM, XDG_CONFIG_HOME or EMAIL is set,
// git's identity can come from elsewhere, so an auto entry is not used.
func TestAnAutoEntryWaitsWhileTheEnvironmentLocatesGitsSettings(t *testing.T) {
	for _, name := range []string{"GIT_CONFIG_GLOBAL", "GIT_CONFIG_SYSTEM", "XDG_CONFIG_HOME", "EMAIL"} {
		t.Run(name, func(t *testing.T) {
			_, root, sha := personRepo(t, "")
			writeMachineConfig(t, autoConfig)
			writeEntries(t, entry{Email: personEmail, Root: sha, Forge: personForge, Mode: ModeAuto})
			value := os.DevNull
			if name == "EMAIL" {
				value = personEmail
			} else if name == "XDG_CONFIG_HOME" {
				value = t.TempDir()
			}
			t.Setenv(name, value)
			if f := readShown(t, root); f.Mode != ModeHuman || f.Layer != "person" || !f.Withheld || !strings.Contains(f.Origin, name) {
				t.Fatalf("reading = %+v; want human from the person layer, naming %s", f, name)
			}
		})
	}
}
