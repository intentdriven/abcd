package reading

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTestFilesCarryTheTestKind is itd-198 ac-5. The case rule is part of the
// criterion, not an implementation detail: the Go toolchain builds only a
// lowercase _test.go as a test, so a file matching the suffix in any other case
// stays source.
func TestTestFilesCarryTheTestKind(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, "widget.go", "package main\n\nfunc Widget() {}\n")
	writeFile(t, root, "widget_test.go", "package main\n\nfunc TestWidget() {}\n")
	writeFile(t, root, "Gadget_TEST.go", "package main\n\nfunc GadgetUpper() {}\n")
	gitCommitAll(t, root)

	res := assembleFixture(t, root, PositionWidening)

	want := map[string]Kind{
		"widget.go":      KindSource,
		"widget_test.go": KindTest,
		"Gadget_TEST.go": KindSource,
	}
	got := map[string]Kind{}
	for _, m := range res.Manifest.Items {
		if _, ok := want[m.Path]; ok {
			got[m.Path] = m.Kind
		}
	}
	for path, wantKind := range want {
		gotKind, ok := got[path]
		if !ok {
			t.Errorf("%s was not admitted at all; the split must not change admission", path)
			continue
		}
		if gotKind != wantKind {
			t.Errorf("%s carries kind %q, want %q", path, gotKind, wantKind)
		}
	}
}

// TestKindSplitDoesNotMoveAdmission is itd-198 ac-4, proved by MUTATION rather
// than by a passing assertion: the test row is removed from the table and the
// admitted path set must be unchanged, because the row labels and never admits.
// A test that only asserted the current path set would pass just as happily if
// the row had narrowed admission at one position.
func TestKindSplitDoesNotMoveAdmission(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, "widget.go", "package main\n\nfunc Widget() {}\n")
	writeFile(t, root, "widget_test.go", "package main\n\nfunc TestWidget() {}\n")
	gitCommitAll(t, root)

	// Restored via t.Cleanup, not by a trailing assignment: assembleFixture
	// calls t.Fatalf on error, which would skip a trailing restore and leave
	// the package-global Table missing the test row for every later test in
	// the run — a mutating test that corrupts the suite it belongs to.
	restore := Table
	t.Cleanup(func() { Table = restore })

	for _, p := range AssemblingPositions() {
		Table = restore
		withRow := itemPaths(assembleFixture(t, root, p).Manifest)

		without := make([]Row, 0, len(restore))
		for _, row := range restore {
			if row.Kind == KindTest {
				continue
			}
			without = append(without, row)
		}
		Table = without
		withoutRow := itemPaths(assembleFixture(t, root, p).Manifest)
		Table = restore

		if strings.Join(withRow, "\n") != strings.Join(withoutRow, "\n") {
			t.Errorf("position %s: removing the test row changed the admitted path set; "+
				"the split must relabel and never admit or deny", p)
		}
	}
}

// TestTestKindIsReachableAtEveryPosition guards the mutation above from going
// vacuous: if no test file were admitted anywhere, the path sets would match
// trivially and the criterion would be untested.
func TestTestKindIsReachableAtEveryPosition(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, "widget_test.go", "package main\n\nfunc TestWidget() {}\n")
	gitCommitAll(t, root)

	for _, p := range AssemblingPositions() {
		// Not at comparative, and by design rather than by omission: every
		// shipped-tree row withdraws from that position, whose whole object is
		// the derived run's candidates and the criteria discipline
		// (adr-2609021016272867). A test kind reachable there would be the
		// widening the withdrawal exists to prevent.
		if p == PositionComparative {
			continue
		}
		res := assembleFixture(t, root, p)
		found := false
		for _, m := range res.Manifest.Items {
			if m.Kind == KindTest {
				found = true
			}
		}
		if !found {
			t.Errorf("position %s admitted no item of kind %q, so the admission "+
				"mutation proof above is vacuous there", p, KindTest)
		}
	}
}

// TestSizeReportSumsToTotal is itd-198 ac-1. Items and bytes sum exactly; the
// token total is derived from the total bytes rather than from summing the
// per-kind estimates, because integer division per kind would drift from the
// figure a reader compares against a capacity.
func TestSizeReportSumsToTotal(t *testing.T) {
	root := fixtureRepo(t)
	res := assembleFixture(t, root, PositionWidening)
	rep := res.Size

	if rep.Items != res.ItemCount {
		t.Errorf("the report counts %d items, the assembly passed %d", rep.Items, res.ItemCount)
	}
	var items, bytes int
	for _, k := range rep.ByKind {
		items += k.Items
		bytes += k.Bytes
	}
	if items != rep.Items {
		t.Errorf("per-kind items sum to %d, total says %d", items, rep.Items)
	}
	if bytes != rep.Bytes {
		t.Errorf("per-kind bytes sum to %d, total says %d", bytes, rep.Bytes)
	}
	if want := estimateTokens(rep.Bytes); rep.TokensEst != want {
		t.Errorf("total token estimate is %d, want %d from %d bytes", rep.TokensEst, want, rep.Bytes)
	}

	// The bytes reported are the bytes that actually travel.
	var bundleBytes int
	for _, it := range res.Bundle.Items {
		bundleBytes += len(it.Text)
	}
	if bundleBytes != rep.Bytes {
		t.Errorf("the report says %d bytes, the bundle carries %d", rep.Bytes, bundleBytes)
	}
}

// TestSizeReportOmitsKindsThatPassedNothing holds the distinction between an
// absent kind and an empty one. The vocabulary is closed and most of it is
// reachable, so a report listing every kind at zero would say less, not more.
func TestSizeReportOmitsKindsThatPassedNothing(t *testing.T) {
	root := fixtureRepo(t)
	res := assembleFixture(t, root, PositionWidening)

	present := map[Kind]bool{}
	for _, m := range res.Manifest.Items {
		present[m.Kind] = true
	}
	for _, k := range res.Size.ByKind {
		if !present[k.Kind] {
			t.Errorf("the report carries kind %q, which passed no item", k.Kind)
		}
		if k.Items == 0 {
			t.Errorf("the report carries kind %q with zero items", k.Kind)
		}
	}
	for kind := range present {
		found := false
		for _, k := range res.Size.ByKind {
			if k.Kind == kind {
				found = true
			}
		}
		if !found {
			t.Errorf("kind %q passed items and is missing from the report", kind)
		}
	}
}

// TestSizeReportRowsFollowTheVocabularyOrder keeps the rendering stable across
// runs: a map iteration would reorder the report for no reason a reader could
// see, and a diff of two runs would show a change that did not happen.
func TestSizeReportRowsFollowTheVocabularyOrder(t *testing.T) {
	root := fixtureRepo(t)
	rep := assembleFixture(t, root, PositionWidening).Size

	order := map[Kind]int{}
	for i, k := range Kinds() {
		order[k] = i
	}
	for i := 1; i < len(rep.ByKind); i++ {
		prev, cur := rep.ByKind[i-1].Kind, rep.ByKind[i].Kind
		if order[prev] >= order[cur] {
			t.Errorf("report row %d (%s) follows %s, which is not the vocabulary's order", i, cur, prev)
		}
	}
}

// TestDryRunCarriesTheSizeReport is itd-198 ac-2's core half: the report exists
// precisely so a size can be learned WITHOUT producing the artefact whose size
// is in question.
func TestDryRunCarriesTheSizeReport(t *testing.T) {
	root := fixtureRepo(t)
	res, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionDetection, Target: "HEAD", DryRun: true,
	})
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	if res.Written {
		t.Fatal("a dry run wrote an artefact")
	}
	if res.Size.Bytes == 0 || res.Size.Items == 0 {
		t.Error("a dry run produced an empty size report")
	}
	if res.Size.TokensEst == 0 {
		t.Error("a dry run produced no token estimate")
	}
}

// TestSizeReportLabelsItsEstimate is itd-198 ac-3. The label travels in the
// artefact rather than only in a rendering, so a report read out of context
// still says it is an estimate rather than looking like a count.
func TestSizeReportLabelsItsEstimate(t *testing.T) {
	root := fixtureRepo(t)
	basis := assembleFixture(t, root, PositionWidening).Size.Basis
	if basis == "" {
		t.Fatal("the report states no basis")
	}
	for _, want := range []string{"estimated", "bytes /"} {
		if !strings.Contains(basis, want) {
			t.Errorf("the basis %q does not contain %q", basis, want)
		}
	}
	if strings.Contains(strings.ToLower(basis), "tokenizer's count") &&
		!strings.Contains(strings.ToLower(basis), "not a tokenizer's count") {
		t.Errorf("the basis %q reads as a tokenizer's count", basis)
	}
}

// TestBundleGainsNoFieldFromTheReport is itd-198 ac-8. The bundle's shape is
// the reading's whole working set, so a field added there is a field a reading
// must be told how to read.
func TestBundleGainsNoFieldFromTheReport(t *testing.T) {
	root := fixtureRepo(t)
	res := assembleFixture(t, root, PositionWidening)
	raw, err := EncodeBundle(res.Bundle)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	// Decoded and inspected by SHAPE, not scanned as a substring. A bundle
	// carries verbatim Go source, so a scan for "bytes" matches any fixture
	// file that imports the bytes package — the assertion would have been
	// about the fixture's imports rather than about the bundle's shape.
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		t.Fatalf("decode bundle: %v", err)
	}
	// "preset" is expected: itd-199 puts what a run was GIVEN into the bundle
	// deliberately, because a reader told its object is the shipped tree and
	// handed a tenth of it reports the missing nine tenths as a finding. What
	// must stay out is the size report, which is the operator's fact and not
	// the reading's.
	want := map[string]bool{"_type": true, "schema_version": true, "position": true,
		"preset": true, "items": true}
	for key := range top {
		if !want[key] {
			t.Errorf("the bundle carries the top-level key %q; the report rides on the result alone", key)
		}
	}

	var items []map[string]json.RawMessage
	if err := json.Unmarshal(top["items"], &items); err != nil {
		t.Fatalf("decode items: %v", err)
	}
	// `candidate` and `field` join the bundle ITEM's allow-set with the
	// comparative channel, and they are the exception that proves the rule: a
	// comparative body cites a `candidate_id`, so the reading has to be told
	// which rdi-N each text belongs to and whether it is the configuration or
	// what admits it. Neither is a repository path — an item id is an ordinal the
	// ingest verb minted, and a field name is a record's own key — so brief
	// invariant 15 holds (adr-2609021016272867). They are omitempty and absent
	// from a widening bundle, which is the assembly this case decodes; the
	// comparative case below carries them.
	wantItem := map[string]bool{"item_key": true, "kind": true, "text": true,
		"candidate": true, "field": true}
	for i, item := range items {
		for key := range item {
			if !wantItem[key] {
				t.Errorf("bundle item %d carries the key %q", i, key)
			}
		}
		for _, absent := range []string{"candidate", "field"} {
			if _, present := item[absent]; present {
				t.Errorf("widening bundle item %d carries %q; both are a candidate item's alone "+
					"and are omitempty everywhere else", i, absent)
			}
		}
	}

	// The comparative bundle carries the two and nothing more, over the same
	// allow-set: a field added to that shape is caught here whichever position
	// introduced it.
	comparative, err := assembleComparative(t, root)
	if err != nil {
		t.Fatalf("assemble at comparative: %v", err)
	}
	cRaw, err := EncodeBundle(comparative.Bundle)
	if err != nil {
		t.Fatalf("encode the comparative bundle: %v", err)
	}
	var cTop map[string]json.RawMessage
	if err := json.Unmarshal(cRaw, &cTop); err != nil {
		t.Fatalf("decode the comparative bundle: %v", err)
	}
	for key := range cTop {
		if !want[key] {
			t.Errorf("the comparative bundle carries the top-level key %q", key)
		}
	}
	var cItems []map[string]json.RawMessage
	if err := json.Unmarshal(cTop["items"], &cItems); err != nil {
		t.Fatalf("decode the comparative items: %v", err)
	}
	carried := false
	for i, item := range cItems {
		for key := range item {
			if !wantItem[key] {
				t.Errorf("comparative bundle item %d carries the key %q", i, key)
			}
		}
		if _, present := item["candidate"]; present {
			carried = true
		}
	}
	if !carried {
		t.Error("no comparative bundle item carries `candidate`, so the allow-set above is not " +
			"being exercised by the position that needs it")
	}

	// The MANIFEST item's allow-set moves where the bundle item's does not.
	// itd-194 adds `scan` there and only there: the mark is the auditor's fact
	// about whether the examination reached this item, and a reading has no
	// more use for it than it has for the size report. Both sets are pinned in
	// one test so a field can never be added to the bundle by copying the
	// manifest's shape.
	mRaw, err := EncodeManifest(res.Manifest)
	if err != nil {
		t.Fatalf("encode manifest: %v", err)
	}
	var mTop map[string]json.RawMessage
	if err := json.Unmarshal(mRaw, &mTop); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	var mItems []map[string]json.RawMessage
	if err := json.Unmarshal(mTop["items"], &mItems); err != nil {
		t.Fatalf("decode manifest items: %v", err)
	}
	wantManifestItem := map[string]bool{"item_key": true, "path": true, "field": true,
		"candidate": true, "kind": true, "scan": true, "bytes": true, "sha256": true}
	sawScan := false
	for i, item := range mItems {
		for key := range item {
			if !wantManifestItem[key] {
				t.Errorf("manifest item %d carries the key %q", i, key)
			}
			if key == "scan" {
				sawScan = true
			}
		}
	}
	if !sawScan {
		t.Error("no manifest item carries `scan`; the mark is not omitempty, so an item " +
			"without one is a defect a reader must be able to see")
	}
}

// TestManifestItemRoundTripsKind is itd-198 ac-7.
func TestManifestItemRoundTripsKind(t *testing.T) {
	root := fixtureRepo(t)
	res := assembleFixture(t, root, PositionWidening)
	raw, err := EncodeManifest(res.Manifest)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	back, err := DecodeManifest(raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(back.Items) != len(res.Manifest.Items) {
		t.Fatalf("decoded %d items, encoded %d", len(back.Items), len(res.Manifest.Items))
	}
	for i, m := range back.Items {
		if m.Kind == "" {
			t.Errorf("%s decoded with no kind", m.ItemKey)
		}
		if m.Kind != res.Manifest.Items[i].Kind {
			t.Errorf("%s decoded kind %q, encoded %q", m.ItemKey, m.Kind, res.Manifest.Items[i].Kind)
		}
	}
}

// TestManifestItemKindIsNotOmitted holds the not-omitempty decision. An item
// with no kind must be visible in the bytes, because a shape that can omit the
// field cannot tell a defect from a well-formed item.
func TestManifestItemKindIsNotOmitted(t *testing.T) {
	m := Manifest{
		Type:          ManifestType,
		SchemaVersion: SchemaVersion,
		Items:         []ManifestItem{{ItemKey: "itm-0001", Path: "a.go", SHA256: "x"}},
	}
	raw, err := EncodeManifest(m)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if !strings.Contains(string(raw), "\"kind\"") {
		t.Error("an item with an empty kind encoded without the field; a missing kind " +
			"must be visible rather than indistinguishable from a well-formed item")
	}
}

// TestRenderCoversKindAndSuffix proves by MUTATION that the rendering the
// assembler version digests actually covers the two fields itd-198 adds to it.
// The rendering previously emitted neither, so a kind reassignment moved every
// bundle while the version stood still — the exact vacuity this criterion
// exists to close, which is why it is proved by breaking it.
func TestRenderCoversKindAndSuffix(t *testing.T) {
	restore := Table
	t.Cleanup(func() { Table = restore })

	base := Render()

	kindMutated := make([]Row, len(restore))
	copy(kindMutated, restore)
	kindMutated[0].Kind = KindConfig
	Table = kindMutated
	if Render() == base {
		t.Error("reassigning a row's kind did not move the rendering")
	}

	suffixMutated := make([]Row, len(restore))
	copy(suffixMutated, restore)
	suffixMutated[0].MatchSuffix = append([]string{"_extra.go"}, suffixMutated[0].MatchSuffix...)
	Table = suffixMutated
	if Render() == base {
		t.Error("adding a match suffix to a row did not move the rendering")
	}

	// And the Floor column itd-194 adds. The narrowing the table performs is
	// stated in the table, so it has to be in the text the version digests: a
	// row that stopped claiming the floor parses it would otherwise move what a
	// reading receives while the stamped version stood still.
	scanMutated := make([]Row, len(restore))
	copy(scanMutated, restore)
	scanMutated[0].Scan = ScanUnscanned
	Table = scanMutated
	if Render() == base {
		t.Error("reassigning a row's Scan did not move the rendering; the charter must state " +
			"per row whether the floor parses what it admits")
	}
}

// TestASuffixOnlyRowIsNotReadAsEveryFile is the trap the two match forms set
// for each other. An empty Match means "every file" on a row that selects by
// nothing else, and means "this form contributes nothing" on a row that selects
// by suffix. Reading the second as the first would admit the whole tree, and
// would render the contract as admitting it too.
func TestASuffixOnlyRowIsNotReadAsEveryFile(t *testing.T) {
	row := Row{
		Positions:   allPositions,
		Source:      ".",
		MatchSuffix: []string{"_test.go"},
		Kind:        KindTest,
		Rule:        "fixture",
	}
	if !row.matches("widget_test.go") {
		t.Error("a suffix-only row does not match the suffix it declares")
	}
	if row.matches("widget.go") {
		t.Error("a suffix-only row matched a file its suffix does not select; an empty " +
			"Match beside a non-empty MatchSuffix must not fall through to every-file")
	}
	if got := matchList(row); got == "every file" {
		t.Error("a suffix-only row renders its Matches column as \"every file\", stating " +
			"the opposite of what it admits in the text the version digests")
	}
}

// TestMatchListPinsEveryBranch pins the three cases of the Matches column
// independently of Row.matches. The two functions decide the same question in
// different files — what a row selects, and what the rendering SAYS it selects
// — and a mutation to one leaves the other's branch unproven, which is how a
// rendering comes to disagree with the table it renders. Each branch is pinned
// by its exact output rather than by "not every file", so a regression that
// merely reworded it is caught too.
func TestMatchListPinsEveryBranch(t *testing.T) {
	for name, tc := range map[string]struct {
		row  Row
		want string
	}{
		"neither form selects anything": {
			row: Row{Source: "."}, want: "every file",
		},
		"suffix only": {
			row:  Row{Source: ".", MatchSuffix: []string{"_test.go"}},
			want: "none",
		},
		"match only": {
			row: Row{Source: ".", Match: []string{".go"}}, want: "`.go`",
		},
		"both forms": {
			row:  Row{Source: ".", Match: []string{".go"}, MatchSuffix: []string{"_test.go"}},
			want: "`.go`",
		},
	} {
		if got := matchList(tc.row); got != tc.want {
			t.Errorf("%s: matchList = %q, want %q", name, got, tc.want)
		}
	}
}

// TestSuffixListPinsEveryBranch does the same for the Suffixes column.
func TestSuffixListPinsEveryBranch(t *testing.T) {
	if got := suffixList(nil); got != "none" {
		t.Errorf("suffixList(nil) = %q, want %q", got, "none")
	}
	if got := suffixList([]string{"_test.go"}); got != "`_test.go`" {
		t.Errorf("suffixList one = %q, want %q", got, "`_test.go`")
	}
	if got := suffixList([]string{"_test.go", "_gen.go"}); got != "`_test.go`, `_gen.go`" {
		t.Errorf("suffixList two = %q, want %q", got, "`_test.go`, `_gen.go`")
	}
}

// TestSizeBasisBringsNoParenthesesOfItsOwn pins the shape the rehearsal caught.
// The CLI wraps the basis in a parenthetical, so a basis carrying its own
// rendered as "((...))" on every run. Every gate was green and every fixture
// passed; only running the thing over a real tree showed it.
func TestSizeBasisBringsNoParenthesesOfItsOwn(t *testing.T) {
	if strings.ContainsAny(sizeBasis, "()") {
		t.Errorf("sizeBasis %q carries a parenthesis; the CLI supplies the pair", sizeBasis)
	}
}

// TestDecodeManifestRefusesAnItemWithoutAKind is the read side of the
// not-omitempty decision. The write side alone was not enough: the type's own
// comment claimed a missing kind is distinguishable from a well-formed item,
// and that was true of what this package WRITES and false of what it READS,
// which is an attestation asserting more than its examination establishes.
func TestDecodeManifestRefusesAnItemWithoutAKind(t *testing.T) {
	root := fixtureRepo(t)
	res := assembleFixture(t, root, PositionWidening)
	raw, err := EncodeManifest(res.Manifest)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	// Mutated through the decoded document rather than by substituting text.
	// A text substitution on "kind" was the obvious way to write this and it
	// silently stopped testing anything the moment the manifest gained a scope
	// block, because the scope's selectors carry "kind" too and the first
	// occurrence is no longer the item's. Addressing items[0] by structure
	// cannot drift that way.
	mutate := func(t *testing.T, f func(item map[string]any)) []byte {
		t.Helper()
		var doc map[string]any
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("decode for mutation: %v", err)
		}
		items, ok := doc["items"].([]any)
		if !ok || len(items) == 0 {
			t.Fatal("the manifest carries no items to mutate")
		}
		item, ok := items[0].(map[string]any)
		if !ok {
			t.Fatal("the first manifest item is not an object")
		}
		f(item)
		out, err := json.Marshal(doc)
		if err != nil {
			t.Fatalf("re-encode: %v", err)
		}
		return out
	}

	for name, f := range map[string]func(map[string]any){
		"empty kind":   func(item map[string]any) { item["kind"] = "" },
		"absent kind":  func(item map[string]any) { delete(item, "kind") },
		"unknown kind": func(item map[string]any) { item["kind"] = "warm-ledger" },
	} {
		bad := mutate(t, f)
		if _, err := DecodeManifest(bad); err == nil {
			t.Errorf("a manifest with an %s decoded", name)
		}
	}
}

// TestTheSizeReportIsCheckableAgainstTheManifest is the criterion the intent
// actually stated — "so the report is checkable against the manifest rather
// than asserted beside it" — held rather than approximated.
//
// The kind alone did not deliver it: an auditor could recompute per-kind item
// COUNTS from the manifest and not per-kind BYTES, which is the figure the
// intent exists to add. The manifest now carries each item's length, so this
// test rebuilds the whole report from the manifest and demands it match.
func TestTheSizeReportIsCheckableAgainstTheManifest(t *testing.T) {
	root := fixtureRepo(t)
	res := assembleFixture(t, root, PositionWidening)

	rebuilt := map[Kind]*KindSize{}
	var totalBytes, totalItems int
	for _, m := range res.Manifest.Items {
		k, ok := rebuilt[m.Kind]
		if !ok {
			k = &KindSize{Kind: m.Kind}
			rebuilt[m.Kind] = k
		}
		k.Items++
		k.Bytes += m.Bytes
		totalItems++
		totalBytes += m.Bytes
	}
	if totalItems != res.Size.Items || totalBytes != res.Size.Bytes {
		t.Errorf("the manifest rebuilds to %d items / %d bytes; the report says %d / %d",
			totalItems, totalBytes, res.Size.Items, res.Size.Bytes)
	}
	for _, row := range res.Size.ByKind {
		got, ok := rebuilt[row.Kind]
		if !ok {
			t.Errorf("the report names kind %q, which the manifest does not corroborate", row.Kind)
			continue
		}
		if got.Items != row.Items || got.Bytes != row.Bytes {
			t.Errorf("kind %q: manifest says %d items / %d bytes, report says %d / %d",
				row.Kind, got.Items, got.Bytes, row.Items, row.Bytes)
		}
		if want := estimateTokens(got.Bytes); want != row.TokensEst {
			t.Errorf("kind %q: the manifest's bytes estimate %d tokens, the report says %d",
				row.Kind, want, row.TokensEst)
		}
	}
	// And the byte length must be the length of the text that actually
	// travelled, not of the file on disk.
	for i, m := range res.Manifest.Items {
		if m.Bytes != len(res.Bundle.Items[i].Text) {
			t.Errorf("%s: the manifest says %d bytes, the bundle carries %d",
				m.ItemKey, m.Bytes, len(res.Bundle.Items[i].Text))
		}
	}
}

// TestRenderCannotForgeARowBoundary closes the collision channel fidelity review
// found in ac-9's absolute claim.
//
// Render() flattens the table into an unescaped pipe-delimited markdown table
// and nothing constrained the free-text fields it digests, so a Rule containing
// a newline and pipes could forge a row boundary and make two structurally
// different tables render — and therefore stamp — identically. That is the same
// author-controlled channel the spec cited to refuse a truncated digest, left
// open against rendering ambiguity while it was closed against truncation.
func TestRenderCannotForgeARowBoundary(t *testing.T) {
	for _, row := range Table {
		fields := map[string]string{"Source": row.Source, "Rule": row.Rule, "Store": row.Store,
			"Bucket": row.Bucket, "Kind": string(row.Kind)}
		for _, m := range row.Match {
			fields["Match "+m] = m
		}
		for _, m := range row.MatchSuffix {
			fields["MatchSuffix "+m] = m
		}
		for _, f := range row.Fields {
			fields["Field "+f] = f
		}
		for name, val := range fields {
			if strings.ContainsAny(val, "|\n\r") {
				t.Errorf("row %q field %s contains a pipe or newline (%q); the rendering the "+
					"assembler version digests is an unescaped table, so such a value can forge "+
					"a row boundary and make two different tables stamp alike",
					row.Source, name, val)
			}
		}
	}
}

// TestOptedInSourceAndTestsTravelWholeMarkedUnscanned is itd-194 ac-3. The
// maintainer's ruling of 2026-09-02 keeps `source` and `test` in the include
// table, because both design documents name code and tests as the shipped tree
// a reading may see, and pays for that with disclosure: an item from a row the
// floor does not parse arrives WHOLE — no redaction was attempted over it — and
// its manifest entry says the examination did not run (adr-56's refinement;
// brief invariant 16).
func TestOptedInSourceAndTestsTravelWholeMarkedUnscanned(t *testing.T) {
	for kind, probe := range map[Kind]string{KindTest: "main_test.go", KindSource: "main.go"} {
		root := fixtureRepo(t)
		writeFile(t, root, "main_test.go",
			"package main\n\n// the fixture's own test file is corpus, never built\n")
		// The entry names the two probe files as its object set: a tree row is
		// narrowed to the object set's paths, and an entry with no path hands
		// nothing from the tree whatever kinds it lists
		// (spc-2609020626048722).
		writeFile(t, root, ".abcd/config/reading-presets.json", fmt.Sprintf(`{
  "schema_version": 1,
  "presets": {
    "default": {"positions": {"detection":
      {"kinds": [%q], "records": [], "paths": ["main.go", "main_test.go"]}}}
  }
}`, string(kind)))
		gitCommitAll(t, root)

		res, err := Assemble(AssembleRequest{
			RepoRoot: root, Position: PositionDetection, Target: "HEAD", DryRun: true,
		})
		if err != nil {
			t.Fatalf("assemble under an entry naming %s: %v", kind, err)
		}
		found := false
		for i, m := range res.Manifest.Items {
			if m.Scan != ScanUnscanned {
				t.Errorf("the %s entry passed %s marked %q; the floor does not parse this row, "+
					"so every item it yields is disclosed as unexamined", kind, m.Path, m.Scan)
			}
			if m.Path != probe {
				continue
			}
			found = true
			raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(m.Path)))
			if err != nil {
				t.Fatalf("read %s: %v", m.Path, err)
			}
			if got := res.Bundle.Items[i].Text; got != string(raw) {
				t.Errorf("%s did not travel whole: the bundle carries %d bytes and the file "+
					"holds %d", m.Path, len(got), len(raw))
			}
		}
		if !found {
			t.Errorf("an entry naming %s passed no item at %s, so the mark was asserted over "+
				"nothing", kind, probe)
		}
	}
}

// TestNoParsedItemCarriesTheUnscannedMark is ac-3's other half and the
// falsifier for a mark stamped by habit rather than by the row: the mark is a
// fact about each item, so an item the floor DID parse must not carry it.
func TestNoParsedItemCarriesTheUnscannedMark(t *testing.T) {
	root := fixtureRepo(t)
	for _, p := range AssemblingPositions() {
		res := assembleFixture(t, root, p)
		markdown := 0
		for _, m := range res.Manifest.Items {
			if !strings.EqualFold(filepath.Ext(m.Path), ".md") {
				continue
			}
			markdown++
			if m.Scan != ScanParsed {
				t.Errorf("at %s the manifest marks the markdown item %s %q; the floor parses "+
					"markdown, so a mark saying otherwise understates the examination behind it",
					p, m.Path, m.Scan)
			}
		}
		if markdown == 0 {
			t.Errorf("the assembly at %s carries no markdown item, so this assertion ran over "+
				"nothing", p)
		}
	}
}

// TestSizeReportCountsUnscannedItems: the operator's own figure for how much of
// the assembly the exclusion floor never looked at. It rides on the result and
// on neither artefact, so it moves no version (spc-2609021003136831, "The size
// report counts what was not examined").
func TestSizeReportCountsUnscannedItems(t *testing.T) {
	root := fixtureRepo(t)
	res := assembleFixture(t, root, PositionDetection)
	want := 0
	for _, m := range res.Manifest.Items {
		if m.Scan == ScanUnscanned {
			want++
		}
	}
	if want == 0 {
		t.Fatal("the fixture assembly carries no unscanned item, so the count is asserted over nothing")
	}
	if res.Size.Unscanned != want {
		t.Errorf("the size report counts %d unscanned item(s) and the manifest marks %d",
			res.Size.Unscanned, want)
	}
	if res.Size.Unscanned >= res.Size.Items {
		t.Errorf("the report counts %d of %d items unscanned; a count that equals the whole "+
			"assembly is not the narrowing figure", res.Size.Unscanned, res.Size.Items)
	}
}

// TestDecodeManifestRefusesAnItemWithoutAScanMark is ac-4's read side. An item
// with no mark is a manifest that cannot say whether its key and heading
// exclusions were established for that item, which is exactly the artefact
// brief invariant 16 forbids — so it is refused rather than defaulted.
func TestDecodeManifestRefusesAnItemWithoutAScanMark(t *testing.T) {
	for name, f := range map[string]func(map[string]any){
		"empty scan":  func(item map[string]any) { item["scan"] = "" },
		"absent scan": func(item map[string]any) { delete(item, "scan") },
	} {
		bad := mutateFirstManifestItem(t, f)
		if _, err := DecodeManifest(bad); err == nil {
			t.Errorf("a manifest with an %s decoded", name)
		} else if !strings.Contains(err.Error(), "scan") {
			t.Errorf("the %s refusal does not name the key: %v", name, err)
		}
	}
}

// TestDecodeManifestRefusesAnUnknownScanMark holds the vocabulary closed on the
// read side, beside the kind check it sits next to.
func TestDecodeManifestRefusesAnUnknownScanMark(t *testing.T) {
	bad := mutateFirstManifestItem(t, func(item map[string]any) { item["scan"] = "skimmed" })
	if _, err := DecodeManifest(bad); err == nil {
		t.Error("a manifest naming an unknown scan mark decoded; the vocabulary is closed")
	} else if !strings.Contains(err.Error(), "skimmed") {
		t.Errorf("the refusal does not name the unknown mark: %v", err)
	}
}

// mutateFirstManifestItem encodes a real assembly's manifest, mutates its first
// item through the decoded document, and returns the bytes. Addressing the item
// by structure rather than by substituting text is deliberate: the manifest
// carries "kind" inside its preset selectors too, so a text substitution stops
// testing the item the moment another block spells the same key.
func mutateFirstManifestItem(t *testing.T, f func(item map[string]any)) []byte {
	t.Helper()
	root := fixtureRepo(t)
	res := assembleFixture(t, root, PositionWidening)
	raw, err := EncodeManifest(res.Manifest)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("decode for mutation: %v", err)
	}
	items, ok := doc["items"].([]any)
	if !ok || len(items) == 0 {
		t.Fatal("the manifest carries no items to mutate")
	}
	item, ok := items[0].(map[string]any)
	if !ok {
		t.Fatal("the first manifest item is not an object")
	}
	f(item)
	out, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("re-encode: %v", err)
	}
	return out
}
