//go:build smoke || coldreading

package evals

// The order-adversarial corpus and the lexicographic order oracle.
//
// Byte-identity across two runs is satisfied by ANY order the assembler picks
// consistently, including a filesystem-dependent one that changes on another
// machine. So this file carries an oracle the assembler does not supply: the
// item paths are collected from what the assembler wrote, a copy is sorted with
// the EVAL'S OWN lexicographic sort, and the two are required to agree. Sorting
// with the assembler's own comparator would agree by construction and could
// only confirm it.
//
// An oracle needs something to catch. `testdata/cold-reading/order/` is that:
// six records whose names sort one way by byte, another by case-folded
// comparison, a third by numeric suffix and a fourth by path component,
// materialised in a creation order that is none of the four.
// TestFixtureOrderIsAdversarial asserts those disagreements hold, so the order
// oracle cannot pass by coincidence.

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"unicode"
)

// orderDir is the corpus this file owns. It mirrors repo-relative paths, so a
// file's path inside it is the path it is materialised at.
const orderDir = corpusDir + "/order"

// orderRecord is one record of the order-adversarial corpus.
type orderRecord struct {
	// Path is repo-relative, and is also the record's path inside orderDir.
	Path string
	// Marker is a distinctive string from the record's own text, so the eval can
	// tell the record's CONTENT arriving from its name arriving. A manifest names
	// what an assembly says it passed; only the bundle says what it passed.
	Marker string
	// Why states which comparator disagreement this record exists to expose.
	Why string
}

// orderRecords is the corpus in CREATION order, which is deliberately not its
// sorted order under any of the four comparators below. materialiseOrderPair
// writes the files in exactly this sequence.
//
// Writing them in that sequence is belt to the braces rather than the reach
// itself: `os.ReadDir` sorts by name, so no assembler reachable through this
// harness can observe creation order, and the creation-order row exists for an
// enumeration that does not go through it.
var orderRecords = []orderRecord{
	{
		Path:   ".abcd/development/brief/glossary/ordering/item-2-term.md",
		Marker: "A one-digit suffix, which a natural-order sort would put before the two-digit one.",
		Why:    "the one-digit suffix; a numeric-suffix sort puts it before item-10, byte order after",
	},
	{
		Path:   ".abcd/development/brief/glossary/ordering/Zulu-term.md",
		Marker: "An uppercase name, so byte order and case-insensitive order disagree about it.",
		Why:    "the uppercase name; byte order puts it first, a case-folded sort last",
	},
	{
		Path:   ".abcd/development/brief/glossary/ordering/item-10-term.md",
		Marker: "A two-digit suffix, which byte order puts before the one-digit suffix.",
		Why:    "the two-digit suffix, the other half of the numeric-suffix disagreement",
	},
	{
		Path:   ".abcd/development/brief/glossary/ordering/beta-term.md",
		Marker: "A second lowercase name, so the fold-order disagreement spans more than one pair.",
		Why:    "a second lowercase name, so the fold disagreement is not a single adjacent swap",
	},
	{
		Path:   ".abcd/development/brief/glossary/ordering/alpha-term.md",
		Marker: "A lowercase name that sorts after the uppercase one by byte and before it by fold.",
		Why:    "the lowercase name the uppercase one is compared against",
	},
	{
		Path:   ".abcd/development/brief/glossary/ordering.md",
		Marker: "A file whose name shares its stem with the directory beside it",
		Why: "the file sharing a stem with the directory beside it; byte order puts `.` before " +
			"`/` and so puts this file first, while a walk that compares path components " +
			"descends into the directory first — and component order is what a directory walk " +
			"yields, so it is the order an assembler most easily produces by dropping its sort",
	},
}

// requireOrderTable refuses a corpus table that has changed size behind the
// assertions consuming it, on the idiom requireOracleTables carries: a
// greater-than-zero floor on a table whose size is known is not a floor, and the
// count is duplicated rather than derived because a derived count agrees with
// the table by construction.
func requireOrderTable(t *testing.T) {
	t.Helper()
	if got, want := len(orderRecords), 6; got != want {
		t.Fatalf("the orderRecords table holds %d row(s), and this eval is written against %d; "+
			"the order oracle's whole capacity to catch a wrong comparator lives in these "+
			"names, so update the declared count deliberately", got, want)
	}
}

// orderPaths returns the corpus's repo-relative paths in creation order.
func orderPaths() []string {
	out := make([]string, 0, len(orderRecords))
	for _, r := range orderRecords {
		out = append(out, r.Path)
	}
	return out
}

// materialiseOrderPair materialises the order-adversarial corpus, commits it
// once, and returns TWO fixtures over that ONE commit at two distinct absolute
// paths.
//
// Two paths rather than two runs in one directory: a run-to-run comparison in
// one directory cannot see an absolute path or a temporary-directory name
// embedded in the output, and that leak is both a determinism failure and a
// breach of the repository's rule that no absolute local path enters an
// artefact. The second tree is a copy of the first INCLUDING `.git`, so the two
// runs see the identical commit rather than two commits of identical content.
func materialiseOrderPair(t *testing.T) (fixture, fixture) {
	t.Helper()
	requireOracleTables(t)
	requireOrderTable(t)

	base := t.TempDir()
	first := fixture{
		Root:    filepath.Join(base, "first", "repo"),
		Home:    filepath.Join(base, "first", "home"),
		Variant: "order",
	}
	copyTree(t, baselineDir, first.Root)
	// In declared creation order, one file at a time, so the corpus's creation
	// order is a fact about the materialised tree rather than a claim in a
	// comment.
	for _, r := range orderRecords {
		copyFile(t,
			filepath.Join(orderDir, filepath.FromSlash(r.Path)),
			filepath.Join(first.Root, filepath.FromSlash(r.Path)))
	}
	gitCommitFixture(t, first.Root)
	// The order corpus is the baseline plus its own records, so it carries the
	// baseline's two widening runs and needs their commit markers for the same
	// reason: without one the comparative position has no candidate set to
	// derive and cannot be ordered at all (adr-2609021016272867).
	stampFixtureRuns(t, first.Root)
	first.RootSHA = rootCommit(t, first.Root)
	copyTree(t, fixtureHomeDir, first.Home)
	renamePlaceholder(t, first.Home, first.RootSHA)

	second := fixture{
		Root:    filepath.Join(base, "second", "repo"),
		Home:    filepath.Join(base, "second", "home"),
		RootSHA: first.RootSHA,
		Variant: "order",
	}
	// `.git` travels with the tree, so the second assembly runs over the same
	// commit object rather than a re-commit of the same bytes.
	copyTree(t, first.Root, second.Root)
	copyTree(t, fixtureHomeDir, second.Home)
	renamePlaceholder(t, second.Home, second.RootSHA)
	if sha := rootCommit(t, second.Root); sha != first.RootSHA {
		t.Fatalf("the copied tree reports root commit %s, the original %s; the two assemblies "+
			"must run over ONE commit, or a difference between them says nothing about the "+
			"assembler", sha, first.RootSHA)
	}
	return first, second
}

// TestFixtureOrderIsAdversarial is the anti-coincidence guard under the order
// oracle: the corpus on disk is the corpus declared, and its names genuinely
// separate the four comparators an assembler might plausibly have used.
//
// Without it the oracle can pass because the corpus happens to sort the same way
// under every comparator, which is a green for a check that checked nothing.
func TestFixtureOrderIsAdversarial(t *testing.T) {
	requireOrderTable(t)

	// The corpus on disk is the corpus declared. A record that vanished would
	// silently take its comparator disagreement with it.
	var found []string
	err := filepath.WalkDir(orderDir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, rerr := filepath.Rel(orderDir, p)
		if rerr != nil {
			return rerr
		}
		found = append(found, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatalf("walking the order corpus at %s: %v", orderDir, err)
	}
	declared := append([]string{}, orderPaths()...)
	sort.Strings(declared)
	sort.Strings(found)
	if strings.Join(found, ",") != strings.Join(declared, ",") {
		t.Fatalf("the order corpus on disk holds %v, and this eval is written against %v",
			found, declared)
	}

	// Each record's marker is in its own file, so the bundle-content assertion
	// downstream is looking for text that exists.
	for _, r := range orderRecords {
		data, rerr := os.ReadFile(filepath.Join(orderDir, filepath.FromSlash(r.Path)))
		if rerr != nil {
			t.Fatalf("reading the order record %s: %v", r.Path, rerr)
		}
		if !bytes.Contains(data, []byte(r.Marker)) {
			t.Errorf("the order record %s does not carry its own declared marker %q, so the "+
				"content assertion over it would be looking for text that is not there",
				r.Path, r.Marker)
		}
	}

	creation := orderPaths()
	byteOrder := sortedCopy(creation, func(a, b string) bool { return a < b })
	foldOrder := sortedCopy(creation, func(a, b string) bool {
		la, lb := strings.ToLower(a), strings.ToLower(b)
		if la != lb {
			return la < lb
		}
		return a < b
	})
	naturalOrder := sortedCopy(creation, naturalLess)
	componentOrder := sortedCopy(creation, componentLess)

	for _, c := range []struct {
		name  string
		order []string
		why   string
	}{
		{
			name:  "creation order",
			order: creation,
			why: "an assembler that emitted its walk in creation order — no sort at all — " +
				"would be indistinguishable from a sorted one",
		},
		{
			name:  "case-insensitive order",
			order: foldOrder,
			why: "an assembler sorting case-insensitively is consistent but not lexicographic, " +
				"and the oracle could not tell the two apart",
		},
		{
			name:  "numeric-suffix order",
			order: naturalOrder,
			why: "an assembler sorting numeric suffixes naturally is consistent but not " +
				"lexicographic, and the oracle could not tell the two apart",
		},
		{
			name:  "directory-walk order",
			order: componentOrder,
			why: "an assembler comparing path COMPONENTS rather than bytes is emitting exactly " +
				"what a directory walk yields, which is the order dropping the sort produces " +
				"and so the likeliest wrong order of the four",
		},
	} {
		if strings.Join(c.order, ",") == strings.Join(byteOrder, ",") {
			t.Errorf("the order corpus's %s is identical to its byte order (%v); %s",
				c.name, byteOrder, c.why)
		}
	}
}

// TestWalkOrderIsLexicographic is the order oracle: at every position, the item
// paths the assembler wrote agree with the eval's own lexicographic sort of
// them.
//
// The manifest is where the paths are — the bundle carries item keys and text —
// so the sequence is read there, and the bundle's own item-key sequence is
// required to match it first. Without that binding the oracle would be checking
// the order of a description rather than the order of the thing described: a
// manifest listing paths in perfect order while the bundle carried its items in
// another would pass.
func TestWalkOrderIsLexicographic(t *testing.T) {
	first, _ := materialiseOrderPair(t)
	for _, position := range assemblingPositions {
		t.Run(position, func(t *testing.T) {
			a := assemble(t, first, position)
			if err := nonVacuous(a); err != nil {
				t.Fatalf("the assembly at %s cannot support an order assertion: %v", position, err)
			}
			requireItemSequencesAgree(t, a)

			paths := make([]string, 0, len(a.ManifestItems))
			for _, it := range a.ManifestItems {
				paths = append(paths, it.Path)
			}
			// The oracle: the eval's own sort, on a copy, never the assembler's.
			want := sortedCopy(paths, func(x, y string) bool { return x < y })
			for i := range paths {
				if paths[i] == want[i] {
					continue
				}
				t.Fatalf("the assembly at %s puts %q at index %d where a lexicographic order "+
					"puts %q; the assembler's order is not the eval's sort\n  assembled: %v\n  sorted:    %v",
					position, paths[i], i, want[i], paths, want)
			}
		})
	}
}

// requireItemSequencesAgree binds the manifest's item order to the bundle's.
func requireItemSequencesAgree(t *testing.T, a assembled) {
	t.Helper()
	if len(a.Items) != len(a.ManifestItems) {
		t.Fatalf("the assembly at %s carries %d bundle item(s) and %d manifest item(s); the "+
			"manifest is the only place the paths are, so it can only speak for the bundle's "+
			"order if the two are the same sequence",
			a.Position, len(a.Items), len(a.ManifestItems))
	}
	for i := range a.Items {
		if a.Items[i].ItemKey == a.ManifestItems[i].ItemKey {
			continue
		}
		t.Fatalf("the assembly at %s has bundle item %d keyed %q and manifest item %d keyed %q; "+
			"an order assertion over the manifest says nothing about the bundle unless the two "+
			"sequences agree", a.Position, i, a.Items[i].ItemKey, i, a.ManifestItems[i].ItemKey)
	}
}

// nonVacuous refuses an assembly that cannot support an identity or order
// assertion: two artefacts that are identical because both are empty are green
// and worthless, and so is a sorted sequence with nothing adversarial in it.
//
// It returns an error rather than failing, so TestComparatorReportsADifference
// can assert that it actually refuses an empty assembly rather than promising it.
func nonVacuous(a assembled) error {
	if len(a.Items) == 0 {
		return errVacuous("the assembly carries no items at all")
	}
	seen := map[string]bool{}
	for _, it := range a.ManifestItems {
		seen[it.Path] = true
	}
	// The comparative position reads none of the order corpus, and that is the
	// withdrawal adr-2609021016272867 makes rather than a hole: its two sources
	// are the derived widening run's candidates and the criteria discipline. So
	// its non-vacuity is asked over what it DOES carry — the three candidate
	// records, whose ids are what an ordering assertion there is about — and an
	// assembly that lost them is refused here exactly as one that lost the order
	// records is.
	if a.Position == posComparative {
		for _, id := range derivedCandidateItems {
			rel := ".abcd/work/issues/readings/" + derivedCandidateRun + "/" + id + ".md"
			if !seen[rel] {
				return errVacuous("the comparative assembly does not carry the candidate record " +
					rel + ", so the ordering the derived run's items are placed in is not in this " +
					"assembly")
			}
		}
		if !bytes.Contains(a.BundleRaw, []byte(sentinelPrefix+"CANDIDATE")) {
			return errVacuous("the comparative assembly names the candidate records in its " +
				"manifest but the bundle does not carry their text; a manifest names what an " +
				"assembly says it passed, not what it passed")
		}
		return nil
	}
	for _, r := range orderRecords {
		if !seen[r.Path] {
			return errVacuous("the assembly does not carry the order record " + r.Path +
				" (" + r.Why + "), so the comparator disagreement it exists to expose is not in this assembly")
		}
		if !bytes.Contains(a.BundleRaw, []byte(r.Marker)) {
			return errVacuous("the assembly names the order record " + r.Path +
				" in its manifest but the bundle does not carry its text (" + r.Marker +
				"); a manifest names what an assembly says it passed, not what it passed")
		}
	}
	return nil
}

// errVacuous is the error nonVacuous returns, named so a reader of a failure
// knows the assertion was refused rather than broken.
type errVacuous string

func (e errVacuous) Error() string { return string(e) }

// sortedCopy sorts a COPY under the given comparator, so no assertion here can
// reorder the sequence it is judging.
func sortedCopy(in []string, less func(a, b string) bool) []string {
	out := append([]string{}, in...)
	sort.SliceStable(out, func(i, j int) bool { return less(out[i], out[j]) })
	return out
}

// componentLess compares two paths component by component, which is the order a
// directory walk yields: a separator sorts before every other byte there,
// whereas a byte comparison puts `.` before `/` and so puts a file ahead of the
// directory it shares a stem with. Like naturalLess it exists only to prove the
// corpus separates the comparator from byte order; the oracle itself sorts by
// byte.
func componentLess(a, b string) bool {
	as, bs := strings.Split(a, "/"), strings.Split(b, "/")
	for i := 0; i < len(as) && i < len(bs); i++ {
		if as[i] != bs[i] {
			return as[i] < bs[i]
		}
	}
	return len(as) < len(bs)
}

// naturalLess compares two paths treating runs of digits as numbers, which is
// the "numeric suffix" order a plausible assembler might have used and which the
// order corpus exists to separate from byte order. It is used only to prove the
// corpus separates the two comparators; the oracle itself sorts by byte.
func naturalLess(a, b string) bool {
	ai, bi := 0, 0
	for ai < len(a) && bi < len(b) {
		ad, bd := unicode.IsDigit(rune(a[ai])), unicode.IsDigit(rune(b[bi]))
		if ad && bd {
			aj, bj := ai, bi
			for aj < len(a) && unicode.IsDigit(rune(a[aj])) {
				aj++
			}
			for bj < len(b) && unicode.IsDigit(rune(b[bj])) {
				bj++
			}
			an := strings.TrimLeft(a[ai:aj], "0")
			bn := strings.TrimLeft(b[bi:bj], "0")
			if len(an) != len(bn) {
				return len(an) < len(bn)
			}
			if an != bn {
				return an < bn
			}
			ai, bi = aj, bj
			continue
		}
		if a[ai] != b[bi] {
			return a[ai] < b[bi]
		}
		ai++
		bi++
	}
	return len(a)-ai < len(b)-bi
}
