package lint

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The three generated-index shapes the lint renders, built over a temp
// repository: a drafts bench as id-and-title bullets, the ADR store as a table
// read from each record's frontmatter, and the command pages as a backticked
// name list. The shipped config declares the last two; the titles shape has no
// shipped region since the drafts bench stopped being committed as a list
// (iss-2610280107422462).
func draftsGenSpec() IndexSpec {
	return IndexSpec{
		ID: "drafts-titles", Doc: "scope.md", Dir: "drafts",
		Suffix: ".md", DirEntry: `^(itd-[0-9]+)`, Mode: indexModeGenerated, Render: indexRenderTitles,
	}
}

func adrGenSpec() IndexSpec {
	return IndexSpec{
		ID: "adr-index", Doc: "adrs/README.md", Dir: "adrs",
		Suffix: ".md", Mode: indexModeGenerated, Render: indexRenderADRTable,
	}
}

func namesGenSpec() IndexSpec {
	return IndexSpec{
		ID: "commands", Doc: "surfaces.md", Dir: "commands",
		Suffix: ".md", Mode: indexModeGenerated, Render: indexRenderNames,
	}
}

func writeGenADR(t *testing.T, root, file, id, title, status, date, supersedes, supersededBy string) {
	t.Helper()
	writeFile(t, root, filepath.Join("adrs", file), strings.Join([]string{
		"---",
		"id: " + id,
		"slug: x",
		"status: " + status,
		"date: " + date,
		"supersedes: " + supersedes,
		"superseded_by: " + supersededBy,
		"---",
		"",
		"# " + strings.ToUpper(id[:3]) + id[3:] + ": " + title,
		"",
		"## Context",
	}, "\n")+"\n")
}

// genDoc is a document holding one region for id, with hand-written prose on
// both sides that the generator must never touch.
func genDoc(id, body string) string {
	return "# Register\n\nHand-written prose above.\n\n<!-- index: " + id + " -->\n" +
		body + "<!-- /index -->\n\nHand-written prose below.\n"
}

// TestIndexDriftGeneratedFlagsStaleRegion is the gate half of the change: a
// generated region is held to the derivation byte for byte, so a stale status,
// a hand-reworded title or a missing row each fail, the finding names the
// command that rewrites the region, and a region equal to the derivation
// passes. Under the old exact mode none of the first two was a finding at all,
// and the ADR index was not gated (iss-374).
func TestIndexDriftGeneratedFlagsStaleRegion(t *testing.T) {
	root := t.TempDir()
	writeGenADR(t, root, "0001-one.md", "adr-1", "One decision", "accepted", "2026-05-04", "null", "adr-2610010000000001")
	writeGenADR(t, root, "2610010000000001-two.md", "adr-2610010000000001", "Two decision", "proposed", "2026-10-01", "[adr-1]", "null")
	writeFile(t, root, filepath.Join("adrs", "README.md"), genDoc("adr-index", strings.Join([]string{
		"| ID | Title | Status | Date |",
		"|---|---|---|---|",
		"| [adr-1](0001-one.md) | One decision (superseded by adr-2610010000000001) | accepted | 2026-05-04 |",
		"| [adr-2610010000000001](2610010000000001-two.md) | Two decision (supersedes adr-1) | accepted | 2026-10-01 |",
	}, "\n")+"\n"))

	fs, err := Lint(indexCfg(adrGenSpec()), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, "index_drift"); n != 1 {
		t.Fatalf("expected 1 index_drift finding for the stale status, got %d: %+v", n, fs)
	}
	if !messageContains(fs, "go generate ./internal/surface/cli") {
		t.Errorf("the finding must name the command that rewrites the region; got %+v", fs)
	}

	// Regenerated, the same region passes.
	rw, err := RegenerateIndexes(root, indexCfg(adrGenSpec()))
	if err != nil {
		t.Fatal(err)
	}
	if len(rw) != 1 || rw[0].Committed == rw[0].Want {
		t.Fatalf("expected one rewrite of the stale index, got %+v", rw)
	}
	if !strings.Contains(rw[0].Want, "| Two decision (supersedes adr-1) | proposed | 2026-10-01 |") {
		t.Errorf("rewrite does not carry the record's own status:\n%s", rw[0].Want)
	}
	writeFile(t, root, filepath.Join("adrs", "README.md"), rw[0].Want)
	fs, err = Lint(indexCfg(adrGenSpec()), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, "index_drift"); n != 0 {
		t.Fatalf("a regenerated region must pass, got %+v", fs)
	}
}

// TestIndexDriftGeneratedRegionMissing keeps the fail-closed half: a generated
// index whose markers are gone is a finding, not a silent pass, and a config
// naming an unknown renderer is a configuration error.
func TestIndexDriftGeneratedRegionMissing(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, filepath.Join("commands", "ahoy.md"), "# ahoy\n")
	writeFile(t, root, "surfaces.md", "# Register\n\nno markers here\n")
	fs, err := Lint(indexCfg(namesGenSpec()), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, "index_drift"); n != 1 {
		t.Fatalf("expected the missing region to be a finding, got %+v", fs)
	}
	if _, err := RegenerateIndexes(root, indexCfg(namesGenSpec())); err == nil {
		t.Fatal("the generator must refuse a document without the region's markers")
	}

	bad := namesGenSpec()
	bad.Render = "guess"
	if _, err := Lint(indexCfg(bad), root); err == nil {
		t.Fatal("an unknown renderer must be a configuration error")
	}
}

// TestRegeneratedIndexIsOrderIndependent is the merge half: two branches each
// add a record and each regenerate the register, so their regions conflict.
// Whichever side's document the merge keeps, rerunning the generator over the
// merged tree yields the same bytes — the same as a fresh render — and a second
// run changes nothing. The prose outside the markers is never touched.
func TestRegeneratedIndexIsOrderIndependent(t *testing.T) {
	cases := []struct {
		name   string
		spec   IndexSpec
		base   func(t *testing.T, root string)
		branch [2]func(t *testing.T, root string)
	}{
		{
			name: "drafts titles",
			spec: draftsGenSpec(),
			base: func(t *testing.T, root string) {
				writeFile(t, root, filepath.Join("drafts", "itd-8-with-code.md"), "---\nid: itd-8\n---\n\n# Lifeboats carry code\n")
				writeFile(t, root, filepath.Join("drafts", "itd-2609151838312703-mailbox.md"), "---\nid: itd-2609151838312703\n---\n\n# Sessions leave each other messages\n")
				writeFile(t, root, filepath.Join("drafts", "README.md"), "# drafts\n")
			},
			branch: [2]func(t *testing.T, root string){
				func(t *testing.T, root string) {
					writeFile(t, root, filepath.Join("drafts", "itd-77-home.md"), "---\nid: itd-77\n---\n\n# Relocatable home\n")
				},
				func(t *testing.T, root string) {
					writeFile(t, root, filepath.Join("drafts", "itd-2610111707085098-persona.md"), "---\nid: itd-2610111707085098\n---\n\n# Each session takes a persona name\n")
				},
			},
		},
		{
			name: "adr table",
			spec: adrGenSpec(),
			base: func(t *testing.T, root string) {
				writeGenADR(t, root, "0001-one.md", "adr-1", "One", "accepted", "2026-05-04", "null", "null")
			},
			branch: [2]func(t *testing.T, root string){
				func(t *testing.T, root string) {
					writeGenADR(t, root, "2610010000000001-a.md", "adr-2610010000000001", "Branch A's decision", "accepted", "2026-10-01", "null", "null")
				},
				func(t *testing.T, root string) {
					writeGenADR(t, root, "2610020000000002-b.md", "adr-2610020000000002", "Branch B's decision", "proposed", "2026-10-02", "null", "null")
				},
			},
		},
		{
			name: "command names",
			spec: namesGenSpec(),
			base: func(t *testing.T, root string) {
				for _, v := range []string{"abcd", "ahoy", "capture"} {
					writeFile(t, root, filepath.Join("commands", v+".md"), "# "+v+"\n")
				}
			},
			branch: [2]func(t *testing.T, root string){
				func(t *testing.T, root string) { writeFile(t, root, filepath.Join("commands", "zeta.md"), "# zeta\n") },
				func(t *testing.T, root string) {
					writeFile(t, root, filepath.Join("commands", "build.md"), "# build\n")
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := indexCfg(tc.spec)
			// regen rewrites the document in root and returns its bytes.
			regen := func(root string) string {
				t.Helper()
				rw, err := RegenerateIndexes(root, cfg)
				if err != nil {
					t.Fatal(err)
				}
				if len(rw) != 1 {
					t.Fatalf("expected one document, got %+v", rw)
				}
				writeFile(t, root, tc.spec.Doc, rw[0].Want)
				return rw[0].Want
			}
			// side builds the base, applies the given branches in order, and
			// regenerates after each, as a lane does before it commits.
			side := func(order ...int) (string, string) {
				root := t.TempDir()
				tc.base(t, root)
				writeFile(t, root, tc.spec.Doc, genDoc(tc.spec.ID, ""))
				regen(root)
				var doc string
				for _, i := range order {
					tc.branch[i](t, root)
					doc = regen(root)
				}
				return root, doc
			}
			_, docA := side(0)
			_, docB := side(1)
			if docA == docB {
				t.Fatal("the two branches must produce conflicting regions for the test to mean anything")
			}
			// The merged tree holds both records; the merge kept one side's
			// document. Rerunning the generator settles it either way.
			mergedA, _ := side(0)
			tc.branch[1](t, mergedA)
			writeFile(t, mergedA, tc.spec.Doc, docA)
			fromA := regen(mergedA)

			mergedB, _ := side(1)
			tc.branch[0](t, mergedB)
			writeFile(t, mergedB, tc.spec.Doc, docB)
			fromB := regen(mergedB)

			if fromA != fromB {
				t.Fatalf("regenerating the merged tree depends on which side was kept:\n--- kept A ---\n%s\n--- kept B ---\n%s", fromA, fromB)
			}
			if _, fresh := side(1, 0); fresh != fromA {
				t.Fatalf("the merged region differs from a fresh render:\n--- merged ---\n%s\n--- fresh ---\n%s", fromA, fresh)
			}
			if again := regen(mergedA); again != fromA {
				t.Fatalf("regeneration is not idempotent:\n%s\n---\n%s", fromA, again)
			}
			if !strings.HasPrefix(fromA, "# Register\n\nHand-written prose above.\n\n<!-- index: "+tc.spec.ID+" -->\n") ||
				!strings.HasSuffix(fromA, "<!-- /index -->\n\nHand-written prose below.\n") {
				t.Fatalf("the prose outside the markers changed:\n%s", fromA)
			}
		})
	}
}

// TestGeneratedIndexesMatchTree is the drift test over the real repository:
// every index the shipped config declares as generated must equal what the
// generator derives from the tree today. A failure names the document and the
// one command that fixes it. It also pins that the registers the hand-edit
// contention came from (iss-2608220150157511) are generated, so the test can
// never pass by the config quietly dropping them back to hand-edited.
func TestGeneratedIndexesMatchTree(t *testing.T) {
	repoRoot := filepath.Join("..", "..", "..")
	cfg, err := LoadConfig(filepath.Join(repoRoot, ".abcd", "record-lint.json"))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"commands": false, "adr-index": false}
	for _, s := range cfg.Rules["index_drift"].Indexes {
		if _, ok := want[s.ID]; ok && s.Mode == indexModeGenerated {
			want[s.ID] = true
		}
	}
	for id, ok := range want {
		if !ok {
			t.Errorf("record-lint.json must declare index %q in %s mode", id, indexModeGenerated)
		}
	}
	rw, err := RegenerateIndexes(repoRoot, cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rw {
		if r.Committed != r.Want {
			t.Errorf("%s: a generated index region differs from the tree; run `go generate ./internal/surface/cli` and commit the result", r.Doc)
		}
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "go.mod")); err != nil {
		t.Fatalf("repo root not found from the test: %v", err)
	}
}

// TestGeneratedIndexMissingDirectoryFailsClosed: a generated index whose source
// directory is gone — moved, or misnamed in the config — must not render as an
// empty listing. Each renderer is first generated over an existing empty
// directory, then the config is pointed at a directory that does not exist: lint
// must name the directory in a finding and the generator must refuse, naming
// it, rather than the region silently staying (or becoming) empty while the
// gate stays green.
func TestGeneratedIndexMissingDirectoryFailsClosed(t *testing.T) {
	for _, spec := range []IndexSpec{namesGenSpec(), draftsGenSpec(), adrGenSpec()} {
		t.Run(spec.Render, func(t *testing.T) {
			root := t.TempDir()
			if err := os.MkdirAll(filepath.Join(root, spec.Dir), 0o755); err != nil {
				t.Fatal(err)
			}
			writeFile(t, root, spec.Doc, genDoc(spec.ID, ""))
			rw, err := RegenerateIndexes(root, indexCfg(spec))
			if err != nil {
				t.Fatal(err)
			}
			writeFile(t, root, spec.Doc, rw[0].Want)

			moved := spec
			moved.Dir = spec.Dir + "-moved"
			fs, err := Lint(indexCfg(moved), root)
			if err != nil {
				t.Fatal(err)
			}
			if n := countRule(fs, "index_drift"); n != 1 || !messageContains(fs, moved.Dir+" does not exist") {
				t.Fatalf("a missing source directory must be one finding naming %s; got %+v", moved.Dir, fs)
			}
			if _, err := RegenerateIndexes(root, indexCfg(moved)); err == nil || !strings.Contains(err.Error(), moved.Dir+" does not exist") {
				t.Fatalf("the generator must refuse a missing source directory, naming %s; got %v", moved.Dir, err)
			}
		})
	}
}

// TestGeneratedIndexDuplicateRegionRefused: a document holding a second region
// for the same id — a pasted copy, or a merge that kept both sides' markers —
// must be a finding and a generator refusal naming both lines. Taking the first
// region and ignoring the second left a stale listing in the document that no
// gate read and no regeneration rewrote.
func TestGeneratedIndexDuplicateRegionRefused(t *testing.T) {
	spec := namesGenSpec()
	root := t.TempDir()
	writeFile(t, root, filepath.Join("commands", "ahoy.md"), "# ahoy\n")
	writeFile(t, root, spec.Doc, genDoc(spec.ID, ""))
	rw, err := RegenerateIndexes(root, indexCfg(spec))
	if err != nil {
		t.Fatal(err)
	}
	// The first region is current; a second, stale one follows it.
	doc := rw[0].Want + "\n<!-- index: commands -->\n`gone`.\n<!-- /index -->\n"
	writeFile(t, root, spec.Doc, doc)

	fs, err := Lint(indexCfg(spec), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, "index_drift"); n != 1 || !messageContains(fs, "lines 5 and 11") {
		t.Fatalf("a second region for the same id must be one finding naming both lines; got %+v", fs)
	}
	if _, err := RegenerateIndexes(root, indexCfg(spec)); err == nil || !strings.Contains(err.Error(), "lines 5 and 11") {
		t.Fatalf("the generator must refuse a second region for the same id; got %v", err)
	}
}

// TestGeneratedIndexSkipsFencedMarkers: a document may show the marker shape in
// a fenced code block. The example is quoted material, never the region: the
// generator must leave it byte for byte and write the real region after it, the
// fenced copy must not count as a second region, and a document whose only
// markers are fenced has no region at all.
func TestGeneratedIndexSkipsFencedMarkers(t *testing.T) {
	spec := namesGenSpec()
	root := t.TempDir()
	writeFile(t, root, filepath.Join("commands", "ahoy.md"), "# ahoy\n")
	example := "```markdown\n<!-- index: commands -->\n`example`.\n<!-- /index -->\n```\n"
	writeFile(t, root, spec.Doc, "# Register\n\nThe region looks like this:\n\n"+example+"\n"+
		"<!-- index: commands -->\n<!-- /index -->\n")

	rw, err := RegenerateIndexes(root, indexCfg(spec))
	if err != nil {
		t.Fatal(err)
	}
	want := "# Register\n\nThe region looks like this:\n\n" + example + "\n" +
		"<!-- index: commands -->\n`ahoy`.\n<!-- /index -->\n"
	if rw[0].Want != want {
		t.Fatalf("the generator must skip the fenced example and write the real region:\n--- got ---\n%s\n--- want ---\n%s", rw[0].Want, want)
	}
	writeFile(t, root, spec.Doc, rw[0].Want)
	fs, err := Lint(indexCfg(spec), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, "index_drift"); n != 0 {
		t.Fatalf("a regenerated region beside a fenced example must pass; got %+v", fs)
	}

	writeFile(t, root, spec.Doc, "# Register\n\n"+example)
	if _, err := RegenerateIndexes(root, indexCfg(spec)); !errors.Is(err, errNoRegion) {
		t.Fatalf("a document whose only markers are fenced has no region; got %v", err)
	}
}

// TestGeneratedIndexRegionAdviceFitsTheCase: each region refusal carries the
// advice for its own case. Only a missing region may suggest deleting the
// config entry (the enumeration went); an unclosed region needs its closing
// marker, and a duplicate needs one region removed — both then regenerated.
func TestGeneratedIndexRegionAdviceFitsTheCase(t *testing.T) {
	spec := namesGenSpec()
	cases := []struct {
		name, doc      string
		want           []string
		mustNotSuggest bool
	}{
		{"missing", "# Register\n", []string{"delete the config entry"}, false},
		{"unclosed", "# Register\n\n<!-- index: commands -->\n`ahoy`.\n",
			[]string{"add a line holding only <!-- /index --> where the region ends", IndexRegenerateCommand}, true},
		{"duplicate", "<!-- index: commands -->\n<!-- /index -->\n<!-- index: commands -->\n<!-- /index -->\n",
			[]string{"keep one region per id", IndexRegenerateCommand}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, filepath.Join("commands", "ahoy.md"), "# ahoy\n")
			writeFile(t, root, spec.Doc, tc.doc)
			fs, err := Lint(indexCfg(spec), root)
			if err != nil {
				t.Fatal(err)
			}
			if countRule(fs, "index_drift") != 1 {
				t.Fatalf("expected one finding, got %+v", fs)
			}
			for _, w := range tc.want {
				if !messageContains(fs, w) {
					t.Errorf("the %s finding must say %q; got %q", tc.name, w, fs[0].Message)
				}
			}
			if tc.mustNotSuggest && messageContains(fs, "delete the config entry") {
				t.Errorf("the %s finding must not advise deleting the config entry; got %q", tc.name, fs[0].Message)
			}
		})
	}
}

// TestRegeneratedIndexKeepsCRLF holds the generator to the document's own line
// endings (iss-2610150749065106). It read a CRLF register normalised to LF and
// wrote the whole file back as LF, so a regeneration rewrote every line outside
// the markers too, and the lint step normalises as well, so the churn was
// silent. A current CRLF region must regenerate to the committed bytes exactly,
// and a stale one must change only the region, its new lines ending as the
// document's do.
func TestRegeneratedIndexKeepsCRLF(t *testing.T) {
	crlf := func(s string) string { return strings.ReplaceAll(s, "\n", "\r\n") }
	root := t.TempDir()
	writeFile(t, root, filepath.Join("commands", "ahoy.md"), "# ahoy\n")
	writeFile(t, root, filepath.Join("commands", "build.md"), "# build\n")

	current := crlf(genDoc("commands", "`ahoy`, `build`.\n"))
	writeFile(t, root, "surfaces.md", current)
	rw, err := RegenerateIndexes(root, indexCfg(namesGenSpec()))
	if err != nil {
		t.Fatal(err)
	}
	if len(rw) != 1 || rw[0].Want != current {
		t.Fatalf("a current CRLF region must regenerate to the committed bytes; got %q, want %q", rw[0].Want, current)
	}

	writeFile(t, root, "surfaces.md", crlf(genDoc("commands", "`ahoy`.\n")))
	rw, err = RegenerateIndexes(root, indexCfg(namesGenSpec()))
	if err != nil {
		t.Fatal(err)
	}
	if len(rw) != 1 || rw[0].Want != current {
		t.Fatalf("a stale CRLF region must regenerate with CRLF endings and the prose untouched; got %q, want %q", rw[0].Want, current)
	}
}

// TestGeneratedIndexRefusesControlCharacters holds every value a renderer
// writes into a region to one line of plain text. A directory entry whose name
// holds a line break was written raw, so the next run split it into lines of
// its own and could stand a marker on one: a name cannot hold the closing
// marker's slash, but it can hold an opening marker, which the next run reads
// as a second region for the id. A carriage return or another control
// character left in a record's title or file name is the same hazard in the
// other renderers. Both the generator and the drift check refuse, naming the
// entry in quoted form, and neither writes it. The names are real files: the
// filesystem takes a line break in a name.
func TestGeneratedIndexRefusesControlCharacters(t *testing.T) {
	forged := "a\n<!-- index: commands -->\nb"
	cases := []struct {
		name  string
		spec  IndexSpec
		setup func(t *testing.T, root string)
		quote string
	}{
		{"names renderer, a line break in a file name", namesGenSpec(), func(t *testing.T, root string) {
			writeFile(t, root, filepath.Join("commands", "ahoy.md"), "# ahoy\n")
			writeFile(t, root, filepath.Join("commands", forged+".md"), "# forged\n")
			writeFile(t, root, "surfaces.md", genDoc("commands", "`ahoy`.\n"))
		}, `"a\n<!-- index: commands -->\nb"`},
		{"adr table, a line break in a file name", adrGenSpec(), func(t *testing.T, root string) {
			writeGenADR(t, root, "0001-one.md", "adr-1", "One decision", "accepted", "2026-05-04", "null", "null")
			writeGenADR(t, root, "0002-"+forged+".md", "adr-2", "Two decision", "accepted", "2026-05-04", "null", "null")
			writeFile(t, root, filepath.Join("adrs", "README.md"), genDoc("adr-index", ""))
		}, `"0002-a\n<!-- index: commands -->\nb.md"`},
		{"titles renderer, a carriage return in a title", draftsGenSpec(), func(t *testing.T, root string) {
			writeFile(t, root, filepath.Join("drafts", "itd-1-x.md"), "# One\rtwo\n")
			writeFile(t, root, "scope.md", genDoc("drafts-titles", ""))
		}, `"One\rtwo"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			c.setup(t, root)
			rw, err := RegenerateIndexes(root, indexCfg(c.spec))
			if err == nil {
				t.Fatalf("the generator must refuse a control character in an entry; got %+v", rw)
			}
			if !strings.Contains(err.Error(), c.quote) {
				t.Errorf("the refusal must name the entry as %s; got %v", c.quote, err)
			}
			fs, err := Lint(indexCfg(c.spec), root)
			if err != nil {
				t.Fatal(err)
			}
			if n := countRule(fs, "index_drift"); n != 1 || !messageContains(fs, c.quote) {
				t.Fatalf("the drift check must refuse the same entry, naming it as %s; got %+v", c.quote, fs)
			}
		})
	}
}
