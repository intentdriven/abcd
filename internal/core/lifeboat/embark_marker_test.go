package lifeboat

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"testing"

	"github.com/intentdriven/abcd/internal/core/ahoy"
)

// targetFiles lists every regular file and link under root, as slash paths
// relative to it, so a before/after comparison shows exactly which files a run
// added. The lock files the record stores take while embark writes (a dotted
// name ending .lock) are runtime state, not content, and are left out.
func targetFiles(t *testing.T, root string) []string {
	t.Helper()
	var rels []string
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if name := d.Name(); strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".lock") {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		rels = append(rels, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(rels)
	return rels
}

// saveDocsTarget writes a target's .abcd/config.json the way setup saves it.
func saveDocsTarget(t *testing.T, target, value string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(target, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"docs": {"target": "` + value + `"}}` + "\n"
	if err := os.WriteFile(filepath.Join(target, ".abcd", "config.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestEmbarkPlantsInAgentsMD is A2 (itd-2610030814013772): a lifeboat embarked
// into a project holding only AGENTS.md, with no setup choice saved, plants
// abcd's block into AGENTS.md, names it in MarkerResult.Target, creates no
// CLAUDE.md, and adds no file beyond the records it planned. The probe
// predicts the same.
func TestEmbarkPlantsInAgentsMD(t *testing.T) {
	dest := packSource(t, embarkableSourceFixture(t))
	target := t.TempDir()
	if err := os.WriteFile(filepath.Join(target, "AGENTS.md"), []byte("# Target\n\nRun make check first.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := targetFiles(t, target)

	plan, err := EmbarkProbe(dest, target)
	if err != nil {
		t.Fatalf("EmbarkProbe: %v", err)
	}
	if plan.Marker.Target != "AGENTS.md" || plan.Marker.Action != MarkerActionInstall {
		t.Errorf("probe marker = %+v, want AGENTS.md install", plan.Marker)
	}

	res, err := EmbarkFrom(dest, target)
	if err != nil {
		t.Fatalf("EmbarkFrom: %v", err)
	}
	if res.Marker.Target != "AGENTS.md" || res.Marker.Action != MarkerActionInstall || !res.Marker.Changed {
		t.Errorf("from marker = %+v, want AGENTS.md install changed", res.Marker)
	}
	if changed, err := ahoy.EnsureMarker(filepath.Join(target, "AGENTS.md"), true); err != nil || changed {
		t.Errorf("AGENTS.md does not carry the current block: changed=%v err=%v", changed, err)
	}
	data, err := os.ReadFile(filepath.Join(target, "AGENTS.md"))
	if err != nil || !bytes.Contains(data, []byte("# Target\n")) || !bytes.Contains(data, []byte("Run make check first.\n")) {
		t.Errorf("AGENTS.md lost the owner's words: %q (err %v)", data, err)
	}
	if _, err := os.Lstat(filepath.Join(target, "CLAUDE.md")); !os.IsNotExist(err) {
		t.Errorf("embark created CLAUDE.md (lstat err %v)", err)
	}

	want := map[string]bool{}
	for _, f := range before {
		want[f] = true
	}
	for _, p := range plan.Planned {
		if p.Action == ActionCreate {
			want[p.TargetPath] = true
		}
	}
	after := targetFiles(t, target)
	if len(after) != len(want) {
		t.Errorf("target holds %d files, want %d (the AGENTS.md it had plus the planned records):\n%v", len(after), len(want), after)
	}
	for _, f := range after {
		if !want[f] {
			t.Errorf("embark added %s, which is neither a planned record nor AGENTS.md", f)
		}
	}
}

// TestEmbarkFollowsTheChosenTarget holds every docs.target a project's setup
// can have saved: agents_md plants into AGENTS.md; skip and the two retired
// values plant nowhere, say why, and still land the records; a settings file
// that is malformed or cannot be read plants nowhere; a value setup reads as
// unset is no choice, so the block goes to AGENTS.md. The probe predicts each
// outcome, and a CLAUDE.md already in the target is never written.
func TestEmbarkFollowsTheChosenTarget(t *testing.T) {
	dest := packSource(t, embarkableSourceFixture(t))
	skipNote := "this project's setup chose no conventions file for abcd's block"
	claudeWhy, _ := ahoy.RetiredDocsTarget("claude_md")
	bothWhy, _ := ahoy.RetiredDocsTarget("both")
	if claudeWhy == "" || bothWhy == "" {
		t.Fatal("RetiredDocsTarget gives no explanation for claude_md or both")
	}
	cases := []struct {
		name       string
		config     string // raw .abcd/config.json; "" writes none
		perm       os.FileMode
		wantTarget string
		wantAction MarkerAction
		wantNote   string // "" asserts no note; "*" asserts any non-empty note
	}{
		{"agents_md", `{"docs": {"target": "agents_md"}}`, 0o644, "AGENTS.md", MarkerActionInstall, ""},
		{"skip", `{"docs": {"target": "skip"}}`, 0o644, "", MarkerActionSkip, skipNote},
		{"retired claude_md", `{"docs": {"target": "claude_md"}}`, 0o644, "", MarkerActionSkip, claudeWhy},
		{"retired both", `{"docs": {"target": "both"}}`, 0o644, "", MarkerActionSkip, bothWhy},
		{"no settings file", "", 0o644, "AGENTS.md", MarkerActionInstall, ""},
		{"settings without docs.target", `{"docs": {}}`, 0o644, "AGENTS.md", MarkerActionInstall, ""},
		{"a value setup reads as unset", `{"docs": {"target": "elsewhere"}}`, 0o644, "AGENTS.md", MarkerActionInstall, ""},
		{"malformed settings", `{"docs": `, 0o644, "", MarkerActionSkip, "*"},
		{"unreadable settings", `{"docs": {"target": "agents_md"}}`, 0o000, "", MarkerActionSkip, "*"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			target := t.TempDir()
			if tc.config != "" {
				if err := os.MkdirAll(filepath.Join(target, ".abcd"), 0o755); err != nil {
					t.Fatal(err)
				}
				cfg := filepath.Join(target, ".abcd", "config.json")
				if err := os.WriteFile(cfg, []byte(tc.config), tc.perm); err != nil {
					t.Fatal(err)
				}
				if tc.perm == 0o000 {
					if f, err := os.Open(cfg); err == nil {
						f.Close()
						t.Skip("a mode-000 file is readable here (running as root)")
					}
				}
			}
			// A CLAUDE.md holding abcd's block, as a retired target left it:
			// embark never writes it, whatever was chosen.
			claude := filepath.Join(target, "CLAUDE.md")
			if _, err := ahoy.EnsureMarker(claude, false); err != nil {
				t.Fatal(err)
			}
			claudeBefore, _ := os.ReadFile(claude)
			check := func(label string, m MarkerResult) {
				t.Helper()
				if m.Target != tc.wantTarget || m.Action != tc.wantAction {
					t.Errorf("%s marker = %+v, want target %q action %s", label, m, tc.wantTarget, tc.wantAction)
				}
				switch tc.wantNote {
				case "":
					if m.Note != "" {
						t.Errorf("%s marker note = %q, want none", label, m.Note)
					}
				case "*":
					if m.Note == "" {
						t.Errorf("%s marker gives no note for a skip", label)
					}
				default:
					if m.Note != tc.wantNote {
						t.Errorf("%s marker note = %q, want %q", label, m.Note, tc.wantNote)
					}
				}
			}

			plan, err := EmbarkProbe(dest, target)
			if err != nil {
				t.Fatalf("EmbarkProbe: %v", err)
			}
			check("probe", plan.Marker)
			res, err := EmbarkFrom(dest, target)
			if err != nil {
				t.Fatalf("EmbarkFrom: %v", err)
			}
			check("from", res.Marker)
			if res.Written == 0 {
				t.Error("the records did not land")
			}

			_, agentsErr := os.Lstat(filepath.Join(target, "AGENTS.md"))
			if tc.wantTarget == "AGENTS.md" {
				if changed, err := ahoy.EnsureMarker(filepath.Join(target, "AGENTS.md"), true); err != nil || changed {
					t.Errorf("AGENTS.md does not carry the current block: changed=%v err=%v", changed, err)
				}
			} else if !os.IsNotExist(agentsErr) {
				t.Errorf("a skipped marker created AGENTS.md (lstat err %v)", agentsErr)
			}
			if after, _ := os.ReadFile(claude); !bytes.Equal(after, claudeBefore) {
				t.Error("embark wrote CLAUDE.md")
			}
		})
	}
}

// TestEmbarkMarkerNeverWritesThroughALink holds the chosen file to the same
// guard it had as CLAUDE.md: an AGENTS.md that is a link, or not a regular
// file, is skipped with a note, its target untouched, and the records land.
func TestEmbarkMarkerNeverWritesThroughALink(t *testing.T) {
	dest := packSource(t, embarkableSourceFixture(t))

	t.Run("link", func(t *testing.T) {
		target := t.TempDir()
		outside := filepath.Join(t.TempDir(), "elsewhere.md")
		if err := os.WriteFile(outside, []byte("# Not the target's\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(target, "AGENTS.md")); err != nil {
			t.Skipf("symlink: %v", err)
		}
		plan, err := EmbarkProbe(dest, target)
		if err != nil {
			t.Fatal(err)
		}
		res, err := EmbarkFrom(dest, target)
		if err != nil {
			t.Fatal(err)
		}
		for label, m := range map[string]MarkerResult{"probe": plan.Marker, "from": res.Marker} {
			if m.Target != "AGENTS.md" || m.Action != MarkerActionSkip || m.Changed || m.Note == "" {
				t.Errorf("%s marker = %+v, want AGENTS.md skip unchanged with a note", label, m)
			}
		}
		if got, _ := os.ReadFile(outside); string(got) != "# Not the target's\n" {
			t.Errorf("embark wrote through the link: %q", got)
		}
		if res.Written == 0 {
			t.Error("the records did not land")
		}
	})

	// A chosen AGENTS.md the write cannot read is refused by the write, so the
	// probe must predict that same refusal, not an install: the plan a person
	// approves names the action the run then takes (iss-2610032048280232).
	for _, tc := range []struct {
		name string
		make func(t *testing.T, path string)
	}{
		{"a directory", func(t *testing.T, path string) {
			if err := os.Mkdir(path, 0o755); err != nil {
				t.Fatal(err)
			}
		}},
		{"a FIFO", func(t *testing.T, path string) {
			if err := syscall.Mkfifo(path, 0o644); err != nil {
				t.Skipf("mkfifo unsupported: %v", err)
			}
		}},
		{"an unreadable file", func(t *testing.T, path string) {
			if err := os.WriteFile(path, []byte("# The target's own\n"), 0o000); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chmod(path, 0o644) })
			if f, err := os.Open(path); err == nil {
				f.Close()
				t.Skip("a mode-000 file is readable here (running as root)")
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			target := t.TempDir()
			tc.make(t, filepath.Join(target, "AGENTS.md"))
			plan, err := EmbarkProbe(dest, target)
			if err != nil {
				t.Fatal(err)
			}
			res, err := EmbarkFrom(dest, target)
			if err != nil {
				t.Fatal(err)
			}
			if res.Marker.Target != "AGENTS.md" || res.Marker.Action != MarkerActionSkip || res.Marker.Changed || res.Marker.Note == "" {
				t.Errorf("from marker = %+v, want AGENTS.md skip unchanged with a note", res.Marker)
			}
			if plan.Marker != res.Marker {
				t.Errorf("probe marker = %+v, but from gave %+v: the probe mispredicts", plan.Marker, res.Marker)
			}
			if res.Written == 0 {
				t.Error("the records did not land")
			}
		})
	}
}

// TestEmbarkRendersASkippedMarkerWithoutAFile holds the two human renders to
// a skip with no file chosen: neither names a file, and both carry the note.
func TestEmbarkRendersASkippedMarkerWithoutAFile(t *testing.T) {
	dest := packSource(t, embarkableSourceFixture(t))
	target := t.TempDir()
	saveDocsTarget(t, target, "skip")
	plan, err := EmbarkProbe(dest, target)
	if err != nil {
		t.Fatal(err)
	}
	res, err := EmbarkFrom(dest, target)
	if err != nil {
		t.Fatal(err)
	}
	note := "this project's setup chose no conventions file for abcd's block"
	for label, out := range map[string]string{"probe": plan.Render(), "from": res.Render()} {
		if !strings.Contains(out, note) {
			t.Errorf("%s render omits the skip note:\n%s", label, out)
		}
		for _, f := range []string{"AGENTS.md", "CLAUDE.md"} {
			if strings.Contains(out, f) {
				t.Errorf("%s render names %s for a marker that chose no file:\n%s", label, f, out)
			}
		}
	}
}

// TestEmbarkProbePredictsAnUnwritableRoot (iss-2610032202263648): a target
// whose root cannot take a new file, with a writable .abcd/, lets the records
// land and refuses the marker, whose write creates its lock and its temporary
// file beside the conventions file. The probe predicts that refusal, word for
// word, whether the file is absent or already holds the current block, and
// the note names no absolute path.
func TestEmbarkProbePredictsAnUnwritableRoot(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can write a mode-0555 folder")
	}
	dest := packSource(t, embarkableSourceFixture(t))
	for _, tc := range []struct {
		name    string
		current bool
	}{{"no AGENTS.md", false}, {"AGENTS.md holding the current block", true}} {
		t.Run(tc.name, func(t *testing.T) {
			target := t.TempDir()
			if err := os.MkdirAll(filepath.Join(target, ".abcd"), 0o755); err != nil {
				t.Fatal(err)
			}
			if tc.current {
				if _, err := ahoy.EnsureMarker(filepath.Join(target, "AGENTS.md"), false); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Chmod(target, 0o555); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chmod(target, 0o755) })
			plan, err := EmbarkProbe(dest, target)
			if err != nil {
				t.Fatal(err)
			}
			res, err := EmbarkFrom(dest, target)
			if err != nil {
				t.Fatal(err)
			}
			if res.Marker.Target != "AGENTS.md" || res.Marker.Action != MarkerActionSkip || res.Marker.Changed || res.Marker.Note == "" {
				t.Errorf("from marker = %+v, want AGENTS.md skip unchanged with a note", res.Marker)
			}
			if plan.Marker != res.Marker {
				t.Errorf("probe marker = %+v, but from gave %+v: the probe mispredicts", plan.Marker, res.Marker)
			}
			if strings.Contains(res.Marker.Note, target) {
				t.Errorf("the note carries the target's absolute path: %q", res.Marker.Note)
			}
			if res.Written == 0 {
				t.Error("the records did not land")
			}
		})
	}
}
