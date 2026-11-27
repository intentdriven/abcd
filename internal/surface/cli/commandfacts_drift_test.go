package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/surface"
)

// commandfacts_drift_test.go — the drift gate on the derived copies of the
// command facts (itd-2610301346085246 criterion 3 and criterion 5;
// spc-2610301431075964 step 3). The per-verb JSON files and the two generated
// blocks of each command page are written by `make command-facts-sync` from
// CommandFacts; each test here fails, naming the file, when a committed copy
// differs from a fresh rendering. Every name carries one of the
// PRECHECK_AUDIT_TESTS words, so `make precheck` and `make preflight` run them.

// regenerateCommandFacts is the fix every failure here names.
const regenerateCommandFacts = "run `make command-facts-sync` and commit the result"

// committedFactsFiles reads every JSON file under CommandFactsDir, keyed by
// its repo-relative path.
func committedFactsFiles(t *testing.T) map[string][]byte {
	t.Helper()
	dir := filepath.Join(testRepoRoot(), filepath.FromSlash(CommandFactsDir))
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v; %s", CommandFactsDir, err, regenerateCommandFacts)
	}
	out := map[string][]byte{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		out[CommandFactsDir+"/"+e.Name()] = data
	}
	return out
}

// The JSON-file drift test: every file the facts render is committed byte for
// byte, and no committed file is one the facts no longer render.
func TestCommandFactsFilesMatchTheCommittedCopies(t *testing.T) {
	want, err := CommandFactsFiles(CommandFacts())
	if err != nil {
		t.Fatal(err)
	}
	got := committedFactsFiles(t)
	var paths []string
	for p := range want {
		paths = append(paths, p)
	}
	for p := range got {
		if _, ok := want[p]; !ok {
			paths = append(paths, p)
		}
	}
	sort.Strings(paths)
	for _, p := range paths {
		w, rendered := want[p]
		g, committed := got[p]
		switch {
		case !committed:
			t.Errorf("%s is not committed; %s", p, regenerateCommandFacts)
		case !rendered:
			t.Errorf("%s is committed for a verb the tree no longer has; %s", p, regenerateCommandFacts)
		case !bytes.Equal(w, g):
			t.Errorf("%s is stale: it no longer matches the command facts; %s", p, regenerateCommandFacts)
		}
	}
}

// The page-block drift test: every command page already carries both blocks
// as the facts render them, each in its place.
func TestCommandPageBlocksMatchTheCommittedPages(t *testing.T) {
	facts := CommandFacts()
	pages := commandFileBodies(t)
	if len(pages) == 0 {
		t.Fatalf("%s holds no page", CommandPagesDir)
	}
	for name, page := range pages {
		rel := CommandPagesDir + "/" + name + ".md"
		synced, err := SyncCommandPage([]byte(page), name, facts)
		if err != nil {
			t.Errorf("%s: %v", rel, err)
			continue
		}
		if string(synced) != page {
			t.Errorf("%s carries a stale or missing generated block; %s", rel, regenerateCommandFacts)
		}
	}
}

// Criterion 5: on every command page the first content after the H1 heading
// is the ask-the-binary block, it names that page's verb, and the page carries
// it once.
func TestEveryCommandPageOpensWithTheAskTheBinaryCall(t *testing.T) {
	for name, page := range commandFileBodies(t) {
		rel := CommandPagesDir + "/" + name + ".md"
		h := headingEnd(page)
		if h < 0 {
			t.Errorf("%s has no H1 heading", rel)
			continue
		}
		first := strings.TrimLeft(page[h:], "\n")
		if !strings.HasPrefix(first, AskTheBinaryBegin+"\n") {
			line, _, _ := strings.Cut(first, "\n")
			t.Errorf("%s: the first content after the H1 heading is %q, not the ask-the-binary block", rel, line)
			continue
		}
		block, _, ok := strings.Cut(first, GeneratedEnd)
		if !ok {
			t.Errorf("%s opens the ask-the-binary block and never closes it", rel)
			continue
		}
		if call := " explain " + name + "`"; !strings.Contains(block, call) {
			t.Errorf("%s: the ask-the-binary block does not name the page's verb as `…%s`:\n%s", rel, call, block)
		}
		if n := strings.Count(page, AskTheBinaryBegin); n != 1 {
			t.Errorf("%s carries the ask-the-binary block %d times", rel, n)
		}
	}
}

// Criterion 3's third test: `abcd explain <path> --json` is byte-equal to that
// command's slice of its committed JSON file, the file's document cut to the
// one command and encoded as the files are (EncodeCommandFacts, so with no
// HTML escaping), never with explain's own encoder, which would hold explain
// to itself, for every command a file holds, followed only by explain's own
// next_steps, which every successful result carries (spc-2610301431108175
// step 6). A file is decoded strictly, so a field explain does not print
// fails here too.
func TestExplainJSONIsTheCommittedFileSliceForEveryCommand(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	explainNext, ok := surface.NextFor("abcd explain")
	if !ok {
		t.Fatal("abcd explain has no entry in internal/core/surface/next.go")
	}
	files := committedFactsFiles(t)
	if len(files) == 0 {
		t.Fatalf("%s holds no file; %s", CommandFactsDir, regenerateCommandFacts)
	}
	seen := 0
	for rel, data := range files {
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.DisallowUnknownFields()
		var doc CommandFactsDocument
		if err := dec.Decode(&doc); err != nil {
			t.Errorf("%s does not decode as a facts document: %v", rel, err)
			continue
		}
		if doc.FormatVersion != surface.FactsFormatVersion {
			t.Errorf("%s: format_version %d, want %d", rel, doc.FormatVersion, surface.FactsFormatVersion)
		}
		for _, f := range doc.Commands {
			seen++
			want, err := EncodeCommandFacts(CommandFactsDocument{FormatVersion: doc.FormatVersion, Commands: []surface.Facts{f}})
			if err != nil {
				t.Fatal(err)
			}
			if want, err = spliceNextSteps(want, explainNext.Steps, docStyle{indent: true}); err != nil {
				t.Fatal(err)
			}
			code, out, errOut := explainRun(t, append(explainWords(f.Path), "--json")...)
			if code != 0 {
				t.Errorf("%s: `abcd explain %s --json` exits %d: %s", rel, explainArgs(f.Path), code, errOut)
				continue
			}
			if out != string(want) {
				t.Errorf("%s: `abcd explain %s --json` is not the file's slice for the command:\ngot  %s\nwant %s",
					rel, explainArgs(f.Path), out, want)
			}
		}
	}
	if seen < 100 {
		t.Errorf("the committed files hold %d commands; the files are not the tree", seen)
	}
}
