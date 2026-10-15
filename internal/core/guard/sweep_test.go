package guard

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// The false-positive sweep (iss-2609281134544802, ruling AT).
//
// Each tightening of how the guard reads a value a line carries in a variable
// was weighed by the blocks and warns it adds on ordinary work. This is that
// measurement, committed so the next reading has to keep it: every command line
// the sweep holds goes through Check with the bundled registry, and the counts
// of blocks and warns are held under a ceiling. The sweep reads:
//
//   - testdata/corpus/repo-mined.txt, a line at a time;
//   - testdata/corpus/everyday-variables.txt, a line at a time;
//   - testdata/corpus/forge-posting.txt, a line at a time: benign posting gh
//     commands, which the second pass judges with the forge-text check on;
//   - every recipe line of the repository's Makefile, with make's `$$` read as
//     the shell's `$`;
//   - every line of scripts/*.sh, hooks/*.sh and .githooks/*, and each of those
//     files whole, as one text.
//
// Many lines block for reasons of their own — a script line read alone can
// open a here-document it never closes, and a test harness names the hazards
// it tests — so the ceilings are the measured counts, not zero. Raising one is
// allowed and never quiet: the test logs every block and warn, so the diff
// shows what started firing.
//
// Measured when the carried-value reading landed (iss-2609281134544802): 5,937
// inputs, 390 unparsable, 25 blocks and 29 warns both before the reading and
// after it, the same lines each time.
//
// Blocks raised to 26 when the installer's refuse_older landed
// (hooks/bootstrap.sh): its printf argument line, read alone, blocks as
// program-name-unknown, the same way say_refusal's argument line already did.
const (
	maxSweepBlocks = 26
	maxSweepWarns  = 29
)

func TestFalsePositiveSweepCeiling(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	var inputs []string
	inputs = append(inputs, readCorpus(t, filepath.Join("testdata", "corpus", "repo-mined.txt"))...)
	inputs = append(inputs, readCorpus(t, filepath.Join("testdata", "corpus", "everyday-variables.txt"))...)
	forgeLines := readCorpus(t, filepath.Join("testdata", "corpus", "forge-posting.txt"))
	if len(forgeLines) < 20 {
		t.Fatalf("the forge-posting corpus holds %d lines; it has gone missing", len(forgeLines))
	}
	inputs = append(inputs, forgeLines...)

	mk, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatalf("reading the Makefile: %v", err)
	}
	for _, line := range strings.Split(string(mk), "\n") {
		if !strings.HasPrefix(line, "\t") {
			continue
		}
		line = strings.TrimLeft(strings.TrimSpace(line), "@-")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		inputs = append(inputs, strings.ReplaceAll(line, "$$", "$"))
	}

	var files []string
	for _, glob := range []string{"scripts/*.sh", "hooks/*.sh", ".githooks/*"} {
		ms, _ := filepath.Glob(filepath.Join(root, glob))
		files = append(files, ms...)
	}
	sort.Strings(files)
	if len(files) < 15 {
		t.Fatalf("found %d scripts; the sweep's script corpus has gone missing", len(files))
	}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("reading %s: %v", f, err)
		}
		for _, line := range strings.Split(string(b), "\n") {
			if l := strings.TrimSpace(line); l != "" && !strings.HasPrefix(l, "#") {
				inputs = append(inputs, l)
			}
		}
		if len(b) <= maxCommandBytes {
			inputs = append(inputs, string(b))
		}
	}

	// Two passes: the bundled registry alone, and a registry loaded for a
	// repository that opted into the forge-text check (forge.go,
	// iss-2610142210511831), so the posting commands in the sweep are judged.
	// The armed pass is held to the same ceilings, and to no forge-text
	// verdict at all: every posting command the sweep holds is benign.
	armedRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(armedRoot, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(armedRoot, ".abcd", "config.json"), []byte(`{"outbound": {"forge": true}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	armed, err := Load(armedRoot)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	armed = armed.ReadingFrom(NewFiles(cwd))
	for _, pass := range []struct {
		name string
		r    Registry
	}{{"bundled", Defaults()}, {"forge-text armed", armed}} {
		t.Run(pass.name, func(t *testing.T) {
			sweepPass(t, pass.r, inputs)
		})
	}
}

// sweepPass checks every input against r and holds the counts to the
// ceilings; a forge-text verdict on any input fails it outright.
func sweepPass(t *testing.T, r Registry, inputs []string) {
	t.Helper()
	var allows, warns, blocks, unparsable, forge int
	for _, in := range inputs {
		d, err := r.Check(in)
		if err != nil {
			unparsable++
			continue
		}
		shown := in
		if len(shown) > 160 {
			shown = shown[:160] + "…"
		}
		if contains(d.Matches, forgeTextEntryID) || contains(d.Matches, forgeUnreadEntryID) {
			forge++
			t.Errorf("a forge-text verdict on ordinary work: %s %q", d.EntryID, shown)
		}
		switch d.Verdict {
		case VerdictAllow:
			allows++
		case VerdictWarn:
			warns++
			t.Logf("warn  %-28s %q", d.EntryID, shown)
		case VerdictBlock:
			blocks++
			t.Logf("block %-28s %q", d.EntryID, shown)
		}
	}
	t.Logf("sweep: %d inputs, %d allow, %d block, %d warn, %d unparsable, %d forge-text",
		len(inputs), allows, blocks, warns, unparsable, forge)
	if len(inputs) < 2000 {
		t.Fatalf("the sweep holds %d inputs; a corpus has gone missing", len(inputs))
	}
	if blocks > maxSweepBlocks {
		t.Errorf("the sweep blocks %d inputs, over the ceiling of %d: a reading now refuses ordinary work (the log names each)", blocks, maxSweepBlocks)
	}
	if warns > maxSweepWarns {
		t.Errorf("the sweep warns on %d inputs, over the ceiling of %d: a reading now warns on ordinary work (the log names each)", warns, maxSweepWarns)
	}
}
