package landing

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/guard"
	"github.com/intentdriven/abcd/internal/core/identity"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/statusline"
	"github.com/intentdriven/abcd/internal/gittest"
)

// Two places decide the facilitator's mode (itd-2611252049031692, the product
// owner's ruling of 2026-11-26): what the project commits on its default
// branch, and otherwise the person's own global setting in
// ~/.abcd.noindex/config.json, read only under the home the account database
// names. With neither, a person is the facilitator. There is no setting of a
// person's for one project alone.

// readShown is the board's and the prompt line's reading of root.
func readShown(t *testing.T, root string) Facilitator {
	t.Helper()
	f, err := readDefaultFacilitatorContext(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// TestTheProjectDecidesThenThePersonsGlobalSetting is the table over the
// project's committed value and the person's global setting, each human, auto
// or absent: the project's value decides; with none, the global setting; with
// neither, human. Every reader that asks the default branch gives the one
// answer: the board's reading, the gate's, and the prompt line's.
func TestTheProjectDecidesThenThePersonsGlobalSetting(t *testing.T) {
	values := []string{ModeHuman, ModeAuto, ""}
	config := map[string]string{ModeHuman: humanConfig, ModeAuto: autoConfig, "": ""}
	for _, project := range values {
		for _, global := range values {
			t.Run("project "+project+", global "+global, func(t *testing.T) {
				repo, root := mergingRepo(t, config[project])
				if global != "" {
					writeMachineConfig(t, config[global])
				}
				mode, layer := ModeHuman, layered.Bundled.String()
				switch {
				case project != "":
					mode, layer = project, layered.Repo.String()
				case global != "":
					mode, layer = global, layered.Machine.String()
				}
				if f := readShown(t, root); f.Mode != mode || f.Layer != layer {
					t.Errorf("the board reads %+v; want %s from the %s layer", f, mode, layer)
				}
				gate, err := readFacilitatorAt(layered.Roots{Home: os.Getenv("HOME")}, root, repo.Git("rev-parse", "HEAD"), "main")
				if err != nil || gate.Mode != mode || gate.Layer != layer {
					t.Errorf("the gate reads %+v, %v; want %s from the %s layer", gate, err, mode, layer)
				}
				if f, err := AutoForSessions(context.Background(), root); err != nil || f.Mode != mode || f.Layer != layer {
					t.Errorf("the prompt line reads %+v, %v; want %s from the %s layer", f, err, mode, layer)
				}
			})
		}
	}
}

// writeProjectEntry writes the per-project file the branch once read,
// ~/.abcd.noindex/facilitator.json, holding one well-formed entry for the
// project whose root commit is root, for the person an identity record names
// and git authors as: everything such an entry needed to be selected.
func writeProjectEntry(t *testing.T, repo *gittest.Repo, root, mode string) {
	t.Helper()
	const name, email = "Ada Person", "ada@example.invalid"
	repo.Git("remote", "add", "origin", "https://forge.invalid/Owner/Repo.git")
	repo.Git("config", "user.name", name)
	repo.Git("config", "user.email", email)
	home := os.Getenv("HOME")
	if err := identity.WriteRecord(home, identity.MachineRecordRef(), identity.Record{Name: name, Email: email}); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(map[string]any{"schema_version": 1, "entries": []map[string]string{{
		"email": email, "root": gittest.RootCommit(t, root), "forge": "forge.invalid/Owner/Repo",
		"mode": mode, "set_at": "2026-11-26T01:00:00Z",
	}}})
	if err != nil {
		t.Fatal(err)
	}
	dir := abcdhome.Path(home)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "facilitator.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestASettingForOneProjectIsNotRead: there is no per-project personal
// setting (the ruling of 2026-11-26). A file of entries for one project in
// abcd's home, well formed and matching this project, this person and git's
// identity, decides nothing for any reader: where the project says nothing,
// the global setting does, and the kept reading keeps nothing of the entry.
func TestASettingForOneProjectIsNotRead(t *testing.T) {
	for _, c := range []struct{ global, entry string }{
		{ModeHuman, ModeAuto},
		{ModeAuto, ModeHuman},
		{"", ModeAuto},
	} {
		repo, root := mergingRepo(t, "")
		mode, layer := ModeHuman, layered.Bundled.String()
		if c.global != "" {
			writeMachineConfig(t, `{"facilitator":{"mode":"`+c.global+`"}}`)
			mode, layer = c.global, layered.Machine.String()
		}
		writeProjectEntry(t, repo, root, c.entry)
		if f := readShown(t, root); f.Mode != mode || f.Layer != layer {
			t.Errorf("global %q, entry %s: the board reads %+v; want %s from the %s layer", c.global, c.entry, f, mode, layer)
		}
		gate, err := readFacilitatorAt(layered.Roots{Home: os.Getenv("HOME")}, root, repo.Git("rev-parse", "HEAD"), "main")
		if err != nil || gate.Mode != mode || gate.Layer != layer {
			t.Errorf("global %q, entry %s: the gate reads %+v, %v; want %s from the %s layer", c.global, c.entry, gate, err, mode, layer)
		}
		raw, err := os.ReadFile(kept(t, root))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw), `"person"`) || !strings.Contains(string(raw), `"schema_version":1`) {
			t.Errorf("global %q, entry %s: the kept reading is %s; want schema 1 and nothing of the entry", c.global, c.entry, raw)
		}
		m, _ := KeptMerging(context.Background(), root)
		if want := statusline.WhoseComputer; layer == layered.Machine.String() && m.Whose != want {
			t.Errorf("global %q, entry %s: the row says %q, want %q", c.global, c.entry, m.Whose, want)
		}
	}
}

// TestUnderAnotherHomeAPersonHasNoSetting: the person's global setting is the
// one in the home the account database names; under any other HOME, a hosted
// session's among them, it is not read, so the mode is human where the
// project says nothing, and the project's value where it has one.
func TestUnderAnotherHomeAPersonHasNoSetting(t *testing.T) {
	for _, c := range []struct {
		project, mode, layer string
		row                  statusline.Merging
	}{
		{"", ModeHuman, foreignHomeLayer, statusline.Merging{State: statusline.MergingHuman, Whose: statusline.WhoseNotYourHome}},
		{autoConfig, ModeAuto, "repo", statusline.Merging{State: statusline.MergingAuto, Whose: statusline.WhoseProject}},
	} {
		repo, root := mergingRepo(t, c.project)
		writeMachineConfig(t, autoConfig)
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
		{"over the tip's tree, which holds no config file", "", func(t *testing.T, repo *gittest.Repo, tip string) {
			repo.Write(".abcd/config.json", autoConfig)
			repo.Commit("auto")
			other := repo.Git("rev-parse", "HEAD^{tree}")
			repo.Git("reset", "-q", "--hard", tip)
			repo.Git("replace", "-f", repo.Git("rev-parse", tip+"^{tree}"), other)
		}, "bundled"},
		{"over the tip itself, which holds no config file", "", func(t *testing.T, repo *gittest.Repo, tip string) {
			repo.Write(".abcd/config.json", autoConfig)
			repo.Commit("auto")
			other := repo.Git("rev-parse", "HEAD")
			repo.Git("reset", "-q", "--hard", tip)
			repo.Git("replace", "-f", tip, other)
		}, "bundled"},
	} {
		t.Run(c.name, func(t *testing.T) {
			repo, root := mergingRepo(t, c.project)
			// The tip holds a file, so its tree is never the empty tree: from
			// git 2.55 that tree is served from git's in-memory object source
			// by its own name, never through a replace ref, so a replacement
			// over it would not present auto, and the case would read nothing.
			repo.Write("README.md", "a file, so the tip's tree is not the empty tree\n")
			repo.Commit("a file")
			tip := repo.Git("rev-parse", "HEAD")
			if tree := repo.Git("rev-parse", tip+"^{tree}"); tree == repo.Git("hash-object", "-t", "tree", "/dev/null") {
				t.Fatalf("the tip's tree is the empty tree %s", tree)
			}
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

// TestTheGuardRefusesWritesOfTheFileTheResolverReads: the guard names the
// person's global settings file itself, since it cannot import this package,
// so the two are held together here; the per-project file it once also
// named is read by nothing, and is not held.
func TestTheGuardRefusesWritesOfTheFileTheResolverReads(t *testing.T) {
	home := t.TempDir()
	if p := abcdhome.Path(home, layered.Config.MachineRel); !guard.FacilitatorStorePath(p) {
		t.Errorf("guard.FacilitatorStorePath(%q) = false; the resolver reads that file", p)
	}
	if p := abcdhome.Path(home, "facilitator.json"); guard.FacilitatorStorePath(p) {
		t.Errorf("guard.FacilitatorStorePath(%q) = true; nothing reads that file", p)
	}
}
