//go:build smoke || coldreading

package evals

// The amnesia eval: the same repository state, assembled twice, produces the
// same assembled input.
//
// Amnesia is a property of what the assembler PASSES, not an instruction an
// agent can be trusted to follow, so it is checked here rather than exhibited in
// a case run. The identity relation is byte-equality of the assembled input with
// the manifest excluded, because the manifest legitimately carries a run
// identifier that differs between runs. The manifest is not therefore
// unasserted: it is held to two weaker properties — no timestamp-shaped key or
// scalar (here), and item paths in lexicographic order
// (coldreading_order_test.go).
//
// An identity assertion fails the way an absence assertion fails: a comparator
// that compares nothing, or two artefacts that agree because both are empty, is
// green and worthless. Three guards stand under it. nonVacuous refuses an
// assembly that lost the order corpus. The two runs are required to carry
// DIFFERENT run identifiers, so the comparison is over two invocations rather
// than one file read twice. And TestComparatorReportsADifference feeds the
// comparator artefacts that differ, and demands it name the item that differs.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// TestAssembledInputIsByteIdenticalAcrossRuns is ac-1: one repository state at
// one commit, assembled twice from two distinct absolute paths, produces
// byte-identical assembled input, the manifest excluded.
//
// The two paths are the strengthening this eval decided on itd-187's criterion.
// A run-to-run comparison in one directory cannot see an absolute path or a
// temporary-directory name embedded in the output, and that leak is both a
// determinism failure and a breach of the rule that no absolute local path
// enters an artefact. Comparing across paths catches it on the first run, and
// the subtest below refuses either artefact that names its own tree at all.
func TestAssembledInputIsByteIdenticalAcrossRuns(t *testing.T) {
	first, second := materialiseOrderPair(t)
	if first.Root == second.Root {
		t.Fatal("both assemblies would run at the same path, so the comparison could not see " +
			"an absolute path embedded in the output")
	}
	for _, position := range assemblingPositions {
		t.Run(position, func(t *testing.T) {
			a := assemble(t, first, position)
			b := assemble(t, second, position)

			// Freshness. Two invocations mint two run identifiers, so one
			// identifier on both sides would mean one artefact was read twice —
			// the classic silent pass for an identity assertion.
			//
			// The second assembly is retried once before that is believed. A run
			// identifier is a one-second stamp and a uniform four-digit draw, and
			// the two assemblies land in the same second, so a collision is a
			// documented one-in-ten-thousand outcome of the mint rather than
			// evidence of anything. Failing on the first collision would fail a
			// required CI job about once in every few thousand runs and blame the
			// assembler for what the mint is specified to do; two collisions in a
			// row is one in a hundred million, which is a real finding.
			if runIdentifier(t, a) == runIdentifier(t, b) {
				b = assemble(t, second, position)
			}
			if ra, rb := runIdentifier(t, a), runIdentifier(t, b); ra == rb {
				t.Fatalf("both assemblies at %s report run identifier %q twice over, so this is "+
					"one run compared with itself rather than two", position, ra)
			}

			for _, side := range []struct {
				name string
				a    assembled
			}{{"first", a}, {"second", b}} {
				if err := nonVacuous(side.a); err != nil {
					t.Fatalf("the %s assembly at %s cannot support an identity assertion: %v",
						side.name, position, err)
				}
			}

			// The path leak first. It is a difference the byte comparison also
			// catches, but only the byte comparison's message would be read as a
			// determinism fault when it is also a privacy one — and the manifest,
			// which the byte comparison excludes, is not covered by it at all.
			t.Run("no-absolute-path-in-either-artefact", func(t *testing.T) {
				for _, side := range []struct {
					name string
					f    fixture
					a    assembled
				}{{"first", first, a}, {"second", second, b}} {
					known := fixtureAbsolutePaths(t, side.f)
					for _, art := range []struct {
						name string
						raw  []byte
					}{{bundleFile, side.a.BundleRaw}, {manifestFile, side.a.ManifestRaw}} {
						for _, leak := range absolutePathLeaks(known, art.raw) {
							t.Errorf("the %s assembly's %s at %s %s; an absolute local path "+
								"in an artefact is both a determinism failure and a privacy one",
								side.name, art.name, position, leak)
						}
					}
				}
			})

			if diffs := compareArtefacts(bundleFile, a.BundleRaw, b.BundleRaw); len(diffs) > 0 {
				t.Fatalf("the assembled input at %s differs between two assemblies of ONE commit "+
					"at two paths (%d difference(s)):\n%s\nthis is the assembler failing to be "+
					"deterministic, not the eval being strict", position, len(diffs), reportDifferences(diffs))
			}
		})
	}
}

// ---------------------------------------------------------------------------
// The absolute-path detector.
// ---------------------------------------------------------------------------

// leakPath is one absolute path an artefact must not name, and what it is.
type leakPath struct {
	What string
	Path string
}

// absolutePathShape matches an absolute path by SHAPE rather than by name: a
// POSIX path of two or more components, or a Windows drive path.
//
// It is the second of the two mechanisms, and it is a second MECHANISM rather
// than a wider list on purpose. The list below knows the paths this harness
// created; this knows the shape of any other — the machine's own home directory,
// a checkout path, a cache directory — none of which the list could enumerate.
//
// The leading boundary is a CONSUMED character because RE2 has no lookbehind,
// and without one an ordinary repo-relative path matches on its own second
// slash: `.abcd/development/brief` contains `/development/brief`. `/` and `:`
// are excluded from the boundary class so a URL's `//host/path` is not read as a
// filesystem path.
var absolutePathShape = regexp.MustCompile(
	`(?:^|[^A-Za-z0-9._~+%:/-])((?:/[A-Za-z0-9._~+%-]+){2,}|[A-Za-z]:\\[A-Za-z0-9._~+%\\-]+)`)

// fixtureAbsolutePaths returns every absolute path the fixture occupies: the
// repository root, the HOME, and EVERY ancestor of either up to and including
// the process temporary directory they were created under.
//
// The ancestors are the repair. A two-string check over the two roots misses the
// leak class the two-path design exists to close, one level up: both trees are
// made under ONE temporary parent, so that parent is a real absolute local path
// that both runs carry IDENTICALLY — the byte comparison agrees about it and
// still reports nothing, and a check looking for the two leaves is not looking
// for it.
//
// Bounding the walk at the process temporary directory is not a threshold that
// can be raised again. It is where this harness's own directories stop and the
// machine's begin, and `t.TempDir` creates under exactly that directory, so a
// fixture outside it is a broken assumption rather than a case to widen for.
func fixtureAbsolutePaths(t *testing.T, f fixture) []leakPath {
	t.Helper()
	tmp := filepath.Clean(os.TempDir())
	if tmp == string(filepath.Separator) || tmp == "." {
		t.Fatalf("the process temporary directory is %q, so the ancestor sweep has no bound; "+
			"this guard cannot be run without one", tmp)
	}
	seen := map[string]bool{}
	var out []leakPath
	add := func(what, p string) {
		p = filepath.Clean(p)
		if seen[p] {
			return
		}
		seen[p] = true
		out = append(out, leakPath{What: what, Path: p})
	}
	for _, leaf := range []struct{ what, path string }{
		{"its repository root", f.Root},
		{"the HOME it ran under", f.Home},
	} {
		clean := filepath.Clean(leaf.path)
		if !underOrEqual(clean, tmp) {
			t.Fatalf("%s is at %s, which is not under the process temporary directory; the "+
				"ancestor sweep is bounded there and cannot be bounded anywhere else",
				leaf.what, elidePath(clean))
		}
		add(leaf.what, clean)
		for dir := filepath.Dir(clean); underOrEqual(dir, tmp); dir = filepath.Dir(dir) {
			add("a directory the fixtures were created under", dir)
			if dir == tmp {
				break
			}
		}
	}
	return out
}

// underOrEqual reports whether p is base or lies beneath it.
func underOrEqual(p, base string) bool {
	return p == base || strings.HasPrefix(p, base+string(filepath.Separator))
}

// absolutePathLeaks reports every absolute local path an artefact names: the
// fixture's own directories and their ancestors by exact match, and anything
// else that is merely SHAPED like one.
//
// The two mechanisms overlap deliberately. A known path is reported with what it
// is, which is the message an operator can act on; a shaped one is reported
// because no list of known paths can enumerate the machine's own.
func absolutePathLeaks(known []leakPath, raw []byte) []string {
	text := string(raw)
	var out []string
	var reported []string
	for _, k := range known {
		if !strings.Contains(text, k.Path) {
			continue
		}
		out = append(out, fmt.Sprintf("names %s (%s)", k.What, elidePath(k.Path)))
		reported = append(reported, k.Path)
	}
	shaped := map[string]bool{}
	for _, loc := range absolutePathShape.FindAllStringSubmatchIndex(text, -1) {
		hit := text[loc[2]:loc[3]]
		if shaped[hit] {
			continue
		}
		shaped[hit] = true
		// A root-relative markdown link target is a link, not a filesystem path,
		// and it is a shape the corpus can legitimately grow. Skipping it costs
		// the shape mechanism nothing that matters: the paths this harness made
		// are matched by NAME above, with no boundary rule at all, so one of them
		// written inside a link is still reported. The two mechanisms cover each
		// other, which is why neither has to be widened to do the other's job.
		if loc[2] >= 2 && text[loc[2]-2:loc[2]] == "](" {
			continue
		}
		covered := false
		for _, r := range reported {
			if strings.Contains(r, hit) {
				covered = true
				break
			}
		}
		if covered {
			continue
		}
		out = append(out, fmt.Sprintf("carries the absolute path %s", elidePath(hit)))
	}
	return out
}

// elidePath renders an absolute path as its last two components alone.
//
// The failure message has to name the leak to be usable and must not itself
// publish the directories ABOVE it into a CI log, which is where an account name
// lives and which is the rule the guard is enforcing. Two components identify
// which path leaked; everything above them is dropped.
//
// A path of two components or fewer is returned whole, because there is nothing
// above the last two to drop. `/tmp` is the case that matters — it is what
// `os.TempDir()` reports on the Linux runner, and it names no account.
func elidePath(p string) string {
	parts := strings.Split(filepath.ToSlash(filepath.Clean(p)), "/")
	if len(parts) <= 2 {
		return p
	}
	// The ellipsis carries NO trailing separator, deliberately: `".../" + "tmp/x"`
	// re-forms `/tmp/x`, so an elided path would still contain the path it elided
	// and the elision would be cosmetic.
	return "..." + strings.Join(parts[len(parts)-2:], "/")
}

// TestTheAbsolutePathGuardSeesMoreThanTheTwoRoots is the guard's own falsifier.
//
// The guard it replaces looked for the two fixture roots by name, which misses
// the leak class the two-path design exists to close, one level up. Both trees
// are created under ONE temporary parent: planting that parent leaves the byte
// comparison green, because both runs carry the same string and therefore still
// agree, and a two-string check is not looking for it. The first row below is
// that exact plant.
//
// The negative rows are the other half. A detector that reports every slash
// reports nothing: the repo-relative paths an artefact legitimately carries, and
// a URL, must both stay clean.
func TestTheAbsolutePathGuardSeesMoreThanTheTwoRoots(t *testing.T) {
	base := t.TempDir()
	f := fixture{
		Root:    filepath.Join(base, "first", "repo"),
		Home:    filepath.Join(base, "first", "home"),
		Variant: "order",
	}
	known := fixtureAbsolutePaths(t, f)

	// The mechanism claim, stated separately from the detection: the sweep has
	// to KNOW the shared parent, or the first row below would be passing on the
	// shape detector alone and the ancestor half would be untested.
	held := false
	for _, k := range known {
		if k.Path == filepath.Clean(base) {
			held = true
		}
	}
	if !held {
		t.Fatalf("the ancestor sweep over a fixture at %s does not name the temporary parent "+
			"both trees are created under; that parent is the leak the two-string check missed",
			elidePath(f.Root))
	}

	for _, tc := range []struct {
		name string
		text string
		want bool
	}{
		{"the shared parent both trees are created under", `{"note":"` + base + `"}`, true},
		{"the repository root", `{"note":"` + f.Root + `"}`, true},
		{"the HOME it ran under", `{"note":"` + f.Home + `"}`, true},
		{"a path inside the fixture tree", `{"note":"` + filepath.Join(f.Root, "main.go") + `"}`, true},
		{"an absolute path this harness never made", `{"note":"/opt/elsewhere/cache"}`, true},
		{"repo-relative item paths", `{"path":".abcd/development/brief/01-product/01-press-release.md"}`, false},
		{"a repo-relative source path", `{"path":"internal/core/reading/include.go"}`, false},
		{"a URL", `{"source":"https://example.invalid/one/two"}`, false},
		{"a root-relative markdown link target", `{"text":"see [the map](/docs/reference/thing.md)"}`, false},
		{"a single-component reference", `{"note":"and/or a/b"}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := absolutePathLeaks(known, []byte(tc.text))
			switch {
			case tc.want && len(got) == 0:
				t.Errorf("the absolute-path guard reports nothing over %q, which carries one; "+
					"a guard that cannot see it is the two-string check again", tc.text)
			case !tc.want && len(got) > 0:
				t.Errorf("the absolute-path guard reports %v over %q, which carries no absolute "+
					"path at all; a detector that reports every slash reports nothing", got, tc.text)
			}
			// The message names the leak without republishing the directories
			// above it, which is the rule this guard is enforcing.
			//
			// The property is ELISION, not the absence of some substring: on the
			// Linux runner `os.TempDir()` is `/tmp`, and every fixture path
			// legitimately carries `tmp` as a component, so "the message must not
			// contain the temporary directory" is unsatisfiable there and would
			// fail a lane `make preflight` does not reach. What must hold is that
			// no path elidePath shortened appears in full.
			for _, g := range got {
				for _, k := range known {
					if elidePath(k.Path) == k.Path {
						continue // nothing above the last two components to drop
					}
					if strings.Contains(g, k.Path) {
						t.Errorf("the failure message %q carries %s in full rather than its last "+
							"two components; a guard against absolute paths in artefacts must not "+
							"put one in a CI log itself", g, elidePath(k.Path))
					}
				}
			}
		})
	}
}

// runIdentifier reads the manifest's run identifier, which is the one field the
// assembler is expected to vary between two runs of the same state.
func runIdentifier(t *testing.T, a assembled) string {
	t.Helper()
	var m struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal(a.ManifestRaw, &m); err != nil {
		t.Fatalf("decoding the manifest at %s: %v", a.Position, err)
	}
	if m.RunID == "" {
		t.Fatalf("the manifest at %s carries no run identifier, so two runs cannot be told "+
			"apart and an identity assertion cannot prove it compared two of them", a.Position)
	}
	return m.RunID
}

// ---------------------------------------------------------------------------
// The comparator.
// ---------------------------------------------------------------------------

// artefactDifference is one reported difference: where it is, and what it is.
type artefactDifference struct {
	Where  string
	Detail string
}

func (d artefactDifference) String() string { return d.Where + ": " + d.Detail }

// reportDifferences renders a comparison's findings, one per line.
func reportDifferences(ds []artefactDifference) string {
	out := make([]string, 0, len(ds))
	for _, d := range ds {
		out = append(out, "  - "+d.String())
	}
	return strings.Join(out, "\n")
}

// compareArtefacts is the comparison ac-1 rests on. The RELATION it asserts is
// byte-equality of the whole artefact: equal bytes report nothing, and anything
// else reports at least one difference. The decoded walk exists only to say
// WHERE, because "the bytes differ" names no way back to the item that caused
// it.
//
// A structural comparison alone would tolerate exactly the re-serialisation this
// eval exists to refuse, which is why the byte check comes first and why unequal
// bytes with an equal structure are still reported.
func compareArtefacts(name string, left, right []byte) []artefactDifference {
	if string(left) == string(right) {
		return nil
	}
	var l, r any
	lerr := decodeArtefact(left, &l)
	rerr := decodeArtefact(right, &r)
	if lerr != nil || rerr != nil {
		return []artefactDifference{{
			Where:  name,
			Detail: fmt.Sprintf("the two artefacts differ and at least one does not decode (left: %v, right: %v)", lerr, rerr),
		}}
	}
	diffs := diffValue(name, l, r)
	if len(diffs) == 0 {
		// Equal structure, unequal bytes: unstable serialisation — key order,
		// whitespace, escaping. A structural comparison would call this identical,
		// and it is not.
		diffs = append(diffs, artefactDifference{
			Where: name,
			Detail: fmt.Sprintf("the two artefacts decode identically but differ in their bytes at offset %d, "+
				"so the serialisation itself is unstable", firstByteDifference(left, right)),
		})
	}
	return diffs
}

// decodeArtefact decodes with numbers left as their literal text, so a
// comparison never turns on float formatting.
func decodeArtefact(raw []byte, into *any) error {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	return dec.Decode(into)
}

// firstByteDifference reports the offset of the first differing byte, or the
// length of the shorter artefact when one is a prefix of the other.
func firstByteDifference(a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}

// diffValue walks two decoded artefacts in parallel and reports every scalar
// that differs, labelling an item by its own key so the report names the item
// rather than an index a reader has to go and count.
func diffValue(where string, l, r any) []artefactDifference {
	switch lv := l.(type) {
	case map[string]any:
		rv, ok := r.(map[string]any)
		if !ok {
			return []artefactDifference{{Where: where, Detail: fmt.Sprintf("an object on the left, %s on the right", kindOf(r))}}
		}
		keys := map[string]bool{}
		for k := range lv {
			keys[k] = true
		}
		for k := range rv {
			keys[k] = true
		}
		names := make([]string, 0, len(keys))
		for k := range keys {
			names = append(names, k)
		}
		sort.Strings(names)
		var out []artefactDifference
		for _, k := range names {
			lval, lok := lv[k]
			rval, rok := rv[k]
			switch {
			case lok && !rok:
				out = append(out, artefactDifference{Where: where + "." + k, Detail: "present on the left, absent on the right"})
			case !lok && rok:
				out = append(out, artefactDifference{Where: where + "." + k, Detail: "absent on the left, present on the right"})
			default:
				out = append(out, diffValue(where+"."+k, lval, rval)...)
			}
		}
		return out
	case []any:
		rv, ok := r.([]any)
		if !ok {
			return []artefactDifference{{Where: where, Detail: fmt.Sprintf("an array on the left, %s on the right", kindOf(r))}}
		}
		var out []artefactDifference
		if len(lv) != len(rv) {
			out = append(out, artefactDifference{
				Where:  where,
				Detail: fmt.Sprintf("%d element(s) on the left, %d on the right", len(lv), len(rv)),
			})
		}
		n := len(lv)
		if len(rv) < n {
			n = len(rv)
		}
		for i := 0; i < n; i++ {
			out = append(out, diffValue(elementLabel(where, i, lv[i], rv[i]), lv[i], rv[i])...)
		}
		return out
	case nil:
		if r == nil {
			return nil
		}
		return []artefactDifference{{Where: where, Detail: fmt.Sprintf("null on the left, %s on the right", kindOf(r))}}
	default:
		if fmt.Sprintf("%v", l) == fmt.Sprintf("%v", r) && kindOf(l) == kindOf(r) {
			return nil
		}
		return []artefactDifference{{
			Where:  where,
			Detail: fmt.Sprintf("%q on the left, %q on the right", fmt.Sprintf("%v", l), fmt.Sprintf("%v", r)),
		}}
	}
}

// elementLabel names an array element by its item key where it has one, so a
// reordering reports the items that moved rather than the indices they moved
// between.
func elementLabel(where string, i int, l, r any) string {
	lk, rk := itemKeyOf(l), itemKeyOf(r)
	switch {
	case lk == "" && rk == "":
		return fmt.Sprintf("%s[%d]", where, i)
	case lk == rk:
		return fmt.Sprintf("%s[%d](%s)", where, i, lk)
	default:
		return fmt.Sprintf("%s[%d](%s vs %s)", where, i, orNone(lk), orNone(rk))
	}
}

// itemKeyOf returns an element's item key, or empty for an element that has none.
func itemKeyOf(v any) string {
	m, ok := v.(map[string]any)
	if !ok {
		return ""
	}
	k, ok := m["item_key"].(string)
	if !ok {
		return ""
	}
	return k
}

func orNone(s string) string {
	if s == "" {
		return "no item key"
	}
	return s
}

// kindOf names a decoded value's shape for a difference message.
func kindOf(v any) string {
	switch v.(type) {
	case map[string]any:
		return "an object"
	case []any:
		return "an array"
	case string:
		return "a string"
	case json.Number:
		return "a number"
	case bool:
		return "a boolean"
	case nil:
		return "null"
	default:
		return "an unknown value"
	}
}

// ---------------------------------------------------------------------------
// The manifest timestamp scan.
// ---------------------------------------------------------------------------

// timestampKeyTokens are the underscore- or hyphen-separated key TOKENS that
// name a moment. Matching whole tokens rather than substrings is deliberate: a
// substring match on "date" calls `candidate_set` a timestamp.
var timestampKeyTokens = map[string]bool{
	"at": true, "clock": true, "created": true, "ctime": true, "date": true,
	"datetime": true, "day": true, "ended": true, "epoch": true, "finished": true,
	"generated": true, "modified": true, "mtime": true, "started": true,
	"stamp": true, "time": true, "timestamp": true, "ts": true, "updated": true,
	"when": true,
}

// The value shapes a moment takes. The two punctuated shapes cannot match a
// content hash, which is hex and carries neither hyphen nor colon; the packed
// shape can, which is why it carries the two declared exemptions below.
var (
	isoDatePattern     = regexp.MustCompile(`\d{4}-\d{2}-\d{2}`)
	clockPattern       = regexp.MustCompile(`\d{1,2}:\d{2}:\d{2}`)
	packedDigitPattern = regexp.MustCompile(`\d{8,}`)
	hexValuePattern    = regexp.MustCompile(`^[0-9a-f]{40}$|^[0-9a-f]{64}$`)
	// assemblerVersionPattern is the stamped version's whole shape: a semver
	// core with the include table's digest as build metadata. It is a third
	// DECLARED exemption, and it is needed because the digest is not a bare
	// hex value — hexValuePattern anchors on the whole string, so the "1.1.0+"
	// prefix put the digest outside every existing exemption while leaving it
	// squarely inside the packed-digit rule. A digest carrying eight
	// consecutive decimal digits would then be reported as a leaked moment, at
	// every position, on a lane preflight does not run.
	//
	// Matched by SHAPE, not by key name: an assembler_version that stopped
	// being a digest is scanned again rather than exempted by the name it
	// happens to go under.
	assemblerVersionPattern = regexp.MustCompile(`^\d+\.\d+\.\d+\+[0-9a-f]{64}$`)
	runIDPattern            = regexp.MustCompile(`^rdg-\d{16}$`)
	epochPattern            = regexp.MustCompile(`^\d{10,}$`)
	// recordIDPattern is a minted record identifier wherever it appears inside a
	// scalar: one of the family tags below, a hyphen, and the timestamp-numeric
	// ordinal the one allocator mints (adr-45).
	//
	// It is the FOURTH declared exemption from the packed-digit rule, and it is
	// the one the manifest's own contents force. spc-65 confines this scan to the
	// manifest because the manifest carries paths, field names and hashes only —
	// but a path names a record, and `<family>-<yymmddHHMMSS><4 random digits>` is
	// sixteen packed digits, so the premise was false for paths from the moment
	// the mint moved to timestamp-numeric ids (iss-2608311502474022). A scan that
	// reports the way this repository NAMES a record is a scan producing a false
	// finding on correct output, which is how an eval stops being read.
	//
	// Narrow in three ways, deliberately. The family list is CLOSED and
	// hand-transcribed from recordid's own grammar, so `run-20260831131824` — a
	// directory named after a moment — is not exempted by looking id-shaped. The
	// elision is TOKEN-LOCAL: only the identifier is removed, so a packed moment
	// sitting beside a record id in the same path is still reported. And it
	// exempts from the packed-digit rule alone, so an id that grew an ISO date or
	// a clock time still fails.
	//
	// It is matched by SHAPE rather than under a key name, like
	// assemblerVersionPattern and unlike runIDKey: the manifest carries record
	// ids in more than one field — an item's path, and a scope selector naming
	// the record a reading is about — and an exemption keyed on `path` would be
	// silently absent from the second.
	recordIDPattern = regexp.MustCompile(`(^|[^0-9A-Za-z])(?:adm|adr|dsp|iss|itd|rdg|rdi|spc|srp)-\d+`)
)

// elideRecordIDs removes every minted record identifier from a scalar, leaving
// the rest of it — including any other run of digits — to be scanned.
func elideRecordIDs(s string) string {
	return recordIDPattern.ReplaceAllString(s, "$1")
}

// runIDKey is the one key whose value is exempt from the packed-digit rule.
//
// The exemption is declared rather than silent, and it is narrow: the manifest's
// run identifier mints from a clock, so the manifest is not literally free of
// encoded time, and saying so is worth more than a scan that quietly skipped it.
// It is exempt from the PACKED-DIGIT rule alone — a run identifier that grew an
// ISO date or a clock time still fails. The exemption is keyed on the NAME at
// any depth, while the shape assertion reads the top-level field only, so a
// nested key spelled run_id would be exempt and unchecked. The Manifest struct
// is closed and declares no such field, which is what makes that unreachable
// rather than merely unobserved.
const runIDKey = "run_id"

// timestampFinding is one timestamp-shaped key or scalar in a manifest.
type timestampFinding struct {
	Where  string
	Detail string
}

func (f timestampFinding) String() string { return f.Where + ": " + f.Detail }

// scanManifestForTimestamps walks a manifest and reports every timestamp-shaped
// key and scalar in it, alongside the set of key names it actually visited.
//
// The visited set is the anti-vacuity half: a scanner that walked nothing would
// report nothing, which reads exactly like a clean manifest.
func scanManifestForTimestamps(raw []byte) ([]timestampFinding, map[string]bool, error) {
	var v any
	if err := decodeArtefact(raw, &v); err != nil {
		return nil, nil, err
	}
	seen := map[string]bool{}
	var findings []timestampFinding
	var walk func(where, key string, node any)
	walk = func(where, key string, node any) {
		if key != "" {
			seen[key] = true
			for _, token := range strings.FieldsFunc(key, func(r rune) bool { return r == '_' || r == '-' || r == '.' }) {
				if timestampKeyTokens[strings.ToLower(token)] {
					findings = append(findings, timestampFinding{
						Where:  where,
						Detail: fmt.Sprintf("the key names a moment (%q)", token),
					})
					break
				}
			}
		}
		switch n := node.(type) {
		case map[string]any:
			names := make([]string, 0, len(n))
			for k := range n {
				names = append(names, k)
			}
			sort.Strings(names)
			for _, k := range names {
				walk(where+"."+k, k, n[k])
			}
		case []any:
			for i, el := range n {
				walk(fmt.Sprintf("%s[%d]", where, i), "", el)
			}
		case string:
			switch {
			case isoDatePattern.MatchString(n):
				findings = append(findings, timestampFinding{Where: where, Detail: fmt.Sprintf("the value %q carries a calendar date", n)})
			case clockPattern.MatchString(n):
				findings = append(findings, timestampFinding{Where: where, Detail: fmt.Sprintf("the value %q carries a clock time", n)})
			case packedDigitPattern.MatchString(elideRecordIDs(n)) && key != runIDKey &&
				!hexValuePattern.MatchString(n) && !assemblerVersionPattern.MatchString(n):
				findings = append(findings, timestampFinding{Where: where, Detail: fmt.Sprintf("the value %q carries a packed run of digits, which is how a moment travels without punctuation", n)})
			}
		case json.Number:
			if epochPattern.MatchString(n.String()) {
				findings = append(findings, timestampFinding{Where: where, Detail: fmt.Sprintf("the value %s is large enough to be an epoch", n.String())})
			}
		}
	}
	walk("manifest", "", v)
	return findings, seen, nil
}

// manifestKeysScanned are the manifest keys the scan must have reached. It is a
// declaration rather than a derivation, on the idiom requireOracleTables
// carries: a scanner that visited nothing reports nothing, and reports nothing
// is what a clean manifest looks like.
var manifestKeysScanned = []string{
	"_type", "schema_version", "run_id", "position", "target_commit",
	"assembler_version", "items", "item_key", "path", "sha256",
	"exclusions", "rule", "signal", "detail",
}

// TestManifestCarriesNoTimestamp is ac-3: no key and no scalar in the manifest
// is timestamp-shaped.
//
// The scan is confined to the MANIFEST. Projected record bodies legitimately
// quote dates in prose, so a timestamp scan over the bundle would fire on the
// record's own text; the manifest carries paths, field names and hashes only, so
// a timestamp-shaped token there is unambiguously a defect.
func TestManifestCarriesNoTimestamp(t *testing.T) {
	first, _ := materialiseOrderPair(t)
	for _, position := range assemblingPositions {
		t.Run(position, func(t *testing.T) {
			a := assemble(t, first, position)
			if err := nonVacuous(a); err != nil {
				t.Fatalf("the assembly at %s cannot support a manifest assertion: %v", position, err)
			}
			findings, seen, err := scanManifestForTimestamps(a.ManifestRaw)
			if err != nil {
				t.Fatalf("decoding the manifest at %s: %v", position, err)
			}
			for _, key := range manifestKeysScanned {
				if !seen[key] {
					t.Fatalf("the timestamp scan of the manifest at %s never reached the key %q; "+
						"a scan that walked nothing reports nothing, which is what a clean "+
						"manifest also looks like", position, key)
				}
			}
			if len(findings) > 0 {
				lines := make([]string, 0, len(findings))
				for _, f := range findings {
					lines = append(lines, "  - "+f.String())
				}
				t.Fatalf("the manifest at %s carries %d timestamp-shaped key(s) or scalar(s):\n%s",
					position, len(findings), strings.Join(lines, "\n"))
			}
			// The one declared exemption, held to its shape so it cannot widen into
			// a free-text field carrying a moment.
			if id := runIdentifier(t, a); !runIDPattern.MatchString(id) {
				t.Fatalf("the manifest's run identifier at %s is %q, which is not the "+
					"rdg-<16 digits> shape the packed-digit exemption is written for; the "+
					"exemption is narrow by construction and this widens it", position, id)
			}
		})
	}

	// The scan's own capacity to fire. Without this, a scan that matched nothing
	// would pass every assertion above.
	t.Run("catches-a-planted-timestamp", func(t *testing.T) {
		for _, c := range []struct {
			name     string
			manifest string
			want     string
		}{
			{
				name:     "a timestamp-shaped key",
				manifest: `{"_type":"abcd.reading.manifest","generated_at":"whenever","items":[]}`,
				want:     "manifest.generated_at",
			},
			{
				name:     "a timestamp-shaped key on an item",
				manifest: `{"_type":"abcd.reading.manifest","items":[{"item_key":"itm-0001","mtime":"whenever"}]}`,
				want:     "manifest.items[0].mtime",
			},
			{
				name:     "a calendar date in a scalar",
				manifest: `{"_type":"abcd.reading.manifest","note":"cut 2026-08-31","items":[]}`,
				want:     "manifest.note",
			},
			{
				name:     "a clock time in a scalar",
				manifest: `{"_type":"abcd.reading.manifest","note":"cut at 13:18:24","items":[]}`,
				want:     "manifest.note",
			},
			{
				name:     "a packed moment in a scalar",
				manifest: `{"_type":"abcd.reading.manifest","note":"20260831131824","items":[]}`,
				want:     "manifest.note",
			},
			{
				// The other side of the record-id exemption below. The elision is
				// token-local, so a path that carries a packed moment BESIDE a
				// record id still reports it: what the exemption removes is the
				// identifier, never the run of digits next to it.
				name: "a packed moment in a path carrying a record id",
				manifest: `{"_type":"abcd.reading.manifest","items":[{"item_key":"itm-0001",` +
					`"path":".abcd/development/specs/open/spc-2609020626048722-20260831131824.md"}]}`,
				want: "manifest.items[0].path",
			},
			{
				// A family tag the mint does not use is not a record id, however
				// id-shaped it looks: `run-20260831131824` is a directory named
				// after a moment, which is exactly how a moment travels.
				name: "a packed moment behind a tag no record family mints",
				manifest: `{"_type":"abcd.reading.manifest","items":[{"item_key":"itm-0001",` +
					`"path":".abcd/.work.local/scratch/run-20260831131824/bundle.json"}]}`,
				want: "manifest.items[0].path",
			},
			{
				name:     "an epoch in a number",
				manifest: `{"_type":"abcd.reading.manifest","note":1756645104,"items":[]}`,
				want:     "manifest.note",
			},
		} {
			t.Run(c.name, func(t *testing.T) {
				findings, _, err := scanManifestForTimestamps([]byte(c.manifest))
				if err != nil {
					t.Fatalf("decoding the synthetic manifest: %v", err)
				}
				if len(findings) == 0 {
					t.Fatalf("the scan reported nothing over a manifest carrying %s, so it would "+
						"report nothing over a real one too", c.name)
				}
				var wheres []string
				for _, f := range findings {
					wheres = append(wheres, f.Where)
				}
				if !contains(wheres, c.want) {
					t.Fatalf("the scan reported %v over a manifest carrying %s, and none of them "+
						"names %s", wheres, c.name, c.want)
				}
			})
		}
	})

	// The premise the scoping decision rests on, held rather than assumed
	// (iss-2608311502474022). spc-65 confines this scan to the manifest because
	// the manifest carries paths, field names and hashes only, so a
	// timestamp-shaped token there is unambiguously a defect. A PATH, though,
	// carries record ids, and every minting family in this repository allocates
	// `<family>-<yymmddHHMMSS><4 random digits>` (adr-45) — sixteen digits, which
	// is a packed run of digits by any reading of the rule. The premise was
	// therefore already false: fed a real id, the unamended scan fired, and it
	// stayed quiet only because the fixture corpus names its records shortly
	// while the live specs directory does not.
	//
	// The ids below are real, taken from this repository's own record.
	t.Run("passes-a-record-id-in-a-path", func(t *testing.T) {
		for _, c := range []struct {
			name string
			path string
		}{
			{"an admitted spec", ".abcd/development/specs/open/spc-2609020626048722-a-committed-preset.md"},
			{"an admitted intent", ".abcd/development/intents/planned/itd-2608311051046981-a-required-check.md"},
			{"a capture, in a path an exclusion names", ".abcd/work/issues/open/iss-2608311502474022-the-amnesia-eval.md"},
		} {
			t.Run(c.name, func(t *testing.T) {
				manifest := `{"_type":"abcd.reading.manifest","items":[{"item_key":"itm-0001","path":"` +
					c.path + `"}]}`
				findings, _, err := scanManifestForTimestamps([]byte(manifest))
				if err != nil {
					t.Fatalf("decoding the synthetic manifest: %v", err)
				}
				if len(findings) > 0 {
					t.Fatalf("the scan reported %v over a manifest naming %s; a minted record id is "+
						"how this repository NAMES a record, not how a moment travels, and a scan "+
						"that calls one a leaked moment produces a false finding the first time "+
						"such a family is admitted", findings, c.path)
				}
			})
		}
	})

	// The other half: the scan does not fire on the shapes a manifest
	// legitimately carries. A scanner that reported everything would also pass
	// the subtest above, and would make the real assertion unusable.
	t.Run("passes-a-clean-manifest", func(t *testing.T) {
		clean := `{"_type":"abcd.reading.manifest","schema_version":1,"run_id":"rdg-2608311318244709",` +
			`"position":"widening","target_commit":"a8dd1373d1b6ba8651f38367af134ce484e8ad99",` +
			`"assembler_version":"1.0.0","items":[{"item_key":"itm-0001",` +
			`"path":".abcd/development/brief/glossary/ordering/alpha-term.md",` +
			`"sha256":"33e9afcc4366a0e7b93938896bc2afc37ed82aadf6f339e33e20c43a807d07ca"}],` +
			`"exclusions":[{"rule":"field projection","signal":"frontmatter key","detail":"origin"}]}`
		findings, _, err := scanManifestForTimestamps([]byte(clean))
		if err != nil {
			t.Fatalf("decoding the clean synthetic manifest: %v", err)
		}
		if len(findings) > 0 {
			t.Fatalf("the scan fired %d time(s) on a manifest carrying only the shapes a real one "+
				"carries — a run identifier, a commit sha, a path and a content hash: %v",
				len(findings), findings)
		}
	})
}

func contains(hay []string, needle string) bool {
	for _, s := range hay {
		if s == needle {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// The comparator meta-test.
// ---------------------------------------------------------------------------

// TestComparatorReportsADifference is ac-4, and it is the anti-vacuity guard
// under ac-1: a comparator that silently compared nothing would pass every other
// assertion in this file.
//
// It also checks the guard that closes the other way an identity assertion goes
// vacuous — two artefacts that agree because both are empty — by asserting that
// nonVacuous actually refuses such an assembly rather than promising to.
func TestComparatorReportsADifference(t *testing.T) {
	const twoItems = `{"_type":"abcd.reading.bundle","schema_version":1,"position":"widening","items":[
	  {"item_key":"itm-0001","kind":"glossary-term","text":"# Alpha term\n"},
	  {"item_key":"itm-0002","kind":"glossary-term","text":"# Beta term\n"}]}`
	const reordered = `{"_type":"abcd.reading.bundle","schema_version":1,"position":"widening","items":[
	  {"item_key":"itm-0002","kind":"glossary-term","text":"# Beta term\n"},
	  {"item_key":"itm-0001","kind":"glossary-term","text":"# Alpha term\n"}]}`
	const oneScalarChanged = `{"_type":"abcd.reading.bundle","schema_version":1,"position":"widening","items":[
	  {"item_key":"itm-0001","kind":"glossary-term","text":"# Alpha term\n"},
	  {"item_key":"itm-0002","kind":"glossary-term","text":"# Gamma term\n"}]}`
	const topLevelScalarChanged = `{"_type":"abcd.reading.bundle","schema_version":1,"position":"detection","items":[
	  {"item_key":"itm-0001","kind":"glossary-term","text":"# Alpha term\n"},
	  {"item_key":"itm-0002","kind":"glossary-term","text":"# Beta term\n"}]}`

	t.Run("identical-artefacts-report-nothing", func(t *testing.T) {
		if diffs := compareArtefacts(bundleFile, []byte(twoItems), []byte(twoItems)); len(diffs) > 0 {
			t.Fatalf("the comparator reported %d difference(s) between an artefact and itself, so "+
				"ac-1 would fail whatever the assembler did:\n%s", len(diffs), reportDifferences(diffs))
		}
	})

	for _, c := range []struct {
		name  string
		left  string
		right string
		names []string
		why   string
	}{
		{
			name:  "items-in-a-different-order",
			left:  twoItems,
			right: reordered,
			names: []string{"itm-0001", "itm-0002"},
			why:   "a walk whose order changed between runs is the failure ac-1 exists to catch",
		},
		{
			name:  "one-scalar-inside-one-item",
			left:  twoItems,
			right: oneScalarChanged,
			names: []string{"itm-0002"},
			why:   "an item whose projected text changed between runs, with everything else equal",
		},
		{
			name:  "one-scalar-outside-the-items",
			left:  twoItems,
			right: topLevelScalarChanged,
			names: []string{"position"},
			why:   "a difference in the artefact's own header, which carries no item to name",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			diffs := compareArtefacts(bundleFile, []byte(c.left), []byte(c.right))
			if len(diffs) == 0 {
				t.Fatalf("the comparator reported no difference between two artefacts differing in "+
					"%s; %s", c.name, c.why)
			}
			report := reportDifferences(diffs)
			for _, name := range c.names {
				if !strings.Contains(report, name) {
					t.Errorf("the comparator's report names no %s, so a failure would give a reader "+
						"no way back to what differed:\n%s", name, report)
				}
			}
		})
	}

	t.Run("equal-structure-unequal-bytes-is-still-a-difference", func(t *testing.T) {
		spaced := strings.Replace(twoItems, `"items":[`, `"items": [`, 1)
		diffs := compareArtefacts(bundleFile, []byte(twoItems), []byte(spaced))
		if len(diffs) == 0 {
			t.Fatal("the comparator called two artefacts identical because they decode the same; " +
				"the relation ac-1 asserts is byte-equality, and a structural comparison tolerates " +
				"exactly the re-serialisation this eval exists to refuse")
		}
	})

	t.Run("nonVacuous-refuses-an-empty-assembly", func(t *testing.T) {
		if err := nonVacuous(assembled{Position: posWidening}); err == nil {
			t.Fatal("nonVacuous accepted an assembly with no items; two empty artefacts are " +
				"byte-identical, so ac-1 would pass over an assembler that assembled nothing")
		}
		// One item, but not the order corpus: the assembly is non-empty and still
		// cannot support the assertions, which is the failure a bare `len > 0`
		// floor misses.
		lost := assembled{
			Position:      posWidening,
			BundleRaw:     []byte(twoItems),
			Items:         []bundleItem{{ItemKey: "itm-0001"}},
			ManifestItems: []manifestItem{{ItemKey: "itm-0001", Path: "README.md"}},
		}
		if err := nonVacuous(lost); err == nil {
			t.Fatal("nonVacuous accepted an assembly that carries items but has lost the order " +
				"corpus; the order oracle would then be sorting a sequence with nothing " +
				"adversarial left in it")
		}
	})
}
