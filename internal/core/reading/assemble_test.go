package reading

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestExcludedFieldsNeverReachTheBundle is itd-183's first criterion: given a
// repository state, the assembler's output contains no field on the exclusion
// list. Every warm class is planted with its own sentinel, so a leak names
// itself.
func TestExcludedFieldsNeverReachTheBundle(t *testing.T) {
	root := fixtureRepo(t)
	banned := map[string]string{
		sentinelEvidence:    "the brief's evidence chapter",
		sentinelDecision:    "a decision record",
		sentinelIssue:       "the issue ledger",
		sentinelAuditNotes:  "a shipped intent's Audit Notes",
		sentinelWhyItMatter: "a shipped intent's Why This Matters",
		sentinelOrigin:      "the origin frontmatter key",
		sentinelProdMode:    "the production_mode frontmatter key",
		sentinelSuperseded:  "a superseded intent",
		sentinelPlan:        "a plan record",
		sentinelPriorRun:    "the instrument's own prior manifest",
		sentinelDefinition:  "a reading definition and its eval",
		sentinelLapse:       "the lapse log",
	}
	for _, p := range AssemblingPositions() {
		res := assembleFixture(t, root, p)
		text := bundleText(res.Bundle)
		for token, what := range banned {
			if strings.Contains(text, token) {
				t.Errorf("position %s passed %s (%s)", p, what, token)
			}
		}
	}
}

// TestDraftBodyIsColdAtEntailmentAndWarmElsewhere is the drafts asymmetry
// measured on the assembled input rather than on the table alone.
func TestDraftBodyIsColdAtEntailmentAndWarmElsewhere(t *testing.T) {
	root := fixtureRepo(t)
	if text := bundleText(assembleFixture(t, root, PositionEntailment).Bundle); !strings.Contains(text, sentinelDraftBody) {
		t.Error("the entailment position did not pass the draft body; articulation precedes selection")
	}
	// Comparative is absent from this list because it no longer assembles
	// (itd-199). The asymmetry it used to demonstrate here is still held at the
	// table level for all four positions by
	// TestWideningExcludesDraftsAndPlannedEntailmentIncludesThem, which tests
	// Admits rather than an assembly, so nothing is lost by narrowing this one.
	for _, p := range []Position{PositionWidening, PositionDetection} {
		if text := bundleText(assembleFixture(t, root, p).Bundle); strings.Contains(text, sentinelDraftBody) {
			t.Errorf("position %s passed the draft body; only entailment sees the candidate set", p)
		}
	}
}

// TestNewRecordFamilyIsAbsentWithoutTableChange is itd-183's second criterion:
// a record type added under .abcd/development/ after the table was written is
// absent, and the assembler is not edited to make it so.
func TestNewRecordFamilyIsAbsentWithoutTableChange(t *testing.T) {
	root := fixtureRepo(t)
	const invented = "SENTINEL-INVENTED-FAMILY"
	writeFile(t, root, ".abcd/development/inventions/inv-1-a-new-record.md",
		"---\nid: inv-1\n---\n\n# An invented record\n\n"+invented+"\n")
	gitCommitAll(t, root)

	for _, p := range AssemblingPositions() {
		res := assembleFixture(t, root, p)
		if strings.Contains(bundleText(res.Bundle), invented) {
			t.Errorf("position %s passed a record family invented after the table was written", p)
		}
		for _, path := range itemPaths(res.Manifest) {
			if strings.HasPrefix(path, ".abcd/development/inventions/") {
				t.Errorf("position %s named %s in the manifest", p, path)
			}
		}
	}
}

// TestShippedIntentProjectsFiveFieldsOnly holds field granularity: a shipped
// intent travels as its claim record and nothing else.
//
// At detection rather than widening: itd-194 withdraws the shipped row from the
// widening position, whose object neither design document states with the
// shipped intents in it, so widening is now the one position where this
// projection has nothing to project.
func TestShippedIntentProjectsFiveFieldsOnly(t *testing.T) {
	root := fixtureRepo(t)
	res := assembleFixture(t, root, PositionDetection)
	const rel = ".abcd/development/intents/shipped/itd-1-a-shipped-intent.md"

	var got []string
	for _, it := range res.Manifest.Items {
		if it.Path == rel {
			got = append(got, it.Field)
		}
	}
	want := []string{"Press Release", "Acceptance Criteria", "Scope Conditions", "Mechanism", "spec_id"}
	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("shipped intent projected %v, want %v", got, want)
	}
}

// TestProjectedFieldsFollowTheDeclaredOrderWithinAPath holds the ordering
// degree of freedom the path-level order assertion leaves open
// (iss-2608311418202794).
//
// Five items can share one path when a record is projected field by field, so
// the order of those fields INSIDE a path decides the assembled bundle's bytes
// as surely as the path order does. Nothing held it. The determinism eval's
// order oracle compares paths only (framework 8.7), and its byte comparison
// sees two runs agree — a map-iterated field order is caught there because it
// differs between processes, but a deterministic reversal is not: both runs
// agree, in the wrong order. Framework 8.3 asks the manifest to map an item by
// path AND field, and this is the field half of that made checkable.
//
// The expectation is derived from the owning row's declared Fields, never from
// what the assembler emitted, so a reversal fails rather than confirming
// itself. A field the file does not carry contributes no item, so the emitted
// sequence is a SUBSEQUENCE of the declaration rather than equal to it.
func TestProjectedFieldsFollowTheDeclaredOrderWithinAPath(t *testing.T) {
	root := fixtureRepo(t)
	projected := 0
	for _, position := range AssemblingPositions() {
		res := assembleFixture(t, root, position)

		byPath := map[string][]string{}
		var order []string
		for _, it := range res.Manifest.Items {
			if it.Field == "" {
				continue
			}
			if _, seen := byPath[it.Path]; !seen {
				order = append(order, it.Path)
			}
			byPath[it.Path] = append(byPath[it.Path], it.Field)
		}

		for _, rel := range order {
			got := byPath[rel]
			if len(got) < 2 {
				continue
			}
			projected++
			row, ok := owningRow(position, rel)
			if !ok {
				t.Errorf("the manifest at %s projects fields out of %s, which no admitted row reaches",
					position, rel)
				continue
			}
			// The declaration, filtered to the fields that actually travelled: a
			// subsequence check stated as an equality, so the failure message can
			// show both sequences whole.
			carried := map[string]bool{}
			for _, f := range got {
				carried[f] = true
			}
			var want []string
			for _, f := range row.Fields {
				if carried[f] {
					want = append(want, f)
				}
			}
			if strings.Join(got, "|") == strings.Join(want, "|") {
				continue
			}
			t.Errorf("at %s the projection of %s emits its fields as %v, and the row that owns "+
				"it declares %v.\n\nThe field order inside a path is part of the assembled "+
				"bundle's bytes, and a stable but wrong one is invisible to a byte comparison "+
				"across two runs — both runs agree.", position, rel, got, want)
		}
	}
	if projected == 0 {
		t.Fatal("no assembled path carried more than one projected field, so this assertion " +
			"held nothing; the fixture's shipped intent is what puts five fields under one path")
	}
}

// owningRow returns the row that owns a path's projection at a position: the
// FIRST row of the table that reaches it, which is the same tie-break collect
// applies.
func owningRow(p Position, rel string) (Row, bool) {
	for _, row := range Table {
		if row.AdmittedAt(p) && row.Reaches(rel) {
			return row, true
		}
	}
	return Row{}, false
}

// TestBundleCarriesNoRepositoryPath is itd-183's fifth criterion. Item text is
// necessarily prose and source that may quote paths of its own, so the claim is
// made where it is a claim: the bundle's own structure carries no location.
func TestBundleCarriesNoRepositoryPath(t *testing.T) {
	root := fixtureRepo(t)
	res := assembleFixture(t, root, PositionWidening)

	skeleton := res.Bundle
	skeleton.Items = nil
	for _, it := range res.Bundle.Items {
		it.Text = ""
		skeleton.Items = append(skeleton.Items, it)
	}
	raw, err := EncodeBundle(skeleton)
	if err != nil {
		t.Fatalf("encode bundle skeleton: %v", err)
	}
	serialised := string(raw)
	for _, fragment := range []string{"/", "\\", ".abcd", "internal", "docs", root} {
		if strings.Contains(serialised, fragment) {
			t.Errorf("the bundle's structure carries %q; only the manifest maps an item to a path", fragment)
		}
	}
	for _, it := range res.Bundle.Items {
		if !itemKeyRe.MatchString(it.ItemKey) {
			t.Errorf("item key %q is not an ordinal of the form itm-NNNN", it.ItemKey)
		}
	}

	// The same claim over the COMPARATIVE bundle, whose items carry two more
	// structural fields. Neither may be a location: a candidate id is an ordinal
	// the ingest verb minted, and a field name is a record's own key, so the
	// bundle stays pathless with the candidate channel in it
	// (brief invariant 15; adr-2609021016272867).
	comparative, err := assembleComparative(t, root)
	if err != nil {
		t.Fatalf("assemble at comparative: %v", err)
	}
	cSkeleton := comparative.Bundle
	cSkeleton.Items = nil
	candidates := 0
	for _, it := range comparative.Bundle.Items {
		if it.Kind == KindCandidate {
			candidates++
		}
		it.Text = ""
		cSkeleton.Items = append(cSkeleton.Items, it)
	}
	if candidates == 0 {
		t.Fatal("the comparative bundle carries no candidate item, so this half asserts nothing " +
			"about the two structural fields the channel adds")
	}
	cRaw, err := EncodeBundle(cSkeleton)
	if err != nil {
		t.Fatalf("encode the comparative bundle skeleton: %v", err)
	}
	for _, fragment := range []string{"/", "\\", ".abcd", "internal", "docs", root} {
		if strings.Contains(string(cRaw), fragment) {
			t.Errorf("the comparative bundle's structure carries %q; `candidate` is an item id and "+
				"`field` is a record key, and neither is a location", fragment)
		}
	}
	// And neither is a SCOPE SELECTOR: nothing a reading is told can name a
	// record, a path or a material kind, which are the three clauses a committed
	// entry is made of.
	for _, it := range comparative.Bundle.Items {
		if it.Candidate == "" {
			continue
		}
		if recordIDRe.MatchString(it.Candidate) {
			t.Errorf("the bundle's candidate id %q is shaped like a record id an entry could name",
				it.Candidate)
		}
		for _, k := range Kinds() {
			if it.Field == string(k) {
				t.Errorf("the bundle's field %q collides with the material kind of the same name; "+
					"one token may not mean two things", it.Field)
			}
		}
	}
}

// TestWalkIsLexicographicAndByteStable is the determinism precondition itd-187's
// eval checks independently: one state assembled twice is one bundle.
func TestWalkIsLexicographicAndByteStable(t *testing.T) {
	root := fixtureRepo(t)
	first := assembleFixture(t, root, PositionEntailment)
	second := assembleFixture(t, root, PositionEntailment)

	a, err := EncodeBundle(first.Bundle)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	b, err := EncodeBundle(second.Bundle)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if string(a) != string(b) {
		t.Error("two assemblies of one state produced different bundles")
	}

	paths := itemPaths(first.Manifest)
	sorted := append([]string{}, paths...)
	sort.Strings(sorted)
	if strings.Join(paths, "\n") != strings.Join(sorted, "\n") {
		t.Error("the walk is not lexicographic over repo-relative paths")
	}
}

// TestAssembleRefusesDirtyIncludedPath holds the re-runnability the manifest
// promises: a dirty tree cannot be described by a commit reference.
func TestAssembleRefusesDirtyIncludedPath(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, "README.md", "# The repository\n\nEdited, uncommitted.\n")

	_, err := Assemble(AssembleRequest{RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true})
	if err == nil {
		t.Fatal("assembly over a dirty included path succeeded; the manifest would promise a re-run it cannot deliver")
	}
	if !strings.Contains(err.Error(), "README.md") {
		t.Errorf("the refusal does not name the dirty path: %v", err)
	}
}

// TestAssembleIgnoresDirtinessOutsideTheIncludeTable keeps the refusal
// proportionate: an edit the reading can never see does not block a run.
func TestAssembleIgnoresDirtinessOutsideTheIncludeTable(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, ".abcd/work/DECISIONS.md", "# Decisions\n\nedited, uncommitted.\n")

	if _, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true,
	}); err != nil {
		t.Fatalf("a dirty excluded path blocked the assembly: %v", err)
	}
}

// TestTargetRefusesBranchAndTag holds the second operand's closed shape.
func TestTargetRefusesBranchAndTag(t *testing.T) {
	root := fixtureRepo(t)
	for _, bad := range []string{"main", "v0.6.8", "HEAD~1", "origin/main", "", "abc", strings.Repeat("a", 41)} {
		_, err := Assemble(AssembleRequest{RepoRoot: root, Position: PositionWidening, Target: bad, DryRun: true})
		if err == nil {
			t.Errorf("target %q was accepted; only HEAD or a 7-to-40-digit hex sha may be named", bad)
		}
	}
	head := headOf(t, root)
	for _, good := range []string{"HEAD", head, head[:7], head[:12]} {
		if _, err := Assemble(AssembleRequest{
			RepoRoot: root, Position: PositionWidening, Target: good, DryRun: true,
		}); err != nil {
			t.Errorf("target %q was refused: %v", good, err)
		}
	}
}

// TestTargetMustResolveToHead holds the other half: a well-shaped sha that is
// not the tree in front of the assembler is refused, not silently assembled.
func TestTargetMustResolveToHead(t *testing.T) {
	root := fixtureRepo(t)
	_, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionWidening, Target: "0123456789abcdef", DryRun: true,
	})
	if err == nil {
		t.Fatal("a target that is not HEAD was accepted; assembly reads the working tree")
	}
}

// TestAssembleRefusesAnUnknownPosition holds the first operand at the core.
func TestAssembleRefusesAnUnknownPosition(t *testing.T) {
	root := fixtureRepo(t)
	if _, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: Position("framing"), Target: "HEAD", DryRun: true,
	}); err == nil {
		t.Fatal("an unknown position was accepted")
	}
}

// TestAssembleDryRunWritesNothing holds the render-without-writing idiom: with
// no output directory named, a dry run leaves the tree exactly as it found it.
func TestAssembleDryRunWritesNothing(t *testing.T) {
	root := fixtureRepo(t)
	before := treeSnapshot(t, root)

	res, err := Assemble(AssembleRequest{RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true})
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	if res.Written {
		t.Error("a dry run without an output directory reported a write")
	}
	if after := treeSnapshot(t, root); after != before {
		t.Error("a dry run without an output directory changed the tree")
	}
}

// TestAssembleWritesTwoSeparateArtefacts holds the interface the evals bind to:
// the assembled input and the manifest are two files in a directory the
// operator names.
func TestAssembleWritesTwoSeparateArtefacts(t *testing.T) {
	root := fixtureRepo(t)
	before := treeSnapshot(t, root)
	out := filepath.Join(t.TempDir(), "run")

	res, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionWidening, Target: "HEAD", OutDir: out, DryRun: true,
	})
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	if !res.Written {
		t.Fatal("an assembly into a named output directory reported no write")
	}
	bundleRaw, err := os.ReadFile(filepath.Join(out, BundleFileName))
	if err != nil {
		t.Fatalf("read the assembled input: %v", err)
	}
	manifestRaw, err := os.ReadFile(filepath.Join(out, ManifestFileName))
	if err != nil {
		t.Fatalf("read the manifest: %v", err)
	}
	if strings.Contains(string(bundleRaw), "\"path\"") {
		t.Error("the assembled input carries a path field")
	}
	m, err := DecodeManifest(manifestRaw)
	if err != nil {
		t.Fatalf("the written manifest does not decode strictly: %v", err)
	}
	if m.RunID != res.RunID {
		t.Errorf("the written manifest names run %s, the result names %s", m.RunID, res.RunID)
	}
	if treeSnapshot(t, root) != before {
		t.Error("writing into a named output directory changed the repository")
	}
}

// TestAssembleDefaultsToTheLocalTier holds the default artefact home.
func TestAssembleDefaultsToTheLocalTier(t *testing.T) {
	root := fixtureRepo(t)
	res, err := Assemble(AssembleRequest{RepoRoot: root, Position: PositionWidening, Target: "HEAD"})
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	want := DefaultRunDir + "/" + res.RunID
	if res.OutDir != want {
		t.Errorf("out dir is %q, want %q", res.OutDir, want)
	}
	for _, name := range []string{BundleFileName, ManifestFileName} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(want), name)); err != nil {
			t.Errorf("stat %s: %v", name, err)
		}
	}
}

// treeSnapshot lists every tracked and untracked path with its size, so a test
// can assert an assembly wrote nothing.
func treeSnapshot(t *testing.T, root string) string {
	t.Helper()
	var lines []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		rel, rerr := filepath.Rel(root, path)
		if rerr != nil {
			return rerr
		}
		// .git is skipped WHOLESALE, and before the error is examined. The
		// snapshot is about the worktree, and git's background maintenance
		// creates and removes lock files under .git/objects while the walk
		// runs — so a path enumerated a moment earlier is gone by the lstat,
		// and the old order surfaced that as a walk error before reaching the
		// skip. Returning SkipDir means the walk never descends there at all,
		// which removes the race rather than tolerating it
		// (iss-2609010818066790).
		if rel == ".git" {
			return filepath.SkipDir
		}
		if err != nil {
			// Anything else that vanished mid-walk is equally not a snapshot
			// concern: it cannot be an artefact a dry run wrote, because a dry
			// run writes nothing.
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if info.IsDir() {
			return nil
		}
		lines = append(lines, rel+"\x00"+itoa(info.Size()))
		return nil
	})
	if err != nil {
		t.Fatalf("walk fixture: %v", err)
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// TestSourceFileWithAnUnterminatedFenceDoesNotAbortTheAssembly holds the scope
// of the two record-shaped signals. A heading and a frontmatter key are things a
// RECORD carries; a Go file carries neither, and parsing one as markdown makes a
// raw string literal holding a fence able to stop every assembly the repository
// can run.
func TestSourceFileWithAnUnterminatedFenceDoesNotAbortTheAssembly(t *testing.T) {
	root := fixtureRepo(t)
	// A raw string literal whose content opens a fence at the left margin. The
	// site scan reads it as an unterminated fenced block and refuses.
	writeFile(t, root, "fence.go", "package main\n\nconst usage = `\n```sh\nabcd reading\n`\n")
	gitCommitAll(t, root)

	res, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true,
	})
	if err != nil {
		t.Fatalf("a fence inside a Go raw string aborted the assembly: %v", err)
	}
	found := false
	for _, m := range res.Manifest.Items {
		if m.Path == "fence.go" {
			found = true
		}
	}
	if !found {
		t.Error("the source file did not reach the bundle")
	}
}

// TestAssembleRefusesADeletedIncludedPath closes the hole a
// collected-paths-only dirty check leaves. A file deleted in the working tree
// but present at the target commit is absent from the assembly and absent from
// the status intersection, so the manifest would name a commit whose content it
// did not read — the exact promise the dirty gate exists to keep.
func TestAssembleRefusesADeletedIncludedPath(t *testing.T) {
	root := fixtureRepo(t)
	if err := os.Remove(filepath.Join(root, "README.md")); err != nil {
		t.Fatalf("remove: %v", err)
	}
	_, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true,
	})
	if err == nil {
		t.Fatal("an included path deleted in the working tree did not refuse the assembly")
	}
	if !strings.Contains(err.Error(), "README.md") {
		t.Errorf("the refusal does not name the deleted path: %v", err)
	}
}

// TestIgnoredFilesNeverEnterTheAssembly closes the gap between the filesystem
// and the commit. A gitignored file is not part of the target commit and `git
// status` says nothing about it, so a filesystem walk would pass content the
// dirty gate cannot see and the manifest would name a commit whose content it
// did not read. Build output, a virtual environment and a vendored tree all
// land in this class.
func TestIgnoredFilesNeverEnterTheAssembly(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, ".gitignore", "build/\nignored.json\n")
	gitCommitAll(t, root)

	const canary = "SENTINEL-IGNORED-BUILD-OUTPUT"
	writeFile(t, root, "ignored.json", "{\"note\": \""+canary+"\"}\n")
	writeFile(t, root, "build/generated.go", "package build\n\n// "+canary+"\n")

	res, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true,
	})
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	if strings.Contains(bundleText(res.Bundle), canary) {
		t.Error("an ignored file reached the bundle; it is not part of the commit the manifest names")
	}
	for _, m := range res.Manifest.Items {
		if m.Path == "ignored.json" || strings.HasPrefix(m.Path, "build/") {
			t.Errorf("the manifest names the ignored path %s", m.Path)
		}
	}
}

// TestUntrackedIncludedFileRefusesTheAssembly is the other side of the same
// boundary: a file that is NOT ignored and not yet committed is a genuine
// divergence from the target commit, so it refuses rather than being quietly
// dropped.
func TestUntrackedIncludedFileRefusesTheAssembly(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, "docs/newcomer.md", "# A newcomer\n\nUncommitted.\n")

	_, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true,
	})
	if err == nil {
		t.Fatal("an untracked included file did not refuse the assembly")
	}
	if !strings.Contains(err.Error(), "docs/newcomer.md") {
		t.Errorf("the refusal does not name the untracked path: %v", err)
	}
}

// TestAssembleRefusesAnUnconfiguredRecordScan holds the fail-closed stance at
// the one place a silent success is worse than a refusal. Record enumeration
// runs through the record-lint configuration; without it every record row
// contributes nothing, and the run reports a clean assembly of a reading that
// saw none of the record it exists to read against.
func TestAssembleRefusesAnUnconfiguredRecordScan(t *testing.T) {
	root := fixtureRepo(t)
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(LintConfigPath))); err != nil {
		t.Fatalf("remove: %v", err)
	}
	gitCommitAll(t, root)

	_, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true,
	})
	if err == nil {
		t.Fatal("an unconfigured record scan assembled silently")
	}
	if !strings.Contains(err.Error(), LintConfigPath) {
		t.Errorf("the refusal does not name the missing configuration: %v", err)
	}
}

// TestAssembleRefusesAStoreTheConfigurationDoesNotName is the same refusal one
// level in: a configuration present but silent about a store the include table
// names is the same silent nothing, arriving by a different route.
func TestAssembleRefusesAStoreTheConfigurationDoesNotName(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, LintConfigPath, `{"schema_version": 1, "rules": {"record_schema":
  {"enabled": true, "severity": "blocker", "record_stores": {"itd": ".abcd/development/intents"}}}}`)
	gitCommitAll(t, root)

	_, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true,
	})
	if err == nil {
		t.Fatal("an include row naming an unconfigured store assembled silently")
	}
	if !strings.Contains(err.Error(), "spc") {
		t.Errorf("the refusal does not name the unconfigured store: %v", err)
	}
}

// The three shapes that slip past a first-occurrence, exact-match redaction.
// Each is a warm field the manifest asserts was refused, still in the bundle.

// TestDuplicateExcludedKeyRefusesTheFile: Fields keeps the first occurrence of a
// key and drops the rest silently, so redacting what it reports leaves the
// second copy in place.
func TestDuplicateExcludedKeyRefusesTheFile(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, ".abcd/development/specs/open/spc-2-duplicated.md",
		"---\nid: spc-2\norigin: cold\norigin: "+sentinelOrigin+"\n---\n\n# A spec\n\nProse.\n")
	gitCommitAll(t, root)

	_, err := Assemble(AssembleRequest{RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true})
	if err == nil {
		t.Fatal("a duplicated excluded key did not refuse the file; the second copy survives redaction")
	}
	if !strings.Contains(err.Error(), "origin") {
		t.Errorf("the refusal does not name the duplicated key: %v", err)
	}
}

// TestExcludedKeySurvivingRedactionRefusesTheFile: StripFrontmatter closes on a
// `---` PREFIX, so a four-dash close is cut from the body while Fields, which
// wants the delimiter exactly, reads no fields at all and redacts nothing.
func TestExcludedKeySurvivingRedactionRefusesTheFile(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, ".abcd/development/specs/open/spc-3-four-dashes.md",
		"---\nid: spc-3\norigin: "+sentinelOrigin+"\n----\n\n# A spec\n\nProse.\n")
	gitCommitAll(t, root)

	_, err := Assemble(AssembleRequest{RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true})
	if err == nil {
		t.Fatal("an excluded key survived redaction and was passed")
	}
	if !strings.Contains(err.Error(), "origin") {
		t.Errorf("the refusal does not name the surviving key: %v", err)
	}
}

// TestCaseVariantExcludedHeadingRefusesTheFile: the heading redaction matches a
// title exactly, so another spelling of the same heading travels whole.
func TestCaseVariantExcludedHeadingRefusesTheFile(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, ".abcd/development/specs/open/spc-4-case-variant.md",
		"---\nid: spc-4\n---\n\n# A spec\n\n## audit notes\n\n"+sentinelAuditNotes+"\n")
	gitCommitAll(t, root)

	_, err := Assemble(AssembleRequest{RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true})
	if err == nil {
		t.Fatal("a case-variant excluded heading was passed whole")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "audit notes") {
		t.Errorf("the refusal does not name the surviving heading: %v", err)
	}
}

// TestStagedRenameOutOfTheIncludeSetRefuses: a rename's SOURCE path is the one
// that was in the target commit. Discarding it leaves an included file that is
// neither in the bundle nor refused, and the manifest names HEAD for a bundle
// that is not HEAD.
func TestStagedRenameOutOfTheIncludeSetRefuses(t *testing.T) {
	root := fixtureRepo(t)
	gitRun(t, root, "mv", "docs/reference/thing.md", ".abcd/development/plans/thing.md")

	_, err := Assemble(AssembleRequest{RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true})
	if err == nil {
		t.Fatal("an included file renamed out of the include set did not refuse the assembly")
	}
	if !strings.Contains(err.Error(), "docs/reference/thing.md") {
		t.Errorf("the refusal does not name the rename's source path: %v", err)
	}
}

// TestWorktreeRenameRefuses holds the same case in the second status column,
// which the source-consuming branch has to read as well as the first.
func TestWorktreeRenameRefuses(t *testing.T) {
	root := fixtureRepo(t)
	gitRun(t, root, "mv", "README.md", "READYOU.md")
	gitRun(t, root, "reset", "-q")

	_, err := Assemble(AssembleRequest{RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true})
	if err == nil {
		t.Fatal("a worktree rename of an included file did not refuse the assembly")
	}
	if !strings.Contains(err.Error(), "README.md") {
		t.Errorf("the refusal does not name the renamed path: %v", err)
	}
}

// TestRetargetedStoreDirectoryRefuses: a store key present but pointed at a
// directory that is not there enumerates nothing and reports a clean run.
func TestRetargetedStoreDirectoryRefuses(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, LintConfigPath, `{"schema_version": 1, "rules": {"record_schema":
  {"enabled": true, "severity": "blocker", "record_stores": {"itd": ".abcd/development/intents",
   "spc": ".abcd/development/specifications"}}}}`)
	gitCommitAll(t, root)

	_, err := Assemble(AssembleRequest{RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true})
	if err == nil {
		t.Fatal("a store pointed at an absent directory assembled silently")
	}
	if !strings.Contains(err.Error(), "specifications") {
		t.Errorf("the refusal does not name the absent store directory: %v", err)
	}
}

// TestUncommittedLintConfigRefuses: the record configuration decides what the
// record scan sees, and it sits under the deny, so no include row puts it in the
// dirty set. An uncommitted retarget would silently reshape the assembly.
func TestUncommittedLintConfigRefuses(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, LintConfigPath, `{"schema_version": 1, "rules": {"record_schema":
  {"enabled": true, "severity": "blocker", "record_stores": {"itd": ".abcd/development/intents",
   "spc": ".abcd/development/specs"}}}}`)

	_, err := Assemble(AssembleRequest{RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true})
	if err == nil {
		t.Fatal("an uncommitted record configuration did not refuse the assembly")
	}
	if !strings.Contains(err.Error(), LintConfigPath) {
		t.Errorf("the refusal does not name the configuration: %v", err)
	}
}

// TestUntrackedFileInANewDirectoryRefuses: git collapses an untracked directory
// to a single entry, so an admitted file inside it is never named.
func TestUntrackedFileInANewDirectoryRefuses(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, "docs/newdir/page.md", "# A page\n\nUncommitted, in a new directory.\n")

	_, err := Assemble(AssembleRequest{RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true})
	if err == nil {
		t.Fatal("an untracked admitted file in a new directory did not refuse the assembly")
	}
	if !strings.Contains(err.Error(), "docs/newdir/page.md") {
		t.Errorf("the refusal does not name the untracked file: %v", err)
	}
}

// TestProjectionKeepsASubsectionOfAProjectedField: the redactor ends a section
// at the next heading of level <= its own, and the projection must agree. Ending
// at the next heading of ANY level drops a `###` under a projected `##` — the
// item is silently short and the manifest hashes the short version.
func TestProjectionKeepsASubsectionOfAProjectedField(t *testing.T) {
	root := fixtureRepo(t)
	const nested = "SENTINEL-NESTED-CRITERION"
	writeFile(t, root, ".abcd/development/intents/shipped/itd-6-nested.md",
		"---\nid: itd-6\nspec_id: spc-1\n---\n\n# A shipped intent\n\n"+
			"## Acceptance Criteria\n\n- A top-level criterion.\n\n"+
			"### A grouped set\n\n- "+nested+"\n\n"+
			"## Audit Notes\n\n"+sentinelAuditNotes+"\n")
	gitCommitAll(t, root)

	// At detection: the shipped row withdrew from widening with itd-194.
	res := assembleFixture(t, root, PositionDetection)
	text := bundleText(res.Bundle)
	if !strings.Contains(text, nested) {
		t.Error("a subsection of a projected field was dropped; projection must end where the redactor ends")
	}
	if strings.Contains(text, sentinelAuditNotes) {
		t.Error("the projection ran past its own section into an excluded one")
	}
}

// TestAssembleRefusesANonEmptyOutputDirectory: the two artefacts are one run's
// evidence, and writing them beside another run's leaves a directory whose
// manifest describes only half of what is in it.
func TestAssembleRefusesANonEmptyOutputDirectory(t *testing.T) {
	root := fixtureRepo(t)
	out := filepath.Join(t.TempDir(), "run")
	writeFile(t, out, "leftover.txt", "from an earlier run\n")

	_, err := Assemble(AssembleRequest{RepoRoot: root, Position: PositionWidening, Target: "HEAD", OutDir: out})
	if err == nil {
		t.Fatal("a non-empty output directory was accepted")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("the refusal does not say what is wrong with the directory: %v", err)
	}

	empty := filepath.Join(t.TempDir(), "fresh")
	if _, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionWidening, Target: "HEAD", OutDir: empty,
	}); err != nil {
		t.Errorf("an absent output directory was refused: %v", err)
	}
	if _, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionWidening, Target: "HEAD", OutDir: empty,
	}); err == nil {
		t.Error("a second run over the same directory was accepted; it is no longer empty")
	}
}

// TestOwnArtefactRefusalIsContentSigned: keying the refusal on a .json extension
// and a successful top-level unmarshal makes it a spelling check, not a
// recognition. Each shape below is this assembler's own output, admitted whole,
// while the run's manifest asserts the exclusion.
func TestOwnArtefactRefusalIsContentSigned(t *testing.T) {
	priorManifest := `{
  "_type": "` + ManifestType + `",
  "schema_version": 1,
  "run_id": "rdg-2608301200000001",
  "items": [{"item_key": "itm-0001", "path": ".abcd/work/issues/open/iss-1-a-defect.md", "sha256": "00"}]
}`
	shapes := map[string]string{
		"a manifest committed as YAML": "runs/prior.yaml",
		"a manifest committed as TOML": "runs/prior.toml",
	}
	for what, rel := range shapes {
		root := fixtureRepo(t)
		writeFile(t, root, rel, priorManifest+"\n")
		gitCommitAll(t, root)
		assertRefusesOwnArtefact(t, root, what, rel)
	}

	byteShapes := map[string]string{
		"a JSON manifest behind a byte-order mark": "\ufeff" + priorManifest + "\n",
		"a manifest wrapped in another object":     `{"run": ` + priorManifest + "}\n",
		"a manifest wrapped in an array":           "[" + priorManifest + "]\n",
		"a manifest carrying a duplicate _type": `{"_type": "notes", "_type": "` + ManifestType + `",
  "schema_version": 1}` + "\n",
		// The one shape a byte scan cannot see, and the reason the parse stays on
		// as a second check rather than being replaced by it.
		"a bundle whose tag is unicode-escaped": `{"_type": "abcd.reading.b\u0075ndle",
  "schema_version": 1, "items": []}` + "\n",
	}
	for what, body := range byteShapes {
		root := fixtureRepo(t)
		writeFile(t, root, "runs/prior.json", body)
		gitCommitAll(t, root)
		assertRefusesOwnArtefact(t, root, what, "runs/prior.json")
	}
}

// assertRefusesOwnArtefact runs one assembly and requires it to refuse, naming
// the artefact.
func assertRefusesOwnArtefact(t *testing.T, root, what, rel string) {
	t.Helper()
	_, err := Assemble(AssembleRequest{RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true})
	if err == nil {
		t.Errorf("%s was admitted whole (%s)", what, rel)
		return
	}
	if !strings.Contains(err.Error(), rel) {
		t.Errorf("%s: the refusal does not name the artefact: %v", what, err)
	}
}

// The spellings the floor's heading and key halves did not recognise. Each is an
// excluded field travelling while the manifest asserts it refused.
func TestExcludedHeadingSpellingsAreRecognised(t *testing.T) {
	cases := map[string]string{
		"an ATX closing sequence":       "## Audit Notes ##\n\n" + sentinelAuditNotes + "\n",
		"a single closing hash":         "## Audit Notes #\n\n" + sentinelAuditNotes + "\n",
		"a closing sequence with space": "## Audit Notes ##   \n\n" + sentinelAuditNotes + "\n",
		"a setext underline":            "Audit Notes\n---\n\n" + sentinelAuditNotes + "\n",
		"a setext rule":                 "Audit Notes\n===\n\n" + sentinelAuditNotes + "\n",
		// One to three leading spaces is still an ATX heading to every CommonMark
		// renderer, and four makes it an indented code block instead.
		"a one-space indent":   " ## Audit Notes\n\n" + sentinelAuditNotes + "\n",
		"a three-space indent": "   ## Audit Notes\n\n" + sentinelAuditNotes + "\n",
	}
	for what, body := range cases {
		root := fixtureRepo(t)
		writeFile(t, root, ".abcd/development/specs/open/spc-9-spelled.md",
			"---\nid: spc-9\n---\n\n# A spec\n\nProse.\n\n"+body)
		gitCommitAll(t, root)

		res, err := Assemble(AssembleRequest{
			RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true,
		})
		if err != nil {
			continue // a refusal is a legitimate way to recognise the spelling
		}
		if strings.Contains(bundleText(res.Bundle), sentinelAuditNotes) {
			t.Errorf("%s let an excluded section travel", what)
		}
	}
}

// TestQuotedExcludedKeyIsRecognised: the field reader does not report a quoted
// key, so redaction leaves it and the manifest asserts a refusal that did not
// happen.
func TestQuotedExcludedKeyIsRecognised(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, ".abcd/development/specs/open/spc-7-quoted.md",
		"---\nid: spc-7\n\"origin\": "+sentinelOrigin+"\n---\n\n# A spec\n\nProse.\n")
	gitCommitAll(t, root)

	res, err := Assemble(AssembleRequest{RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true})
	if err == nil && strings.Contains(bundleText(res.Bundle), sentinelOrigin) {
		t.Error("a quoted excluded key travelled")
	}
}

// TestBlockScalarWithABlankLineIsFullyRedacted: the drop loop stopped at the
// first blank line, so the rest of a block scalar stayed in the frontmatter.
func TestBlockScalarWithABlankLineIsFullyRedacted(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, ".abcd/development/specs/open/spc-8-block.md",
		"---\nid: spc-8\norigin: |\n  a first line\n\n  "+sentinelOrigin+"\nspec: kept\n---\n\n# A spec\n\nProse.\n")
	gitCommitAll(t, root)

	res, err := Assemble(AssembleRequest{RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true})
	if err == nil && strings.Contains(bundleText(res.Bundle), sentinelOrigin) {
		t.Error("a block scalar's tail survived the blank line inside it")
	}
}

// TestFencedTemplateExampleDoesNotRefuse is the false-positive half, and it
// matters as much as the leaks: the redactor is fence-aware and the verifier was
// not, so a fenced example of the record template in ANY admitted document
// refused every assembly the repository could run.
func TestFencedTemplateExampleDoesNotRefuse(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, "docs/reference/template.md",
		"# The record template\n\nA record looks like this:\n\n"+
			"```markdown\n---\nid: itd-1\norigin: scribe\n---\n\n## Audit Notes\n\nThe audit goes here.\n```\n")
	gitCommitAll(t, root)

	if _, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true,
	}); err != nil {
		t.Fatalf("a fenced template example refused the assembly: %v", err)
	}
}

// TestSymlinkedOutDirIntoAnAdmittedRootRefuses: the self-admitting check
// compares path text, so a symlink whose name is outside the table's reach and
// whose target is inside it walks straight through.
func TestSymlinkedOutDirIntoAnAdmittedRootRefuses(t *testing.T) {
	root := fixtureRepo(t)
	target := filepath.Join(root, "runs")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "elsewhere")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionWidening, Target: "HEAD", OutDir: link,
	}); err == nil {
		t.Fatal("an output directory symlinked into an admitted root was accepted")
	}
}

// TestSymlinkedStoreDirectoryRefuses: os.Stat follows the link, so a store
// pointed through one passes the directory check and then enumerates nothing,
// because the record scan reads the real path.
func TestSymlinkedStoreDirectoryRefuses(t *testing.T) {
	root := fixtureRepo(t)
	target := filepath.Join(root, "elsewhere-specs")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, ".abcd/development/linked-specs")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	writeFile(t, root, LintConfigPath, `{"schema_version": 1, "rules": {"record_schema":
  {"enabled": true, "severity": "blocker", "record_stores": {"itd": ".abcd/development/intents",
   "spc": ".abcd/development/linked-specs"}}}}`)
	gitCommitAll(t, root)

	_, err := Assemble(AssembleRequest{RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true})
	if err == nil {
		t.Fatal("a store reached through a symlink was accepted")
	}
	if !strings.Contains(err.Error(), "linked-specs") {
		t.Errorf("the refusal does not name the symlinked store: %v", err)
	}
}

// TestAbsentWalkSourceRefuses: a walk row names a directory that must be there —
// a brief chapter, the glossary. Absent, it enumerates nothing and the run
// reports clean. A record store's BUCKET is different: an empty lifecycle bucket
// is legitimate, so those stay silent by design.
func TestAbsentWalkSourceRefuses(t *testing.T) {
	root := fixtureRepo(t)
	if err := os.RemoveAll(filepath.Join(root, ".abcd/development/brief/glossary")); err != nil {
		t.Fatal(err)
	}
	gitCommitAll(t, root)

	_, err := Assemble(AssembleRequest{RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true})
	if err == nil {
		t.Fatal("an absent include-row source directory assembled silently")
	}
	if !strings.Contains(err.Error(), "glossary") {
		t.Errorf("the refusal does not name the absent source: %v", err)
	}
}

// TestAnEmptyBucketStaysSilent is the other side of that line: removing a record
// store's bucket is a legitimate state and must not refuse.
func TestAnEmptyBucketStaysSilent(t *testing.T) {
	root := fixtureRepo(t)
	if err := os.RemoveAll(filepath.Join(root, ".abcd/development/intents/drafts")); err != nil {
		t.Fatal(err)
	}
	gitCommitAll(t, root)

	if _, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionEntailment, Target: "HEAD", DryRun: true,
	}); err != nil {
		t.Fatalf("an absent lifecycle bucket refused the assembly: %v", err)
	}
}

// TestAFailedManifestWriteLeavesNoBundle: the two artefacts are one run's
// evidence. A bundle left behind by a half-finished run is both a partial record
// and a directory that refuses every later run for being non-empty.
func TestAFailedManifestWriteLeavesNoBundle(t *testing.T) {
	dir := t.TempDir()
	// A directory where the manifest must go: the write cannot succeed.
	if err := os.MkdirAll(filepath.Join(dir, ManifestFileName), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writePair(dir, []byte("{}\n"), []byte("{}\n")); err == nil {
		t.Fatal("a manifest write onto a directory succeeded")
	}
	if _, err := os.Stat(filepath.Join(dir, BundleFileName)); !os.IsNotExist(err) {
		t.Error("the bundle survived a failed manifest write")
	}
}

// TestHeadingsThatRenderAsAnExcludedTitleAreRecognised: a heading that renders
// identically to an excluded one but differs in bytes is the same heading to
// every reader of the rendered page, and the byte comparison misses it.
func TestHeadingsThatRenderAsAnExcludedTitleAreRecognised(t *testing.T) {
	cases := map[string]string{
		"bold":        "## **Audit Notes**\n\n" + sentinelAuditNotes + "\n",
		"a code span": "## `Audit Notes`\n\n" + sentinelAuditNotes + "\n",
		"emphasis":    "## _Audit Notes_\n\n" + sentinelAuditNotes + "\n",
		// Every non-ASCII probe is written as a Go escape. Spelled as the literal
		// byte it is one keystroke from a plain space, and a shell or an editor
		// that flattens it turns the probe into a duplicate of the bare title —
		// a test that passes while testing nothing.
		"a non-breaking space": "## Audit\u00a0Notes\n\n" + sentinelAuditNotes + "\n",
		// The same spellings on the two paths the section scan cannot see, where
		// the raw-line refusal is the only thing between the section and the
		// bundle.
		"bold under a one-space indent":      " ## **Audit Notes**\n\n" + sentinelAuditNotes + "\n",
		"a non-breaking space when indented": " ## Audit\u00a0Notes\n\n" + sentinelAuditNotes + "\n",
		"bold over a setext rule":            "**Audit Notes**\n---\n\n" + sentinelAuditNotes + "\n",
		"a code span over a setext rule":     "`Audit Notes`\n===\n\n" + sentinelAuditNotes + "\n",
	}
	for what, body := range cases {
		root := fixtureRepo(t)
		writeFile(t, root, ".abcd/development/specs/open/spc-11-rendered.md",
			"---\nid: spc-11\n---\n\n# A spec\n\nProse.\n\n"+body)
		gitCommitAll(t, root)

		res, err := Assemble(AssembleRequest{
			RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true,
		})
		if err != nil {
			continue // a refusal is a legitimate way to recognise the spelling
		}
		if strings.Contains(bundleText(res.Bundle), sentinelAuditNotes) {
			t.Errorf("a heading spelled with %s let an excluded section travel", what)
		}
	}
}

// TestIndentedFrontmatterBlockIsRecognised: a block whose keys all carry one
// space of indent is valid YAML, and both the field reader and the column-0 key
// pattern look straight past it.
func TestIndentedFrontmatterBlockIsRecognised(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, ".abcd/development/specs/open/spc-12-indented.md",
		"---\n id: spc-12\n origin: "+sentinelOrigin+"\n---\n\n# A spec\n\nProse.\n")
	gitCommitAll(t, root)

	res, err := Assemble(AssembleRequest{RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true})
	if err == nil && strings.Contains(bundleText(res.Bundle), sentinelOrigin) {
		t.Error("an indented frontmatter block let an excluded key travel")
	}
}

// TestRawHTMLHeadingIsRecognised: the site's markdown subset admits h1-h6, so a
// raw HTML heading renders as a heading and the markdown scan never sees one.
func TestRawHTMLHeadingIsRecognised(t *testing.T) {
	cases := map[string]string{
		"an h2":             "<h2>Audit Notes</h2>\n\n" + sentinelAuditNotes + "\n",
		"an h1":             "<h1>Audit Notes</h1>\n\n" + sentinelAuditNotes + "\n",
		"an h3 with markup": "<h3><strong>Audit Notes</strong></h3>\n\n" + sentinelAuditNotes + "\n",
	}
	for what, body := range cases {
		root := fixtureRepo(t)
		writeFile(t, root, ".abcd/development/specs/open/spc-13-html.md",
			"---\nid: spc-13\n---\n\n# A spec\n\nProse.\n\n"+body)
		gitCommitAll(t, root)

		res, err := Assemble(AssembleRequest{
			RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true,
		})
		if err != nil {
			continue
		}
		if strings.Contains(bundleText(res.Bundle), sentinelAuditNotes) {
			t.Errorf("a raw HTML heading (%s) let an excluded section travel", what)
		}
	}
}

// TestRenderEquivalenceCoversWrappersAndEntities: the slug comparison models
// emphasis and code marks only, so a title carrying an HTML comment, a span
// wrapper, a link wrapper or an entity slugs differently while rendering the
// same.
func TestRenderEquivalenceCoversWrappersAndEntities(t *testing.T) {
	cases := map[string]string{
		"a comment suffix": "## Audit Notes <!-- keep -->\n\n" + sentinelAuditNotes + "\n",
		"a span wrapper":   "## <span>Audit Notes</span>\n\n" + sentinelAuditNotes + "\n",
		"a link wrapper":   "## [Audit Notes](#audit-notes)\n\n" + sentinelAuditNotes + "\n",
		"an entity":        "## Audit&nbsp;Notes\n\n" + sentinelAuditNotes + "\n",
		// One pass decodes this to "Audit & Notes", which slugs onto the excluded
		// title. Skipping the assertion when the run refuses would have made this
		// probe check nothing at all, which is how it sat for a round.
		"an amp entity": "## Audit &amp; Notes\n\n" + sentinelAuditNotes + "\n",
	}
	for what, body := range cases {
		root := fixtureRepo(t)
		writeFile(t, root, ".abcd/development/specs/open/spc-14-wrapped.md",
			"---\nid: spc-14\n---\n\n# A spec\n\nProse.\n\n"+body)
		gitCommitAll(t, root)
		refusesOrWithholds(t, root, "a heading carrying "+what, sentinelAuditNotes)
	}
}

// TestYAMLKeySpellingsAreRecognised: three top-level spellings of an excluded
// key that neither the field reader nor a plain `key:` pattern reports.
func TestYAMLKeySpellingsAreRecognised(t *testing.T) {
	cases := map[string]string{
		"an explicit key":          "---\nid: spc-15\n? origin\n: " + sentinelOrigin + "\n---\n",
		"a top-level flow map":     "---\n{id: spc-15, origin: " + sentinelOrigin + "}\n---\n",
		"a nested flow map":        "---\nid: spc-15\nmeta: {origin: " + sentinelOrigin + "}\n---\n",
		"a quoted key with escape": "---\nid: spc-15\n\"ori\\u0067in\": " + sentinelOrigin + "\n---\n",
	}
	for what, front := range cases {
		root := fixtureRepo(t)
		writeFile(t, root, ".abcd/development/specs/open/spc-15-keys.md",
			front+"\n# A spec\n\nProse.\n")
		gitCommitAll(t, root)

		res, err := Assemble(AssembleRequest{
			RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true,
		})
		if err != nil {
			continue
		}
		if strings.Contains(bundleText(res.Bundle), sentinelOrigin) {
			t.Errorf("%s let an excluded key travel", what)
		}
	}
}

// TestRawHTMLHeadingFormsAreRecognised: the per-line closed-tag pattern is one
// spelling of a raw HTML heading among several, and a scan that only sees that
// one is a spelling check again.
func TestRawHTMLHeadingFormsAreRecognised(t *testing.T) {
	cases := map[string]string{
		"split across lines":  "<h2>\nAudit Notes\n</h2>\n\n" + sentinelAuditNotes + "\n",
		"never closed":        "<h2>Audit Notes\n\n" + sentinelAuditNotes + "\n",
		"carrying attributes": "<h2 id=\"x\" class=\"y\">Audit Notes</h2>\n\n" + sentinelAuditNotes + "\n",
		"opened on its own":   "<h2 >\nAudit Notes\n\n" + sentinelAuditNotes + "\n",
	}
	for what, body := range cases {
		root := fixtureRepo(t)
		writeFile(t, root, ".abcd/development/specs/open/spc-16-html.md",
			"---\nid: spc-16\n---\n\n# A spec\n\nProse.\n\n"+body)
		gitCommitAll(t, root)

		res, err := Assemble(AssembleRequest{
			RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true,
		})
		if err != nil {
			continue
		}
		if strings.Contains(bundleText(res.Bundle), sentinelAuditNotes) {
			t.Errorf("a raw HTML heading %s let an excluded section travel", what)
		}
	}
}

// TestUnresolvableFrontmatterShapesRefuse: three YAML constructions whose keys
// this package cannot resolve without becoming a YAML parser. The
// escape-is-the-signal rationale covers them; the pattern set did not.
func TestUnresolvableFrontmatterShapesRefuse(t *testing.T) {
	cases := map[string]string{
		"a tagged key":                "---\nid: spc-17\n!!str origin: " + sentinelOrigin + "\n---\n",
		"an anchored key":             "---\nid: spc-17\n&a origin: " + sentinelOrigin + "\n---\n",
		"a block-scalar explicit key": "---\nid: spc-17\n? |\n  origin\n: " + sentinelOrigin + "\n---\n",
	}
	for what, front := range cases {
		root := fixtureRepo(t)
		writeFile(t, root, ".abcd/development/specs/open/spc-17-shapes.md", front+"\n# A spec\n\nProse.\n")
		gitCommitAll(t, root)

		if _, err := Assemble(AssembleRequest{
			RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true,
		}); err == nil {
			t.Errorf("%s did not refuse", what)
		}
	}
}

// TestBlockScalarEndingInAnExcludedTitleIsNotASetextHeading: the setext scan ran
// over frontmatter lines, so a block scalar whose last line is the excluded
// title, sitting above the closing `---`, was refused as an underlined heading —
// a true refusal reached by a false reading, which teaches the wrong fix.
func TestBlockScalarEndingInAnExcludedTitleIsNotASetextHeading(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, ".abcd/development/specs/open/spc-18-scalar.md",
		"---\nid: spc-18\nsummary: |\n  a first line\n  Audit Notes\n---\n\n# A spec\n\nProse.\n")
	gitCommitAll(t, root)

	if _, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true,
	}); err != nil && strings.Contains(err.Error(), "underlines") {
		t.Errorf("a block scalar inside the frontmatter was read as a setext heading: %v", err)
	}
}

// TestQuotedScalarMentioningAKeyDoesNotRefuse: the flow-mapping pattern was
// unanchored, so a quoted reason string that merely QUOTES a flow mapping
// refused the run. The corpus writes long quoted reasons; this is one sentence
// away from a repository that cannot assemble.
func TestQuotedScalarMentioningAKeyDoesNotRefuse(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, ".abcd/development/specs/open/spc-19-quoted.md",
		"---\nid: spc-19\nreason: \"we stamped {origin: scribe} on the record and moved on\"\n---\n\n# A spec\n\nProse.\n")
	gitCommitAll(t, root)

	if _, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true,
	}); err != nil {
		t.Fatalf("a quoted scalar mentioning a key refused the assembly: %v", err)
	}
}

// TestHeadingFollowedByAnAutolinkDoesNotRefuse: stripping an autolink as a tag
// turned a heading carrying a URL into a different heading.
func TestHeadingFollowedByAnAutolinkDoesNotRefuse(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, ".abcd/development/specs/open/spc-20-autolink.md",
		"---\nid: spc-20\n---\n\n# A spec\n\n## See <https://example.invalid/spec>\n\nProse.\n")
	gitCommitAll(t, root)

	if _, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true,
	}); err != nil {
		t.Fatalf("a heading carrying an autolink refused the assembly: %v", err)
	}
}

// TestDefaultRunDirectoryRefusalNamesIt: the label falls back to the resolved
// directory only when the caller supplied no spelling of its own; the default
// run directory is the assembler's own, so it must name itself.
func TestDefaultRunDirectoryRefusalNamesIt(t *testing.T) {
	root := fixtureRepo(t)
	// Re-run into the same directory by pinning the mint, so the default lands
	// where a run already sits — the collision the fallback has to survive.
	setMinter(t, fixedMinter("2608301200", 789))
	first, err := Assemble(AssembleRequest{RepoRoot: root, Position: PositionWidening, Target: "HEAD"})
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	_, err = Assemble(AssembleRequest{RepoRoot: root, Position: PositionWidening, Target: "HEAD"})
	if err == nil {
		t.Fatal("a second run into one default directory was accepted")
	}
	if !strings.Contains(err.Error(), first.RunID) {
		t.Errorf("the refusal does not name the directory: %v", err)
	}
}

// spcWithFrontmatter writes one spec with the given frontmatter body and commits.
func spcWithFrontmatter(t *testing.T, root, name, front string) {
	t.Helper()
	writeFile(t, root, ".abcd/development/specs/open/"+name, front+"\n# A spec\n\nProse.\n")
	gitCommitAll(t, root)
}

// refusesOrWithholds asserts the sentinel never reaches the bundle, by refusal or
// by redaction. A refusal IS the floor working; what must not happen is a clean
// run carrying the field.
func refusesOrWithholds(t *testing.T, root, what, sentinel string) {
	t.Helper()
	res, err := Assemble(AssembleRequest{RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true})
	if err != nil {
		return
	}
	if strings.Contains(bundleText(res.Bundle), sentinel) {
		t.Errorf("%s let the excluded field travel", what)
	}
}

// TestFlowMappingKeyShapesAreRecognised covers the flow-mapping spellings an
// anchored scan gave up. Anchoring fixed a false positive by dropping a class.
func TestFlowMappingKeyShapesAreRecognised(t *testing.T) {
	rows := map[string]string{
		"a flow map in a flow sequence":   "---\nid: spc-21\nmeta: [{origin: " + sentinelOrigin + "}]\n---\n",
		"a tagged flow map":               "---\nid: spc-21\nmeta: !!map {origin: " + sentinelOrigin + "}\n---\n",
		"an anchored flow map":            "---\nid: spc-21\nmeta: &m {origin: " + sentinelOrigin + "}\n---\n",
		"a key after a newline in braces": "---\nid: spc-21\nmeta: {a: 1\n, origin: " + sentinelOrigin + "}\n---\n",
		"a flow map in a sequence item":   "---\nid: spc-21\nmeta:\n  - {origin: " + sentinelOrigin + "}\n---\n",
		"a flow map among others":         "---\nid: spc-21\nmeta: [{a: 1}, {origin: " + sentinelOrigin + "}]\n---\n",
	}
	for what, front := range rows {
		root := fixtureRepo(t)
		spcWithFrontmatter(t, root, "spc-21-flow.md", front)
		refusesOrWithholds(t, root, what, sentinelOrigin)
	}
}

// TestSetextSurvivesAnOvershootingFrontmatterCloser: the frontmatter closer the
// stripper looks for is `---`, so a block closed by `...` or never closed makes
// the reported offset overshoot into the body — past a setext heading, which
// then travelled whole.
func TestSetextSurvivesAnOvershootingFrontmatterCloser(t *testing.T) {
	rows := map[string]string{
		"a first block closed by three dots": "---\nid: spc-22\n...\n\n# A spec\n\nAudit Notes\n---\n\n" + sentinelAuditNotes + "\n",
		"a first block never closed":         "---\nid: spc-22\n\n# A spec\n\nAudit Notes\n===\n\n" + sentinelAuditNotes + "\n",
	}
	for what, body := range rows {
		root := fixtureRepo(t)
		writeFile(t, root, ".abcd/development/specs/open/spc-22-closer.md", body)
		gitCommitAll(t, root)
		refusesOrWithholds(t, root, what, sentinelAuditNotes)
	}
}

// TestTagAndExplicitKeyShapesAreRecognised: the unresolvable-shape floor caught
// the double-bang shorthand only, and read an explicit-key line as safe whenever
// its own pattern happened not to match.
func TestTagAndExplicitKeyShapesAreRecognised(t *testing.T) {
	rows := map[string]string{
		"a local tag":                "---\nid: spc-23\n!local origin: " + sentinelOrigin + "\n---\n",
		"a verbatim tag":             "---\nid: spc-23\n!<tag:example.invalid,2026:o> origin: " + sentinelOrigin + "\n---\n",
		"an explicit key with a tag": "---\nid: spc-23\n? !!str origin\n: " + sentinelOrigin + "\n---\n",
		"an explicit key anchored":   "---\nid: spc-23\n? &a origin\n: " + sentinelOrigin + "\n---\n",
		"an explicit key escaped":    "---\nid: spc-23\n? \"ori\\u0067in\"\n: " + sentinelOrigin + "\n---\n",
	}
	for what, front := range rows {
		root := fixtureRepo(t)
		spcWithFrontmatter(t, root, "spc-23-tags.md", front)
		refusesOrWithholds(t, root, what, sentinelOrigin)
	}
}

// TestRawHeadingSpanningAFenceIsRecognised: blanking fenced lines before the
// scan erased the heading's own text, so a block opened outside a fence that
// runs through fence delimiters was mis-titled and admitted.
func TestRawHeadingSpanningAFenceIsRecognised(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, ".abcd/development/specs/open/spc-24-fence.md",
		"---\nid: spc-24\n---\n\n# A spec\n\n<h2>\n```\nAudit Notes\n```\n</h2>\n\n"+sentinelAuditNotes+"\n")
	gitCommitAll(t, root)
	refusesOrWithholds(t, root, "a raw heading spanning a fence", sentinelAuditNotes)
}

// TestHeadingRoleDivIsRecognised: an element carrying a heading ROLE renders as
// a heading to an assistive reader and to a human, and no h-tag appears.
func TestHeadingRoleDivIsRecognised(t *testing.T) {
	rows := map[string]string{
		"a div with a heading role":  "<div role=\"heading\" aria-level=\"2\">Audit Notes</div>\n\n" + sentinelAuditNotes + "\n",
		"a span with a heading role": "<span role=heading>Audit Notes</span>\n\n" + sentinelAuditNotes + "\n",
	}
	for what, body := range rows {
		root := fixtureRepo(t)
		writeFile(t, root, ".abcd/development/specs/open/spc-25-role.md",
			"---\nid: spc-25\n---\n\n# A spec\n\nProse.\n\n"+body)
		gitCommitAll(t, root)
		refusesOrWithholds(t, root, what, sentinelAuditNotes)
	}
}

// mustAssemble runs one assembly and requires it to succeed. A floor that
// refuses an ordinary document stops every assembly the repository can run, so
// the negative cases matter as much as the positive ones.
func mustAssemble(t *testing.T, root, what string) {
	t.Helper()
	if _, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true,
	}); err != nil {
		t.Errorf("%s refused the assembly: %v", what, err)
	}
}

// TestQuoteBlankerOnlyBlanksScalarOpeners: a quote opens a scalar only in
// scalar position. Treating every quote as an opener let an apostrophe in
// ordinary prose, a stray quote, or an escaped quote pair with a later one and
// blank the excluded key sitting between them.
func TestQuoteBlankerOnlyBlanksScalarOpeners(t *testing.T) {
	rows := map[string]string{
		"apostrophes in plain scalars": "---\nid: spc-26\nmeta: {note: it's, origin: " + sentinelOrigin + ", kind: don't}\n---\n",
		"a stray double quote":         "---\nid: spc-26\nmeta: {note: say \" hello, origin: " + sentinelOrigin + "}\n---\n",
		"an escaped quote scalar":      "---\nid: spc-26\nmeta: {a: \"\\\"\", origin: " + sentinelOrigin + "}\n---\n",
	}
	for what, front := range rows {
		root := fixtureRepo(t)
		spcWithFrontmatter(t, root, "spc-26-quotes.md", front)
		refusesOrWithholds(t, root, what, sentinelOrigin)
	}

	// And the false positive the blanking exists to prevent stays prevented: a
	// quoted reason that merely quotes a flow mapping, escapes and all.
	root := fixtureRepo(t)
	spcWithFrontmatter(t, root, "spc-27-reason.md",
		"---\nid: spc-27\nreason: \"say \\\"{origin: scribe}\\\" done\"\n---\n")
	mustAssemble(t, root, "a quoted reason carrying an escaped flow mapping")
}

// TestRawHeadingTitleSurvivesInlineMarkup: cutting the title at the first
// closing tag of ANY element truncates or empties it whenever the heading holds
// inline markup, and the section is admitted.
func TestRawHeadingTitleSurvivesInlineMarkup(t *testing.T) {
	rows := map[string]string{
		"an anchor before the text":  "<h2><a id=\"x\"></a>Audit Notes</h2>\n\n" + sentinelAuditNotes + "\n",
		"emphasis on the first word": "<h2><em>Audit</em> Notes</h2>\n\n" + sentinelAuditNotes + "\n",
	}
	for what, body := range rows {
		root := fixtureRepo(t)
		writeFile(t, root, ".abcd/development/specs/open/spc-28-inline.md",
			"---\nid: spc-28\n---\n\n# A spec\n\nProse.\n\n"+body)
		gitCommitAll(t, root)
		refusesOrWithholds(t, root, "a raw heading with "+what, sentinelAuditNotes)
	}
}

// TestIndentedEllipsisDoesNotCloseTheBlock: YAML closes a document with `...` at
// column 0. Trimming indentation first made an ellipsis inside a block scalar
// close the block, and the file was refused for a shape it does not have.
func TestIndentedEllipsisDoesNotCloseTheBlock(t *testing.T) {
	root := fixtureRepo(t)
	spcWithFrontmatter(t, root, "spc-29-ellipsis.md",
		"---\nid: spc-29\nsummary: |\n  a first line\n  ...\n  a last line\n---\n")
	mustAssemble(t, root, "an indented ellipsis inside a block scalar")
}

// TestBodyRulesAreNotFrontmatter: the stripper recognises a block only at line
// 0, so anything else that opens with three dashes is a thematic break. Reading
// the first `---` found ANYWHERE as a frontmatter opener turned an ordinary
// documentation page into a refusal.
func TestBodyRulesAreNotFrontmatter(t *testing.T) {
	rows := map[string]string{
		"a lone thematic break":        "# A page\n\nProse.\n\n---\n\nMore prose.\n",
		"a break above an image line":  "# A page\n\nProse.\n\n---\n\n![a diagram](x.svg)\n\nMore prose.\n",
		"a break above an anchor line": "# A page\n\nProse.\n\n---\n\n&mdash; an aside\n\nMore prose.\n",
		"a break above a dots line":    "# A page\n\nProse.\n\n---\n\n...and it continues.\n",
	}
	for what, body := range rows {
		root := fixtureRepo(t)
		writeFile(t, root, "docs/reference/page.md", body)
		gitCommitAll(t, root)
		mustAssemble(t, root, what)
	}
}

// TestFrontmatterCloserIsNotASetextUnderline: the closing `---` sits directly
// under the last frontmatter line, so reading it as an underline makes every
// record whose last key happens to name an excluded heading refuse.
func TestFrontmatterCloserIsNotASetextUnderline(t *testing.T) {
	root := fixtureRepo(t)
	spcWithFrontmatter(t, root, "spc-30-closer.md", "---\nid: spc-30\nAudit Notes\n---\n")
	mustAssemble(t, root, "a frontmatter closer under a column-0 last line")
}

// TestQuotedFlowKeyIsRecognised: a quoted token sitting in scalar position is a
// KEY when a colon follows it, and the blanker treated it as a scalar — so the
// span was blanked before the flow scan could read it and the excluded key
// travelled under a manifest asserting its refusal.
func TestQuotedFlowKeyIsRecognised(t *testing.T) {
	rows := map[string]string{
		"a quoted key opening a flow map": "---\nid: spc-31\nmeta: {\"origin\": " + sentinelOrigin + "}\n---\n",
		"a quoted key after a comma":      "---\nid: spc-31\nmeta: {a: 1, 'origin': " + sentinelOrigin + "}\n---\n",
		"a quoted key in a flow sequence": "---\nid: spc-31\nmeta: [{'origin': " + sentinelOrigin + "}]\n---\n",
		"a quoted key on a continuation":  "---\nid: spc-31\nmeta: {a: 1\n, \"origin\": " + sentinelOrigin + "}\n---\n",
	}
	for what, front := range rows {
		root := fixtureRepo(t)
		spcWithFrontmatter(t, root, "spc-31-quoted-key.md", front)
		refusesOrWithholds(t, root, what, sentinelOrigin)
	}
}

// TestScalarOpenersNeedTheirYAMLWhitespace: a colon and a dash are YAML
// indicators only with the whitespace YAML requires after them. Reading either
// as an opener without it let a quote inside a plain scalar open a span that
// blanked the excluded key beside it.
func TestScalarOpenersNeedTheirYAMLWhitespace(t *testing.T) {
	rows := map[string]string{
		"a colon with nothing after it": "---\nid: spc-32\nmeta: {a:'b, origin: " + sentinelOrigin + ", c: 'd}\n---\n",
		"a dash that is not a sequence": "---\nid: spc-32\nmeta: {a: b - 'c, origin: " + sentinelOrigin + ", d: 'e}\n---\n",
		"a dash with nothing after it":  "---\nid: spc-32\nmeta: {a: b-'c, origin: " + sentinelOrigin + ", d: 'e}\n---\n",
		"a colon inside a plain scalar": "---\nid: spc-32\nmeta: {a: b:\"c, origin: " + sentinelOrigin + ", d: \"e}\n---\n",
	}
	for what, front := range rows {
		root := fixtureRepo(t)
		spcWithFrontmatter(t, root, "spc-32-whitespace.md", front)
		refusesOrWithholds(t, root, what, sentinelOrigin)
	}
}

// TestQuotedKeyNamesOnlyItself: the interior of a quoted key is its NAME, not a
// place to read further keys out of. Skipping such a span rather than reading it
// as one token would let `{"a, origin: b": 1}` — whose only key is the whole
// quoted string — refuse a document that carries no excluded key at all.
func TestQuotedKeyNamesOnlyItself(t *testing.T) {
	root := fixtureRepo(t)
	spcWithFrontmatter(t, root, "spc-33-key-interior.md",
		"---\nid: spc-33\nmeta: {\"a, origin: b\": 1}\n---\n")
	mustAssemble(t, root, "a quoted key whose own name mentions a flow key")
}

// TestSequenceDashOpensAScalarAtLineStart: the dash the fix bounds to line start
// still has to open one there, or a quoted sequence item mentioning a flow
// mapping refuses the run.
func TestSequenceDashOpensAScalarAtLineStart(t *testing.T) {
	root := fixtureRepo(t)
	spcWithFrontmatter(t, root, "spc-34-sequence.md",
		"---\nid: spc-34\nmeta:\n  - 'we stamped {origin: scribe} and moved on'\n  - - 'and again {origin: scribe}'\n---\n")
	mustAssemble(t, root, "a quoted sequence item carrying a flow mapping")
}

// TestBodyRuleAboveAnExcludedKeyLineDoesNotRefuse: the key scan opened its block
// at the first `---` found ANYWHERE while every other scan reads a block at line
// 0 only, so an ordinary documentation page with a thematic break above a line
// spelled like a key was refused as frontmatter carrying an excluded key.
func TestBodyRuleAboveAnExcludedKeyLineDoesNotRefuse(t *testing.T) {
	rows := map[string]string{
		"a break above a key-shaped line":   "# A page\n\nProse.\n\n---\n\norigin: where the claim came from\n\nMore prose.\n",
		"a break above a flow-mapping line": "# A page\n\nProse.\n\n---\n\nWe write {origin: scribe} on the record.\n",
		"a break above a quoted key line":   "# A page\n\nProse.\n\n---\n\n\"production_mode\": the phrase, quoted.\n",
	}
	for what, body := range rows {
		root := fixtureRepo(t)
		writeFile(t, root, "docs/reference/page.md", body)
		gitCommitAll(t, root)
		mustAssemble(t, root, what)
	}
}

// TestRawHeadingBoundsIgnoreMarkupData: a closing tag inside an HTML comment or
// inside a quoted attribute value is DATA, and so is a greater-than inside one.
// Reading any of the three as structure cut the title short, and a heading every
// browser renders as the excluded one was judged as something else.
func TestRawHeadingBoundsIgnoreMarkupData(t *testing.T) {
	rows := map[string]string{
		"a close inside a comment":           "<h2>Audit<!-- </h2> --> Notes</h2>\n\n" + sentinelAuditNotes + "\n",
		"a close inside an attribute value":  "<h2><a title=\"</h2>\"></a>Audit Notes</h2>\n\n" + sentinelAuditNotes + "\n",
		"a greater-than in an attribute":     "<h2 title=\"a>b\">Audit Notes</h2>\n\n" + sentinelAuditNotes + "\n",
		"a heading open inside an attribute": "<h2 title=\"<h3>\">Audit Notes</h2>\n\n" + sentinelAuditNotes + "\n",
	}
	for what, body := range rows {
		root := fixtureRepo(t)
		writeFile(t, root, ".abcd/development/specs/open/spc-35-bounds.md",
			"---\nid: spc-35\n---\n\n# A spec\n\nProse.\n\n"+body)
		gitCommitAll(t, root)
		refusesOrWithholds(t, root, "a raw heading with "+what, sentinelAuditNotes)
	}
}

// TestRawHeadingTitleCrossesBlankLinesAndBreaks: a blank line inside a heading
// element is whitespace to every renderer, and a line break element separates
// two words rather than joining them. Bounding at the blank line unconditionally
// emptied the first title, and dropping a tag without the space it stands for
// spelled the second as one word.
func TestRawHeadingTitleCrossesBlankLinesAndBreaks(t *testing.T) {
	rows := map[string]string{
		"a blank line inside the element": "<h2>\n\nAudit Notes</h2>\n\n" + sentinelAuditNotes + "\n",
		"a line break inside the title":   "<h2>Audit<br>Notes</h2>\n\n" + sentinelAuditNotes + "\n",
		"a self-closing break":            "<h2>Audit<br/>Notes</h2>\n\n" + sentinelAuditNotes + "\n",
	}
	for what, body := range rows {
		root := fixtureRepo(t)
		writeFile(t, root, ".abcd/development/specs/open/spc-36-breaks.md",
			"---\nid: spc-36\n---\n\n# A spec\n\nProse.\n\n"+body)
		gitCommitAll(t, root)
		refusesOrWithholds(t, root, "a raw heading with "+what, sentinelAuditNotes)
	}
}

// TestRawHeadingCaseIsFolded: the element name is compared case-insensitively,
// which the per-name compiled pattern got from its own flags. Dropping that
// cache must not drop the fold with it.
func TestRawHeadingCaseIsFolded(t *testing.T) {
	rows := map[string]string{
		"an upper-case pair": "<H2>Audit Notes</H2>\n\n" + sentinelAuditNotes + "\n",
		"a mixed-case close": "<h2>Audit Notes</H2>\n\n" + sentinelAuditNotes + "\n",
		// The one row that DISCRIMINATES the fold: with a blank line after the
		// close, the soft bound rescues the title whether the name folded or
		// not, so every other row stays green with byte equality in place of the
		// fold. Here the close is the only bound there is.
		"a mixed-case close with no blank line after it": "<h2>Audit Notes</H2>\nand trailing prose\n\n" +
			sentinelAuditNotes + "\n",
		"a role element pair": "<DIV role=\"heading\" aria-level=\"2\">Audit Notes</DIV>\n\n" + sentinelAuditNotes + "\n",
	}
	for what, body := range rows {
		root := fixtureRepo(t)
		writeFile(t, root, ".abcd/development/specs/open/spc-37-case.md",
			"---\nid: spc-37\n---\n\n# A spec\n\nProse.\n\n"+body)
		gitCommitAll(t, root)
		refusesOrWithholds(t, root, "a raw heading with "+what, sentinelAuditNotes)
	}
}

// TestTagInsideAWordDoesNotSplitTheTitle: a tag removed from a title stands for
// a boundary in `Audit<br>Notes` and for no boundary at all in
// `Audi<i>t</i> Notes`. Replacing every tag with a space closed the first and
// opened the second on all four heading paths, so BOTH readings are taken and a
// title is excluded if either one names an excluded heading.
func TestTagInsideAWordDoesNotSplitTheTitle(t *testing.T) {
	rows := map[string]string{
		"a section heading":   "## Audi<i>t</i> Notes\n\n" + sentinelAuditNotes + "\n",
		"an indented heading": "  ## Audi<i>t</i> Notes\n\n" + sentinelAuditNotes + "\n",
		"a setext heading":    "Audi<i>t</i> Notes\n===\n\n" + sentinelAuditNotes + "\n",
		"a raw HTML heading":  "<h2>Audit Note<sup>s</sup></h2>\n\n" + sentinelAuditNotes + "\n",
	}
	for what, body := range rows {
		root := fixtureRepo(t)
		writeFile(t, root, ".abcd/development/specs/open/spc-38-split.md",
			"---\nid: spc-38\n---\n\n# A spec\n\nProse.\n\n"+body)
		gitCommitAll(t, root)
		refusesOrWithholds(t, root, "a tag inside a word in "+what, sentinelAuditNotes)
	}
}

// TestUnclosedAttributeQuoteDoesNotEraseAHeading: an attribute value ends at its
// own tag. Ending it at the next matching quote found anywhere in the document
// let one unbalanced quote blank every angle bracket up to some unrelated quote
// thousands of bytes later, erasing a raw HTML heading from the scan.
func TestUnclosedAttributeQuoteDoesNotEraseAHeading(t *testing.T) {
	rows := map[string]string{
		"a double quote": "<div id=\"\n\n<h2>Audit Notes</h2>\n\n" + sentinelAuditNotes + "\n\n\">\n",
		"a single quote": "<a href='x\n\n<h2>Audit Notes</h2>\n\n" + sentinelAuditNotes + "\n\n'>\n",
	}
	for what, body := range rows {
		root := fixtureRepo(t)
		writeFile(t, root, ".abcd/development/specs/open/spc-39-unclosed.md",
			"---\nid: spc-39\n---\n\n# A spec\n\nProse.\n\n"+body)
		gitCommitAll(t, root)
		refusesOrWithholds(t, root, "an attribute value opened with "+what, sentinelAuditNotes)
	}
}

// TestCommentedOutExcludedHeadingRefuses: the blind reader receives raw
// markdown, not a rendered page, so a file that still CARRIES an excluded
// heading refuses the run wherever the heading stands. The markup mask exists to
// stop a comment's or an attribute value's brackets from being read as
// STRUCTURE; it must never stop them from being read as CONTENT.
func TestCommentedOutExcludedHeadingRefuses(t *testing.T) {
	rows := map[string]string{
		"inside an HTML comment": "<!-- open\n\n<h2>Audit Notes</h2>\n\n" +
			sentinelAuditNotes + "\n\nclose -->\n",
		"inside an attribute value": "<div title=\"<h2>Audit Notes</h2>\">\n\n" +
			sentinelAuditNotes + "\n",
	}
	for what, body := range rows {
		root := fixtureRepo(t)
		writeFile(t, root, ".abcd/development/specs/open/spc-40-commented.md",
			"---\nid: spc-40\n---\n\n# A spec\n\nProse.\n\n"+body)
		gitCommitAll(t, root)
		refusesOrWithholds(t, root, "an excluded heading "+what, sentinelAuditNotes)
	}
}

// TestEscapedAndCarriageReturnedKeysAreRecognised: deleting the line-anchored
// escaped-key refusal was justified as a subsumption and was not one. A double
// quoted key whose closing quote is itself escaped never closes its token, so
// the positional scanner reports no quoted key at all and its escape refusal is
// never reached; and a carriage return is YAML whitespace the scanner's own
// blank skip did not cover.
func TestEscapedAndCarriageReturnedKeysAreRecognised(t *testing.T) {
	rows := map[string]string{
		"an escaped closing quote": "---\nid: spc-41\n\"origin\\\": " + sentinelOrigin + "\n---\n",
		"a carriage return before a colon in a flow map": "---\nid: spc-41\nmeta: {\"origin\"\r: " +
			sentinelOrigin + "}\n---\n",
	}
	for what, front := range rows {
		root := fixtureRepo(t)
		spcWithFrontmatter(t, root, "spc-41-escaped.md", front)
		refusesOrWithholds(t, root, what, sentinelOrigin)
	}
}

// TestAttributeValueSpanningALineDoesNotHideTheOpener: where an attribute value
// ends is ambiguous, and the two honest answers disagree about exactly this
// document. A browser reads `<h2 title="a>` continued on the next line as one
// tag and shows the excluded heading; the conservative reading, which ends a
// value on its own line, declines to mask it — and the opener pattern cannot
// cross the `>` inside the value, so the unmasked text reads a different
// heading. Only taking both maskings sees it.
func TestAttributeValueSpanningALineDoesNotHideTheOpener(t *testing.T) {
	rows := map[string]string{
		"a double-quoted value": "<h2 title=\"a>\nb\">Audit Notes</h2>\n\n" + sentinelAuditNotes + "\n",
		"a single-quoted value": "<h2 title='a>\nb'>Audit Notes</h2>\n\n" + sentinelAuditNotes + "\n",
		"a role element":        "<div role=\"heading\" title=\"a>\nb\">Audit Notes</div>\n\n" + sentinelAuditNotes + "\n",
		"a harmless attribute first": "<h2 id=\"x\" title=\"a>\nb\">Audit Notes</h2>\n\n" +
			sentinelAuditNotes + "\n",
	}
	for what, body := range rows {
		root := fixtureRepo(t)
		writeFile(t, root, ".abcd/development/specs/open/spc-42-spanning.md",
			"---\nid: spc-42\n---\n\n# A spec\n\nProse.\n\n"+body)
		gitCommitAll(t, root)
		refusesOrWithholds(t, root, "an excluded heading behind "+what, sentinelAuditNotes)
	}
}

// TestAStrayQuoteDoesNotConsumeALaterValuesMasking: the line-bounded reading is
// load-bearing in the other direction. An unbalanced quote earlier in the file
// pairs, under the browser reading, with the opening quote of a LATER legitimate
// value, so that value is never masked and its `>` still truncates the opener.
func TestAStrayQuoteDoesNotConsumeALaterValuesMasking(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, ".abcd/development/specs/open/spc-43-stray.md",
		"---\nid: spc-43\n---\n\n# A spec\n\n<div id=\"\n\nprose\n\n"+
			"<h2 title=\"a>b\">Audit Notes</h2>\n\n"+sentinelAuditNotes+"\n")
	gitCommitAll(t, root)
	refusesOrWithholds(t, root, "a stray quote ahead of a legitimate attribute value", sentinelAuditNotes)
}

// TestFlowPathEscapedKeyIsRecognised: the escape refusal exists TWICE because
// neither spelling reaches the other. The line-anchored rule cannot see a key
// inside a flow mapping or under a sequence dash, so the positional scanner
// carries its own — and that half had no test, which is exactly how the
// line-anchored half came to be deleted as redundant in the round before.
func TestFlowPathEscapedKeyIsRecognised(t *testing.T) {
	rows := map[string]string{
		"in a flow mapping":      "---\nid: spc-44\nmeta: {\"ori\\u0067in\": " + sentinelOrigin + "}\n---\n",
		"under a sequence dash":  "---\nid: spc-44\nmeta:\n  - \"ori\\u0067in\": " + sentinelOrigin + "\n---\n",
		"after another flow key": "---\nid: spc-44\nmeta: {a: 1, \"ori\\u0067in\": " + sentinelOrigin + "}\n---\n",
	}
	for what, front := range rows {
		root := fixtureRepo(t)
		spcWithFrontmatter(t, root, "spc-44-flowescape.md", front)
		refusesOrWithholds(t, root, "an escaped excluded key "+what, sentinelOrigin)
	}
}

// TestRawHeadingTitleIsReadFromEveryMasking: the maskings bound a title AND are
// read for it. An inline element whose attribute value carries a greater-than
// truncates the tag in the unmasked rest, so the unmasked rendering reads a
// different heading and only the masked rendering names the excluded one. The
// bound dimension was pinned; this is the rendering dimension.
func TestRawHeadingTitleIsReadFromEveryMasking(t *testing.T) {
	rows := map[string]string{
		"an inline element inside the heading": "<h2><span title=\"a>c\">Audit Notes</span></h2>\n\n" +
			sentinelAuditNotes + "\n",
		"a trailing inline element": "<h2>Audit Notes<span title=\"a>c\"></span></h2>\n\n" +
			sentinelAuditNotes + "\n",
	}
	for what, body := range rows {
		root := fixtureRepo(t)
		writeFile(t, root, ".abcd/development/specs/open/spc-45-rendering.md",
			"---\nid: spc-45\n---\n\n# A spec\n\nProse.\n\n"+body)
		gitCommitAll(t, root)
		refusesOrWithholds(t, root, "a raw heading carrying "+what, sentinelAuditNotes)
	}
}
