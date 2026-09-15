//go:build smoke || coldreading

package evals

// The read-block eval: the only component in this workstream capable of
// falsifying the cold reading's blindfold rather than restating it.
//
// Every other component asserts the blindfold. This one plants sentinel warm
// content across every warm location class in a fixture repository state, runs
// the assembler over it, and asserts that no sentinel and no excluded field
// reaches the assembled input — against an exclusion table transcribed from the
// record rather than read from the assembler.

import (
	"bytes"
	"encoding/json"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestReadBlockBaselineIsClean is ac-1: with every plant in its canonical home,
// the assembly passes no planted warm content and no excluded field.
//
// Watched red against an assembler whose exclusion of the local ledger tier is
// removed by a one-line local patch, with ABCD-EVAL-SENTINEL-LEDGER-FRAMING the
// sentinel that must be caught.
func TestReadBlockBaselineIsClean(t *testing.T) {
	f := materialise(t, variantBaseline)
	for _, position := range fullyAsserted {
		t.Run(position, func(t *testing.T) {
			a := assemble(t, f, position)
			vs := checkReadBlock(a)
			if len(vs) > 0 {
				t.Fatalf("the read-block leaked %d time(s) at the %s position:\n%s",
					len(vs), position, reportViolations(vs))
			}
			// The companion assertion over the promoted draft: its plant is COLD at
			// entailment, because the candidate set is what an entailment reading
			// reads — but the ITEM IT WAS GRADUATED FROM is a prior reading's
			// output, and no reading sees another's output (companion 8.3). The
			// join lives in `origin`, `promoted_from` and the Why This Matters line,
			// none of which the intent projection names, so the identifier must
			// reach no bundle at any position, the one that reads the draft
			// included.
			if bytes.Contains(a.BundleRaw, []byte(promotedFixtureItem)) {
				t.Fatalf("the promoted draft's source item %s reached the %s position; a draft is "+
					"admitted at entailment and a prior reading's output must not travel inside it",
					promotedFixtureItem, position)
			}
		})
	}
}

// TestReadBlockCatchesAHoledFirewall is ac-2: a plant relocated into positively
// included material fails, naming the leaked sentinel's class token and the
// position at which it leaked.
//
// It is also the permanent proof that the assertion CAN fail. It demands
// exactly the declared holes, naming their classes, so an assertion that stops
// detecting anything fails here rather than passing quietly.
func TestReadBlockCatchesAHoledFirewall(t *testing.T) {
	if len(holes) == 0 {
		t.Fatal("the holes table is empty, so the negative control controls nothing; " +
			"this test is the permanent proof that the assertion can fail, and it cannot " +
			"be that with no hole in it")
	}
	f := materialise(t, variantHoled)
	// The expectation is PER POSITION, because the four positions no longer read
	// one corpus: a plant relocated into a brief chapter is unreachable at the
	// comparative position, whose only sources are the derived widening run's
	// candidates and the criteria discipline, and one relocated into a candidate
	// is unreachable everywhere else (adr-2609021016272867). A single expectation
	// would have to be wrong at one end or the other.
	for _, position := range fullyAsserted {
		want := []string{}
		for _, h := range holes {
			if h.reachesAt(position) {
				want = append(want, h.Class)
			}
		}
		sort.Strings(want)
		if len(want) == 0 {
			t.Fatalf("no hole is reachable at the %s position, so the negative control controls "+
				"nothing there; a firewall is controlled at every position it is asserted at",
				position)
		}
		t.Run(position, func(t *testing.T) {
			vs := checkReadBlock(assemble(t, f, position))
			got := []string{}
			for _, v := range vs {
				if v.Rule != ruleSentinel {
					t.Errorf("the holed variant produced an unexpected finding: %s", v)
					continue
				}
				got = append(got, v.Class)
			}
			sort.Strings(got)
			if strings.Join(got, ",") != strings.Join(want, ",") {
				t.Fatalf("the holed firewall reported classes %v at the %s position, want %v\n%s",
					got, position, want, reportViolations(vs))
			}
			// The message an operator reads has to carry both, or a failure names
			// no way back to the plant that caused it.
			for _, v := range vs {
				if v.Rule != ruleSentinel {
					continue // already reported above; it carries no class token
				}
				msg := v.String()
				if !strings.Contains(msg, sentinelPrefix+v.Class) || !strings.Contains(msg, position) {
					t.Errorf("the failure message %q names neither the class token nor the position", msg)
				}
			}
		})
	}
}

// TestManifestNamesEveryExcludedFamily holds the disclosure half of the
// exclusion floor (iss-2608311238236490).
//
// The floor is a DECLARATION a reader checks rather than a disclosure a reader
// trusts, so a family refused by construction and named nowhere in the manifest
// is a refusal no reader can verify. That is brief invariant 16 exactly — an
// attestation never states LESS than the examination behind it establishes —
// and the local ledger tier of brief invariant 14 was the family it was short
// of: excluded by absence from the positive walk and by the `.abcd` deny
// segment, asserted by nothing, so a reader could not tell that the framing
// traces and declined construals had been refused.
//
// The oracle's own family list is the standard, transcribed from the record, so
// this fails whenever the assembler's asserted floor falls behind it — in
// either direction of drift.
func TestManifestNamesEveryExcludedFamily(t *testing.T) {
	requireOracleTables(t)
	f := materialise(t, variantBaseline)
	for _, position := range assemblingPositions {
		t.Run(position, func(t *testing.T) {
			a := assemble(t, f, position)
			if len(a.Exclusions) == 0 {
				t.Fatalf("the manifest at %s asserts no exclusions at all; the floor is what a "+
					"reader checks instead of trusting a disclosure", position)
			}
			asserted := make(map[string]bool, len(a.Exclusions))
			for _, e := range a.Exclusions {
				asserted[e.Detail] = true
			}
			for _, fam := range excludedFamilies {
				if !fam.bindsAt(position) || asserted[fam.Path] {
					continue
				}
				t.Errorf("the manifest at %s asserts no exclusion for %s, which the record refuses "+
					"(%s).\n\nThe tier IS excluded, by absence from the positive walk — so this is a "+
					"disclosure gap rather than a leak, and brief invariant 16 is what makes a "+
					"disclosure gap a defect: an attestation never states less than the examination "+
					"behind it establishes.", position, fam.Path, fam.Source)
			}
		})
	}
}

// TestManifestAttestsTheMaterialClassOfEveryItem holds the per-item kind
// attestation itd-198 added and nothing read (iss-2608312019547974).
//
// Framework 8.3 has the manifest map an item to its path AND its class; brief
// invariant 16 has an attestation state no more than its examination
// establishes. A kind nothing checks satisfies neither: a manifest naming every
// item `doc` was green, and so was one that stopped labelling the shipped
// tree's tests apart from its source — the include table's only basename-SUFFIX
// row, whose deletion changes no path in any fixture manifest.
//
// The oracle is the hand-transcribed materialClasses table, so it can disagree
// with the assembler; every class in the closed vocabulary must be attested
// somewhere in the corpus, or a rule of the table would go untested by a corpus
// that stopped carrying its material.
func TestManifestAttestsTheMaterialClassOfEveryItem(t *testing.T) {
	requireOracleTables(t)
	f := materialise(t, variantBaseline)
	attested := map[string]bool{}
	for _, position := range assemblingPositions {
		t.Run(position, func(t *testing.T) {
			a := assemble(t, f, position)
			for _, it := range a.ManifestItems {
				attested[it.Kind] = true
			}
			if vs := checkMaterialClass(a); len(vs) > 0 {
				t.Fatalf("the manifest mis-states the material class of %d item(s) at the %s position:\n%s",
					len(vs), position, reportViolations(vs))
			}
		})
	}
	for _, m := range materialClasses {
		if attested[m.Kind] {
			continue
		}
		t.Errorf("no item in the corpus is attested as %q, so that row of the kind oracle "+
			"judged nothing; the class is in the closed vocabulary and the corpus is what "+
			"makes its rule falsifiable (%s)", m.Kind, m.Source)
	}
}

// TestReadBlockCatchesWarmFieldsOnIncludedTypes is ac-3: a warm field on a
// record type already on the include list fails.
//
// The origin and production-mode plants sit on included record types, and the
// Audit Notes, Why This Matters and scope-condition-disposition plants sit
// inside an included file. Each is a real leak path the moment the key filter
// or the projection is absent — which is how this test is watched red, by
// removing the single exclusion it covers.
func TestReadBlockCatchesWarmFieldsOnIncludedTypes(t *testing.T) {
	f := materialise(t, variantBaseline)
	warm := map[string]bool{"WARM-KEY": true, "WARM-FIELD": true, "DRAFT-ORIGIN": true}
	for _, position := range assemblingPositions {
		t.Run(position, func(t *testing.T) {
			a := assemble(t, f, position)
			var vs []violation
			for _, v := range checkSentinelAbsence(a) {
				if warm[v.Class] {
					vs = append(vs, v)
				}
			}
			vs = append(vs, checkFieldAbsence(a)...)
			if len(vs) > 0 {
				t.Fatalf("a warm field on an included record type reached the %s position %d time(s):\n%s",
					position, len(vs), reportViolations(vs))
			}
		})
	}
}

// TestPriorRunExhaustNeverReaches is ac-4: no prior-run manifest, reading record
// or disposition reaches the assembled input.
//
// The instrument's own exhaust is what a comparative reading is most likely to
// be handed by accident, and nothing else in the cycle tests it.
func TestPriorRunExhaustNeverReaches(t *testing.T) {
	f := materialise(t, variantBaseline)
	for _, position := range assemblingPositions {
		t.Run(position, func(t *testing.T) {
			a := assemble(t, f, position)
			var vs []violation
			for _, v := range checkSentinelAbsence(a) {
				if v.Class == "EXHAUST" {
					vs = append(vs, v)
				}
			}
			for _, v := range checkFamilyAbsence(a) {
				if strings.Contains(v.Detail, ".abcd/development/readings") ||
					strings.Contains(v.Detail, ".abcd/work/issues") {
					vs = append(vs, v)
				}
			}
			if len(vs) > 0 {
				t.Fatalf("the instrument's own exhaust reached the %s position %d time(s):\n%s",
					position, len(vs), reportViolations(vs))
			}
		})
	}
}

// TestTheAssemblerRefusesAnUnredactableShape is the falsifier for the exclusion
// floor's FAIL-CLOSED half, which every other test in this package leaves
// unexercised.
//
// The floor has two halves and only one of them is falsified by a leak. The
// redacting half deletes a span, so removing it leaks; the refusing half stops a
// run, so removing it against a corpus with nothing to refuse changes no output
// at all — which is why deleting the verifyRedaction call outright left this
// lane green. The answer is a corpus that HAS something to refuse: a file the
// include table admits whole, carrying an excluded heading in a form the section
// scan does not report, so the redactor has no span to delete and the refusal is
// the only thing between the warm text and the bundle.
//
// The two shapes are watched red by their own falsifiers: deleting the
// verifyRedaction call, and making sameRendering return false. Under either, the
// same binary exits 0 and the token is in the bundle — which is what the exit-0
// branch below reads and reports.
func TestTheAssemblerRefusesAnUnredactableShape(t *testing.T) {
	requireOracleTables(t)
	if len(refusals) == 0 {
		t.Fatal("the refusals table is empty, so the exclusion floor's fail-closed half has " +
			"nothing to refuse and its removal would change no output; this test is the only " +
			"thing that reaches that half, and it cannot reach it with an empty corpus")
	}
	for _, r := range refusals {
		t.Run(r.Name, func(t *testing.T) {
			f := materialise(t, variantBaseline, r.apply)

			// The anti-vacuity guard, in the shape the corpus already uses: a
			// refusal test over a plant that is not there passes for the wrong
			// reason, and the assembler never sees an untracked file.
			planted, err := os.ReadFile(filepath.Join(f.Root, filepath.FromSlash(r.Path)))
			if err != nil {
				t.Fatalf("the %s refusal plant is not in the materialised tree at %s: %v",
					r.Name, r.Path, err)
			}
			if !bytes.Contains(planted, []byte(r.Token)) {
				t.Fatalf("the %s refusal plant at %s does not carry %s, so a refusal over it "+
					"would be a refusal over nothing", r.Name, r.Path, r.Token)
			}
			if !trackedFiles(t, f.Root)[r.Path] {
				t.Fatalf("the %s refusal plant at %s is not tracked; the assembler walks the "+
					"tracked set, so an untracked plant is never read", r.Name, r.Path)
			}

			for _, position := range assemblingPositions {
				// The refusal is a property of a POSITION THAT READS THE PLANT.
				// Every refusal plant sits in repository material, and the
				// comparative position reads none of it: at that position the
				// include table is the whole account and no source is admitted
				// but the derived widening run's candidates and the criteria
				// discipline (companion 7.2, R3; adr-2609021016272867). A file
				// the position never opens is a file its floor never had to
				// refuse, and asserting a refusal there would assert that a
				// withdrawal did not happen.
				if position == posComparative {
					continue
				}
				outDir := filepath.Join(t.TempDir(), "run-"+position)
				args := append(append([]string{}, assembleVerb...),
					"--position", position, "--target", "HEAD", "--out", outDir, "--dry-run")
				out, code := runIn(t, f.Root, []string{"HOME=" + f.Home}, args...)
				if code == 0 {
					t.Errorf("the assembly at %s over the %s plant exited 0, %s; %s, so the "+
						"exclusion floor's fail-closed half is what refuses it and %s removes "+
						"that refusal\n%s", position, r.Name, whatEscaped(t, outDir, position, r),
						r.Why, r.Falsifier, out)
					continue
				}
				for _, want := range r.Names {
					if !strings.Contains(out, want) {
						t.Errorf("the assembly at %s over the %s plant was refused, but the "+
							"refusal does not name %q, so it is not demonstrably THIS refusal\n%s",
							position, r.Name, want, out)
					}
				}
			}
		})
	}
}

// whatEscaped reads back an assembly that should have been refused and says
// what got out: whether the refusal plant's token is in the bundle, and what the
// field-absence assertion makes of the item carrying it.
//
// The second half is the point. This is one seam with two guards on it — the
// assembler's refusal and this oracle's heading scan — and neither is visible
// from the other, which is how both went unexercised at once. Reporting them
// together means a single failure says whether the refusal went away AND whether
// the oracle would have caught what it was keeping out.
func whatEscaped(t *testing.T, outDir, position string, r refusal) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(outDir, bundleFile))
	if err != nil {
		return "and the bundle could not be read back: " + err.Error()
	}
	if !bytes.Contains(raw, []byte(r.Token)) {
		return "and the bundle does NOT carry " + r.Token
	}
	said := "and " + r.Token + " is in the bundle"
	var bundle struct {
		Items []bundleItem `json:"items"`
	}
	if err := json.Unmarshal(raw, &bundle); err != nil {
		return said + " (which the field-absence assertion cannot be run over: " + err.Error() + ")"
	}
	vs := checkFieldAbsence(assembled{
		Position:    position,
		BundleRaw:   raw,
		ManifestRaw: []byte(`{}`),
		Items:       bundle.Items,
	})
	if len(vs) == 0 {
		return said + ", and the field-absence assertion reports nothing over it — both " +
			"guards on this seam are down at once"
	}
	return said + ", and the field-absence assertion does see it:\n" + reportViolations(vs)
}

// TestFieldAbsenceSeesEveryHeadingForm is the oracle's own falsifier for
// assertion 2's heading half.
//
// The assertion is over a bundle item's TEXT, and a heading is a heading in more
// spellings than ATX. A scan that read `#` alone would report a bundle item
// carrying a setext-underlined or raw-HTML `Audit Notes` as clean — the
// field-absence assertion satisfied completely by the leak it exists to catch.
//
// That the state is unreachable today is not a defence, and it is the reason
// this was invisible from either side. It is unreachable because the ASSEMBLER
// refuses those forms rather than redacting them, and that refusal is the very
// mechanism no plant exercised. Two halves of one seam resting on one unfalsified
// guard: the refusal untested by the eval, and the eval unable to catch what the
// refusal keeps out. This test closes the oracle half, and it does so
// independently of the assembler — it feeds the assertion the item the assembler
// would have to emit, so it holds whether or not that refusal ever softens.
//
// The negative rows are the other half of the claim. A scan widened until it
// reports everything reports nothing, so text that merely NAMES the heading, a
// thematic break and a table divider must all stay clean — and so must the two
// shapes the assembler explicitly declines to read as headings: a frontmatter
// block's closing delimiter, which closes a block rather than underlining one,
// and anything inside a fence, which is an EXAMPLE of a record rather than a
// record. The fenced rows are the cost of widening: setext rules and `<h2>` tags
// are exactly what a fenced example of a record carries.
func TestFieldAbsenceSeesEveryHeadingForm(t *testing.T) {
	const excluded = "Audit Notes"
	found := false
	for _, h := range excludedHeadings {
		if h.Heading == excluded {
			found = true
		}
	}
	if !found {
		t.Fatalf("this test is written against the excluded heading %q, which the "+
			"excludedHeadings table no longer names", excluded)
	}

	for _, tc := range []struct {
		name string
		text string
		want bool
	}{
		{"atx", "## Audit Notes\n\nwarm text\n", true},
		{"atx indented up to three spaces", "   ### Audit Notes\n\nwarm text\n", true},
		{"atx with closing hashes", "## Audit Notes ##\n\nwarm text\n", true},
		{"setext underlined with dashes", "Audit Notes\n-----------\n\nwarm text\n", true},
		{"setext underlined with equals", "Audit Notes\n===\n\nwarm text\n", true},
		{"raw html heading", "<h2>Audit Notes</h2>\n\nwarm text\n", true},
		{"raw html heading across lines", "<h3>\n  Audit Notes\n</h3>\n\nwarm text\n", true},
		{"raw html heading role", `<div role="heading" aria-level="2">Audit Notes</div>` + "\n", true},
		{"emphasised", "## **Audit Notes**\n\nwarm text\n", true},
		{"code marked", "## `Audit Notes`\n\nwarm text\n", true},
		{"lower cased", "## audit notes\n\nwarm text\n", true},
		{"prose naming the heading", "The record's Audit Notes stay behind.\n", false},
		{"a frontmatter close under a key", "---\nid: spc-1\nintent: itd-1\n---\n\nbody\n", false},
		{
			"a frontmatter close under a block scalar naming the heading",
			"---\nsummary: |\n  Open Questions\n---\n\nbody\n",
			false,
		},
		{"a fenced markdown example", "```markdown\nAudit Notes\n-----------\n```\n", false},
		{"a fenced html example", "```html\n<h2>Audit Notes</h2>\n```\n", false},
		{"a fenced atx example", "```markdown\n## Audit Notes\n```\n", false},
		{"a thematic break under a blank line", "Audit Notes are elsewhere.\n\n---\n\nbody\n", false},
		{"a table divider", "| Audit Notes |\n| --- |\n| a cell |\n", false},
		{"an unrelated heading", "## Mechanism\n\nbody\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := assembled{
				Position:    posWidening,
				BundleRaw:   []byte(`{}`),
				ManifestRaw: []byte(`{}`),
				Items:       []bundleItem{{ItemKey: "item-1", Kind: "spec", Text: tc.text}},
			}
			var got []violation
			for _, v := range checkFieldAbsence(a) {
				if v.Rule == ruleExcludedHeader {
					got = append(got, v)
				}
			}
			switch {
			case tc.want && len(got) == 0:
				t.Errorf("the field-absence assertion reports no excluded heading in\n%q\n"+
					"but every reader of that item sees %q in it; a heading the oracle cannot "+
					"see is a leak the oracle calls clean", tc.text, excluded)
			case !tc.want && len(got) > 0:
				t.Errorf("the field-absence assertion reports %s in\n%q\nwhich carries no "+
					"heading at all; a scan widened until it reports everything reports nothing",
					reportViolations(got), tc.text)
			}
		})
	}
}

// bannedImports are the import paths that would make this oracle derived rather
// than transcribed: the assembler's own package, and the include list its
// structural deny is modelled on.
var bannedImports = []string{"internal/core/reading", "internal/core/launch"}

// TestOracleImportsNothingFromTheAssembler is ac-5: no file under evals/ names
// the assembler's package or its include list in an import path, so the
// exclusion table above is transcribed rather than derived.
//
// The check is over DIRECT import paths, which is the independence the record
// asks for. The dedicated lane is stronger than that and can be checked by
// hand: `go list -tags coldreading -deps -test ./evals/` reports this module's
// gitutil and gittest and nothing else, so the assembler is not linked into the
// cold-reading test binary at all, transitively or otherwise. The smoke lane's
// command-tree walk imports the CLI surface, which does reach the assembler;
// that is disclosed rather than caught, and it is harmless because no
// cold-reading assertion reads a Go symbol at all — every one of them reads
// bytes the built binary wrote.
func TestOracleImportsNothingFromTheAssembler(t *testing.T) {
	var files []string
	err := filepath.WalkDir(".", func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(p, ".go") {
			files = append(files, p)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking evals/ for Go files: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("found no Go files under evals/ — the import guard would pass vacuously")
	}

	fset := token.NewFileSet()
	for _, p := range files {
		f, perr := parser.ParseFile(fset, p, nil, parser.ImportsOnly)
		if perr != nil {
			t.Errorf("parsing %s: %v", p, perr)
			continue
		}
		for _, imp := range f.Imports {
			pathValue := strings.Trim(imp.Path.Value, `"`)
			for _, banned := range bannedImports {
				if strings.Contains(pathValue, banned) {
					t.Errorf("%s imports %s; the oracle's exclusion table is transcribed from the "+
						"record, and an eval that read the assembler's own table could only confirm "+
						"it, never falsify it", p, pathValue)
				}
			}
		}
	}
}

// TestEverySentinelIsPlanted is ac-6: every declared sentinel class is present
// in the materialised fixture its declared number of times.
//
// It is the anti-vacuity guard, and it is a criterion rather than a nicety: an
// absence assertion cannot see a corpus that lost its plants, so a corpus that
// silently dropped one would turn every assertion above green for the wrong
// reason. It also checks that every planted repository file is TRACKED, because
// the assembler intersects its walk with the tracked set — a plant git refused
// to track is a plant the assembler could never have leaked.
func TestEverySentinelIsPlanted(t *testing.T) {
	for _, variant := range []string{variantBaseline, variantHoled} {
		t.Run(variant, func(t *testing.T) {
			f := materialise(t, variant)
			tracked := trackedFiles(t, f.Root)
			counts, wheres := countSentinels(t, f)

			for _, c := range sentinelClasses {
				if counts[c.Name] != c.Count {
					t.Errorf("%s is planted %d time(s), want %d; found in %v",
						c.Token(), counts[c.Name], c.Count, wheres[c.Name])
				}
			}
			if variant != variantBaseline {
				return
			}
			for _, c := range sentinelClasses {
				want := append([]string{}, c.Homes...)
				sort.Strings(want)
				got := append([]string{}, wheres[c.Name]...)
				sort.Strings(got)
				if strings.Join(got, ",") != strings.Join(want, ",") {
					t.Errorf("%s is planted in %v, want %v", c.Token(), got, want)
				}
				for _, home := range c.Homes {
					rel, ok := strings.CutPrefix(home, "repo:")
					if !ok {
						continue
					}
					if !tracked[rel] {
						t.Errorf("%s is planted in %s, which the fixture repository does not track; "+
							"the assembler walks the tracked set, so an untracked plant tests nothing",
							c.Token(), rel)
					}
				}
			}
		})
	}
}

// countSentinels walks the materialised fixture — the repository and the
// fixture home both — and reports how many times each class's token occurs and
// which files carry it.
func countSentinels(t *testing.T, f fixture) (map[string]int, map[string][]string) {
	t.Helper()
	counts := map[string]int{}
	wheres := map[string][]string{}
	for _, tree := range []struct{ root, prefix string }{
		{f.Root, "repo:"},
		{f.Home, "home:"},
	} {
		err := filepath.WalkDir(tree.root, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				// The object database is compressed, not planted material, and a
				// packed blob would count a plant twice.
				if d.Name() == ".git" {
					return filepath.SkipDir
				}
				return nil
			}
			data, rerr := os.ReadFile(p)
			if rerr != nil {
				return rerr
			}
			rel, rerr := filepath.Rel(tree.root, p)
			if rerr != nil {
				return rerr
			}
			rel = filepath.ToSlash(rel)
			// The fixture home is keyed on a sha only the run knows, so it is put
			// back to the placeholder the corpus declares its homes with.
			if tree.prefix == "home:" && f.RootSHA != "" {
				rel = strings.ReplaceAll(rel, f.RootSHA, rootSHAPlaceholder)
			}
			for _, c := range sentinelClasses {
				n := bytes.Count(data, []byte(c.Token()))
				if n == 0 {
					continue
				}
				counts[c.Name] += n
				wheres[c.Name] = append(wheres[c.Name], tree.prefix+rel)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walking the materialised fixture at %s: %v", tree.root, err)
		}
	}
	return counts, wheres
}

// TestComparativeChannelCarriesCandidatesAndNothingElse is the read-block
// eval's comparative case, and it replaces TestComparativeRefusesToAssemble,
// which held the refusal adr-2609021016272867 withdraws.
//
// The corpus is planted against it: two committed widening runs at the fixture's
// target, the second with every item dispositioned and admitted and a surprise
// beside it, so the first is the one the assembler derives. What must arrive is
// the derived run's two candidate fields and the criteria discipline; what must
// not is the second run's returned text, any disposition, any admission, any
// surprise, and the derived items' own envelopes.
//
// The exception this holds is narrow by construction and the assertions follow
// its shape: the ledger's own leaf bucket, one run inside it, two fields inside
// each item. Every one of those three boundaries has a plant on the far side of
// it.
func TestComparativeChannelCarriesCandidatesAndNothingElse(t *testing.T) {
	f := materialise(t, variantBaseline)
	a := assemble(t, f, posComparative)

	// The carrier arrives: without it the channel is a table row that supplies
	// nothing, and every absence assertion below would pass over an empty bundle.
	carrier := sentinelPrefix + "CANDIDATE"
	if !bytes.Contains(a.BundleRaw, []byte(carrier)) {
		t.Fatalf("the derived run's configurations did not reach the comparative bundle; the "+
			"channel is the position's whole object, and a bundle without it is a reading about "+
			"nothing.\n%s", string(a.BundleRaw))
	}
	// And so does the discipline the reading characterises against.
	if !bytes.Contains(a.BundleRaw, []byte("Plausibility")) {
		t.Error("the criteria discipline did not reach the comparative bundle; a reading with no " +
			"criteria characterises against nothing (itd-191)")
	}

	// Nothing else from the store. Each of these is a different boundary.
	for _, absent := range []struct{ class, why string }{
		{"ENVELOPE", "the projection is two body fields; the item's pattern is the envelope's"},
		{"EXHAUST", "the exception admits ONE derived run, and another run's text is exhaust"},
		{"FATE", "a candidate's disposition and any surprise beside it are the researcher's judgement"},
		{"GROUNDS", "an admission is the warm half of a candidate's fate"},
		{"DECISION", "the status directories are excluded family by family at this position"},
	} {
		if bytes.Contains(a.BundleRaw, []byte(sentinelPrefix+absent.class)) {
			t.Errorf("the comparative bundle carries %s: %s", sentinelPrefix+absent.class, absent.why)
		}
	}

	// The manifest asserts the exclusions rather than leaving a reader to infer
	// them from silence, and it names the run it derived.
	var m struct {
		CandidateRun    string   `json:"candidate_run"`
		Candidates      int      `json:"candidates"`
		Exercised       *bool    `json:"exercised"`
		CandidateFields []string `json:"candidate_fields"`
		Criteria        []string `json:"criteria"`
		Exclusions      []struct {
			Signal string `json:"signal"`
			Detail string `json:"detail"`
		} `json:"exclusions"`
	}
	if err := json.Unmarshal(a.ManifestRaw, &m); err != nil {
		t.Fatalf("decode the comparative manifest: %v", err)
	}
	if m.CandidateRun != derivedCandidateRun {
		t.Errorf("the manifest names the candidate run %q, want %q — the one committed widening "+
			"run at this target whose items carry no fate", m.CandidateRun, derivedCandidateRun)
	}
	if m.Candidates != len(derivedCandidateItems) {
		t.Errorf("the manifest records %d candidates, want %d", m.Candidates, len(derivedCandidateItems))
	}
	if m.Exercised == nil || !*m.Exercised {
		t.Errorf("the manifest does not state the position as exercised: %v", m.Exercised)
	}
	if strings.Join(m.CandidateFields, "|") != "configuration|what_admits_it" {
		t.Errorf("the manifest states the projected fields as %v", m.CandidateFields)
	}
	if len(m.Criteria) == 0 {
		t.Error("the manifest states no criteria; the slate is what the reading characterises against")
	}
	asserted := map[string]bool{}
	signal := false
	for _, e := range m.Exclusions {
		asserted[e.Detail] = true
		if e.Signal == "readings store" {
			signal = true
		}
	}
	for _, want := range []string{
		".abcd/work/issues/dispositions", ".abcd/work/issues/admissions",
		".abcd/work/issues/surprises", ".abcd/work/issues/open",
		".abcd/work/issues/resolved", ".abcd/work/issues/wontfix",
	} {
		if !asserted[want] {
			t.Errorf("the manifest does not assert the exclusion of %s", want)
		}
	}
	if !signal {
		t.Error("the manifest does not assert what the readings store did NOT supply; the " +
			"directory rows cannot say it, because one run's items do travel")
	}
}

// TestComparativeChannelCatchesAPlantedFate is ac-9, and it is the mechanism
// claim's own falsifier: plant a disposition on a candidate of the DERIVED run
// and the assembly must refuse.
//
// The intent states the claim in one move — "plant a disposition on a candidate
// and show its text in the comparative bundle" — so this is that move performed.
// Either outcome is a finding: a refusal is the property holding, and an
// assembly that succeeded with the disposition's text in the bundle is the
// mechanism shown wrong.
func TestComparativeChannelCatchesAPlantedFate(t *testing.T) {
	planted := sentinelPrefix + "FATE"
	f := materialise(t, variantBaseline, func(t *testing.T, root string) {
		t.Helper()
		dir := filepath.Join(root, ".abcd", "work", "issues", "dispositions", derivedCandidateItems[1])
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		body := "---\nschema_version: 1\nid: \"dsp-901\"\nitem: \"" + derivedCandidateItems[1] +
			"\"\nstate: \"accepted\"\ndisposition_grounds: \"" + planted +
			": answered before it was characterised\"\n---\n"
		if err := os.WriteFile(filepath.Join(dir, "dsp-901.md"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	})

	outDir := filepath.Join(t.TempDir(), "run-comparative")
	args := append(append([]string{}, assembleVerb...),
		"--position", posComparative, "--target", "HEAD", "--out", outDir, "--dry-run")
	out, code := runIn(t, f.Root, []string{"HOME=" + f.Home}, args...)

	if code == 0 {
		t.Fatalf("a candidate carrying a standing disposition was assembled (exit 0). The "+
			"candidate set is defined as PRE-ADMISSION and the design fixes the order as "+
			"characterise first, admit second (companion 8.3).\n%s", out)
	}
	for _, want := range []string{derivedCandidateItems[1], "dsp-901"} {
		if !strings.Contains(out, want) {
			t.Errorf("the refusal does not name %q, so an operator cannot tell which candidate "+
				"stopped it:\n%s", want, out)
		}
	}
	// A refusal that still wrote an artefact would leave a bundle on disk that
	// no assertion in this eval covers — and the disposition's own text would be
	// one field away from it.
	for _, name := range []string{bundleFile, manifestFile} {
		if _, err := os.Stat(filepath.Join(outDir, name)); err == nil {
			raw, _ := os.ReadFile(filepath.Join(outDir, name))
			t.Errorf("the refusal still wrote %s (carries the planted fate: %v)",
				name, bytes.Contains(raw, []byte(planted)))
		}
	}
}

// TestComparativeEntryFitsItsDeclaredWindow is the comparative half of the
// preset-windows eval, which exempts this position by name: its object is
// bounded by the widening run rather than by the tree, so a measurement over the
// tree says nothing about it (spc-2609020626048722).
//
// What is measured instead is a comparative assembly over the fixture's derived
// widening run, held to the declaration the COMMITTED entry carries. The
// fixture's run is small, so this is a bound rather than a calibration — which
// is exactly what the declaration is for.
func TestComparativeEntryFitsItsDeclaredWindow(t *testing.T) {
	declared := committedComparativeWindow(t)
	if declared <= 0 {
		t.Fatalf("%s declares no window at the comparative position, so this eval would hold "+
			"the entry to nothing", presetConfigRel)
	}
	f := materialise(t, variantBaseline)
	raw, code := runIn(t, f.Root, []string{"HOME=" + f.Home},
		append(append([]string{}, assembleVerb...),
			"--position", posComparative, "--target", "HEAD", "--dry-run", "--json")...)
	if code != 0 {
		t.Fatalf("the comparative assembly over the fixture exited %d:\n%s", code, raw)
	}
	var res struct {
		Size struct {
			Bytes     int `json:"bytes"`
			TokensEst int `json:"tokens_est"`
		} `json:"size"`
	}
	if err := json.Unmarshal([]byte(raw), &res); err != nil {
		t.Fatalf("decode the comparative assembly: %v\n%s", err, raw)
	}
	if res.Size.TokensEst <= 0 {
		t.Fatalf("the comparative assembly measured ~%d estimated tokens; a run that measured "+
			"nothing cannot be held to a declaration", res.Size.TokensEst)
	}
	if res.Size.TokensEst > declared {
		t.Fatalf("the comparative entry measures ~%d estimated tokens over %d bytes against a "+
			"declaration of %d. Re-measure and move the declaration in %s, or narrow the entry — "+
			"either is a commit that records why",
			res.Size.TokensEst, res.Size.Bytes, declared, presetConfigRel)
	}
}

// committedComparativeWindow reads the declaration out of the committed preset
// file with this package's own minimal struct, so the figure the entry is held
// to comes from the file rather than from the code under test.
func committedComparativeWindow(t *testing.T) int {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", filepath.FromSlash(presetConfigRel)))
	if err != nil {
		t.Fatalf("read %s: %v", presetConfigRel, err)
	}
	var file struct {
		Positions map[string]struct {
			Window *struct {
				TokensEst int `json:"tokens_est"`
			} `json:"window"`
		} `json:"positions"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatalf("decode %s: %v", presetConfigRel, err)
	}
	e, ok := file.Positions[posComparative]
	if !ok || e.Window == nil {
		return 0
	}
	return e.Window.TokensEst
}

// TestWideningNeverSeesTheShippedIntents is itd-194 ac-7 (the design framework's
// widening object; the readings companion's section 5.2; the maintainer's ruling
// of 2026-09-02, which resolves iss-2609012259587904).
//
// Neither design document lists the shipped intents in the widening object, and
// the assembler passed them there. The row withdraws from that position and the
// exclusion floor asserts the withdrawal, so a reader can CHECK the refusal
// rather than infer it from a row's silence — which is the same disclosure
// argument the local ledger tier's entry rests on.
//
// The other two assembling positions still receive them, and that half is
// asserted here too: a withdrawal that took the shipped intents out everywhere
// would satisfy an absence assertion completely while destroying the object of
// the two positions that are about the record as it stands.
func TestWideningNeverSeesTheShippedIntents(t *testing.T) {
	requireOracleTables(t)
	f := materialise(t, variantBaseline)
	const shipped = ".abcd/development/intents/shipped"

	for _, position := range assemblingPositions {
		t.Run(position, func(t *testing.T) {
			a := assemble(t, f, position)
			present := false
			for _, it := range a.ManifestItems {
				if strings.HasPrefix(path.Clean(it.Path), shipped+"/") {
					present = true
					break
				}
			}
			asserted := false
			for _, e := range a.Exclusions {
				if e.Detail == shipped {
					asserted = true
					break
				}
			}
			if position == posWidening {
				if present {
					t.Errorf("the widening assembly carries an item from %s; neither design "+
						"document lists the shipped intents in the widening object", shipped)
				}
				if !asserted {
					t.Errorf("the widening manifest asserts no exclusion for %s; a refusal a "+
						"reader cannot check is a refusal the manifest does not make", shipped)
				}
				if vs := checkFamilyAbsence(a); len(vs) > 0 {
					t.Errorf("family absence reports %d violation(s) at widening:\n%s",
						len(vs), reportViolations(vs))
				}
				return
			}
			// Comparative withdrew too, and for its own reason: at that position
			// the include table is the whole account and no source is admitted
			// but the derived widening run's candidates and the criteria
			// discipline (companion 7.2, R3; adr-2609021016272867). Its
			// withdrawal is NOT asserted in the floor, because that position's
			// ledger rows are what its manifest asserts — an intents row there
			// would be a claim about a family the position's own rows already
			// leave behind.
			if position == posComparative {
				if present {
					t.Errorf("the comparative assembly carries an item from %s; every row but "+
						"the candidates and the criteria discipline withdraws from that position",
						shipped)
				}
				return
			}
			if !present {
				t.Errorf("the %s assembly carries no item from %s; only widening and comparative "+
					"withdraw, and a withdrawal everywhere would destroy the object of this position",
					position, shipped)
			}
			if asserted {
				t.Errorf("the %s manifest asserts %s excluded while its rows admit it", position, shipped)
			}
		})
	}
}

// TestManifestMarksWhatTheFloorDidNotExamine is itd-194 ac-3 and ac-4 at the
// artefact: every item carries a scan mark, the mark is the one the record says
// its path carries, and the excluded key and heading assertions are held over
// the items marked `parsed` and over no other.
//
// The corpus is what makes it more than a shape check. The baseline carries a
// Go test file with a record-shaped page and a literal `## Audit Notes` section
// in it — the live leak's own shape, found on this repository's corpus by the
// itd-183 audit (iss-2608301450065320). It still travels, because both design
// documents name the shipped tree's code and tests as a reading's object; what
// changed is that the manifest now says, per item, that no examination stood
// behind the exclusion assertion over it.
func TestManifestMarksWhatTheFloorDidNotExamine(t *testing.T) {
	requireOracleTables(t)
	f := materialise(t, variantBaseline)
	marks := map[string]bool{}
	for _, position := range assemblingPositions {
		t.Run(position, func(t *testing.T) {
			a := assemble(t, f, position)
			for _, it := range a.ManifestItems {
				marks[it.Scan] = true
			}
			if vs := checkScanMarks(a); len(vs) > 0 {
				t.Fatalf("the manifest mis-states the examination behind %d item(s) at the %s "+
					"position:\n%s", len(vs), position, reportViolations(vs))
			}
		})
	}
	// Both marks have to occur somewhere in the corpus, or one branch of the
	// oracle judged nothing: a corpus of markdown alone makes "unscanned" a rule
	// no item exercises, and the assertion goes quietly vacuous in the direction
	// that matters.
	for _, want := range []string{"parsed", "unscanned"} {
		if !marks[want] {
			t.Errorf("no item in the corpus is marked %q, so that branch of the scan oracle "+
				"judged nothing; the mark is what tells a scan that ran from one that never did", want)
		}
	}
}

// TestTheFixtureLeakIsAbsentUnderEveryCommittedPreset is itd-194 ac-5 over a
// clone of HEAD, the half `TestOnlyTheTreePositionsNameSourceOrTest` in the
// assembler's own package cannot take: that test reads which kinds the committed
// entries name and applies them to a fixture tree, and this one applies them to
// THIS repository, where the live leak actually sits.
//
// The leak is `internal/core/site/fixture_test.go`, a Go test file carrying a
// record-shaped page with an `## Audit Notes` section in it, found on this
// repository's corpus by the itd-183 audit (iss-2608301450065320). Two claims,
// and they are the two halves of adr-56 as refined on 2026-09-02:
//
//   - No committed entry reaches it. The detection and widening entries name the
//     `source` and `test` kinds, but the object set's paths do not reach
//     `internal/core/site`, so the file is in no assembly this repository runs.
//   - An entry that DOES reach it gets it whole and marked `unscanned`. The
//     acceptance is paid for by disclosure: the manifest says, per item, that no
//     examination stood behind the exclusion assertion over it, which is brief
//     invariant 16 made a property of the artefact rather than of the run.
//
// The second half is proved by committing such an entry IN THE CLONE, so the
// dirty gate and the tracked-file check are satisfied by construction and this
// repository's own committed file is untouched.
func TestTheFixtureLeakIsAbsentUnderEveryCommittedPreset(t *testing.T) {
	const leak = "internal/core/site/fixture_test.go"
	clone := cloneHeadDetached(t)
	if _, err := os.Stat(filepath.Join(clone, filepath.FromSlash(leak))); err != nil {
		t.Fatalf("%s is not in the clone, so this eval asserts the absence of a file that does "+
			"not exist: %v", leak, err)
	}

	for _, position := range assemblingPositions {
		// Not at comparative, and the reason is a fact about THIS repository
		// rather than about the entry: that position derives its candidate set
		// from the record, and this repository holds no committed widening run,
		// so the assembly refuses and there is no item set to look in. The
		// entry's own repository material is the criteria discipline alone, which
		// reaches no package and so can never reach the fixture leak
		// (adr-2609021016272867). The day a widening run is committed here, this
		// exclusion is what has to be revisited.
		if position == posComparative {
			continue
		}
		for _, it := range assembleClone(t, clone, position) {
			if path.Clean(it.Path) == leak {
				t.Errorf("the committed %s entry passed %s; no committed entry's object set "+
					"reaches internal/core/site, and an item the exclusion floor cannot examine "+
					"must not arrive under an entry that never named it", position, leak)
			}
		}
	}

	// The opted-in half. A detection entry naming the `test` kind under
	// `internal/core/site` reaches the file, and the manifest discloses that the
	// floor did not examine it.
	writeCloneFile(t, clone, presetConfigRel, `{
  "schema_version": 2,
  "positions": {
    "detection": {
      "comment": "An eval-only entry: it opts the test kind in under the package holding the live fixture leak, so the disclosure the mark carries is asserted over a real item.",
      "object": {"records": [], "paths": ["internal/core/site"]},
      "kinds": ["test"],
      "window": {"tokens_est": 100000000, "measured_tokens_est": 1, "measured_bytes": 1, "measured_at": "0000000"}
    }
  }
}
`)
	gitInClone(t, clone, "add", presetConfigRel)
	gitInClone(t, clone, "-c", "user.name=abcd eval", "-c", "user.email=eval@example.invalid",
		"commit", "-q", "-m", "opt the test kind in under internal/core/site")

	found := false
	for _, it := range assembleClone(t, clone, posDetection) {
		if path.Clean(it.Path) != leak {
			continue
		}
		found = true
		if it.Scan != "unscanned" {
			t.Errorf("%s arrived marked %q; the exclusion floor parses markdown alone, so an "+
				"item from an unscanned row travels whole and the manifest says the key and "+
				"heading exclusions rest on no examination of it (adr-56 as refined "+
				"2026-09-02; brief invariant 16)", leak, it.Scan)
		}
	}
	if !found {
		t.Errorf("an entry naming the test kind under internal/core/site passed no item at %s, "+
			"so the disclosure was asserted over nothing", leak)
	}
}

// assembleClone dry-runs one position over a clone and returns its manifest
// items. It is separate from `assemble` above, which is bound to the planted
// fixture and its carrier floor: this one runs over this repository's own tree,
// where the carriers do not exist.
func assembleClone(t *testing.T, root, position string) []manifestItem {
	t.Helper()
	outDir := filepath.Join(t.TempDir(), "run-"+position)
	home := t.TempDir()
	args := append(append([]string{}, assembleVerb...),
		"--position", position, "--target", "HEAD", "--out", outDir, "--dry-run")
	out, code := runIn(t, root, []string{"HOME=" + home}, args...)
	if code != 0 {
		t.Fatalf("`abcd %s` over a clone of HEAD exited %d\n%s",
			strings.Join(args, " "), code, out)
	}
	var manifest struct {
		Items []manifestItem `json:"items"`
	}
	if err := json.Unmarshal(readArtefact(t, filepath.Join(outDir, manifestFile)), &manifest); err != nil {
		t.Fatalf("decoding the manifest at %s: %v", position, err)
	}
	if len(manifest.Items) == 0 {
		t.Fatalf("the assembly at %s over a clone of HEAD carried no item, so an absence "+
			"assertion over it establishes nothing", position)
	}
	return manifest.Items
}

// writeCloneFile writes one file inside a clone, creating its parents.
func writeCloneFile(t *testing.T, root, rel, body string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", rel, err)
	}
	if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}
