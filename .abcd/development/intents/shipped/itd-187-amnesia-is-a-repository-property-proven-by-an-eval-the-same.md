---
id: itd-187
slug: amnesia-is-a-repository-property-proven-by-an-eval-the-same
spec_id: spc-65
kind: standalone
suggested_kind: standalone
reclassification_history: []
builds_on: []
severity: minor
impact: additive
---

# Amnesia is a repository property, proven by an eval — the same state assembled twice is byte-identical, so no case run is spent evidencing it

## Press Release

> **Amnesia is a property of what the assembler passes, not an instruction
> to an agent.** The eval assembles one definition twice over an unchanged
> repository state and asserts the two assembled inputs are byte-identical
> — the manifest sits outside the comparison, carries content hashes and
> no timestamps, and the assembler walks paths in lexicographic order.
> Making this a repository eval means any reader can check it, and the
> closing run of Iteration 2 carries only the properties a case run can
> carry (purpose durability and convergence — never amnesia).

## What's In Scope

- The double-assembly comparison in CI, with the identity relation stated:
  byte-equality of the assembled input, manifest excluded.
- The determinism preconditions it enforces on the assembler: hash-only
  manifests, lexicographic walk order.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** an unchanged repository state at one commit, **when** one definition
  is assembled twice from two distinct filesystem paths, **then** the two
  assembled inputs are byte-identical, the manifest excluded from the
  comparison.
- **Given** the order-adversarial fixture, **when** the eval runs, **then** the
  item paths in the assembled input agree with the eval's own lexicographic
  sort, so a consistent-but-not-lexicographic order fails.
- **Given** a manifest carrying a timestamp-shaped key or a timestamp-shaped
  scalar value, **when** the eval runs, **then** it fails.
- **Given** two artefacts differing only in item order, and two differing only in
  one scalar value, **when** the comparator runs over each pair, **then** it
  reports a difference naming the differing item.

**Disclosed residue (ac-2 to ac-4).** A nondeterminism introduced into the
shipped assembler is a precondition no eval may establish for itself, because an
eval must not patch the code under test. The three criteria above catch each
named nondeterminism through an oracle the assembler does not supply, and the
comparator's own capacity to fail is proved by ac-4. What remains is discharged
by hand, and by a mutation that can actually establish the precondition:
rebuilding the assembler's candidate slice from a map, which makes the order
vary between runs, watched red and then reverted before the branch is pushed.
Removing the walk sort does NOT serve — it yields a different order, not an
unstable one, so the byte comparison stays green and only the order oracle
fires. That is a recorded hand-run, not a standing gate.


## Grounds

- pursued: Amnesia is a property of what the assembler passes, not an instruction an agent can be trusted to follow, and a case run could only ever exhibit it rather than prove it. It is pursued now because a case run is the scarcest thing in the cycle: making amnesia a repository eval leaves the closing run of Iteration 2 carrying only the properties a case run can carry, purpose durability and convergence, and lets any reader check the rest for themselves.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-d3041aa2b510 -->
Fidelity review — receipt rcp-d3041aa2b510 (verifier abcd:intent-auditor claude-opus-5[1m]).

Provenance: abcd:intent-auditor@claude-opus-5[1m] · rubric_hash sha256:542ed2cd51ff938717a3f47b2b332e8d47910beec0ca7ecdfd238ae7edf5ced5 · prompt_hash sha256:2faa6d95687daa0a7942ac06a912d8886fdf3bee0039da0c23b270acff7f3888
Input attestations: diff:446c607f..f462b92f on build/itd-187, merged as 607eecc3, plus the two doc-comment corrections settled in the ship commit aa26e07a@sha256:516ed07d88ea6dd374ebbcb911f0110f84ada6a9af0087d517f7063b5ea2123b;

Acceptance rollup: MET 2 · MET_WITH_CONCERNS 2 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: I ran the lane on an extracted copy of aa26e07a rather than reading it: `go test -tags coldreading ./evals/...` is green, and TestAssembledInputIsByteIdenticalAcrossRuns does what the criterion says — materialiseOrderPair commits the corpus once, copies the tree with .git so the second assembly runs at a different absolute path over the identical commit object, and compareArtefacts compares bundle.json byte for byte with manifest.json excluded, at all four positions. It is demonstrably falsifiable, which I established by mutation rather than by its name: rebuilding the candidate slice from a map turns it red at every position with 36 named differences, and a mint patched to a constant turns the freshness guard red saying the identifier is reported 'twice over', so the retry added in f462b92f did not disarm it. CONCERN 1, and the reason this is not a bare MET: the acceptance record's own prescribed proof for this criterion is false. itd-187's residue paragraph and spc-65's first Tests bullet both prescribe watching this test red by removing the assembler's walk sort; I removed the sort at assemble.go:551 and the test PASSED at all four positions while only the order oracle fired — a removed sort yields a different order, not an unstable one. The builder captured this as iss-2608311331229368 in the first commit of the range and corrected the merge-commit body, but left both shipped records saying it. CONCERN 2, named by nobody: the absolute-path subtest added as declared deviation 2 is a two-string containment check against f.Root and f.Home, not an absolute-path detector. With the shared temporary PARENT of the two fixture roots planted in the bundle, the whole lane stays green — the byte comparison cannot see it because both runs carry the same string, and the subtest is not looking for it.
  evidence: evals/coldreading_determinism_test.go:44 — "func TestAssembledInputIsByteIdenticalAcrossRuns(t *testing.T) { — measured: green on the delivered tree at all four positions"
  evidence: evals/coldreading_determinism_test.go:113 — "if diffs := compareArtefacts(bundleFile, a.BundleRaw, b.BundleRaw); len(diffs) > 0 { — bundle only; manifest.json is never compared"
  evidence: evals/coldreading_order_test.go:158 — "copyTree(t, first.Root, second.Root) — `.git` travels with the tree, so the second assembly runs over the same commit object"
  evidence: internal/core/reading/assemble.go:551 — "sort.SliceStable(out, func(i, j int) bool { — MEASURED: removing this whole sort leaves ac-1 PASS at all four positions and turns only TestWalkOrderIsLexicographic red; the prescribed falsifier cannot falsify ac-1"
  evidence: internal/core/reading/assemble.go:551 — "MEASURED: replacing the sort with a map rebuild turns ac-1 RED at every position — 'the assembled input at widening differs between two assemblies of ONE commit at two paths (36 difference(s))'"
  evidence: evals/coldreading_determinism_test.go:70 — "if ra, rb := runIdentifier(t, a), runIdentifier(t, b); ra == rb { — MEASURED: a mint patched to a constant fires it, 'report run identifier ... twice over', so the one-retry tolerance at line 67 does not disarm it"
  evidence: evals/coldreading_determinism_test.go:89 — "t.Run(\"no-absolute-path-in-either-artefact\" — MEASURED red on a manifest-only leak of f.Root, and MEASURED GREEN (whole lane ok) on a bundle leak of the shared temporary PARENT of the two roots, which is neither f.Root nor f.Home"
  evidence: .abcd/development/intents/shipped/itd-187-amnesia-is-a-repository-property-proven-by-an-eval-the-same.md:52 — "the walk sort removed by a one-line local patch, the test watched red — the prescribed hand-run, uncorrected in the shipped intent"
- ac-2 — MET: The oracle is genuinely the eval's own and genuinely independent, and I proved its reach by patching the assembler rather than by reading the comment. Independence: TestWalkOrderIsLexicographic sorts a COPY of the manifest's item paths with a comparator written in package evals, and TestOracleImportsNothingFromTheAssembler refuses any import of internal/core/reading anywhere under evals/, so the eval cannot reach the assembler's comparator even by accident. It first binds the manifest's item sequence to the bundle's by item key, so the order assertion is over the thing described rather than over the description. Corpus: the six order/ records separate byte order from all four rival comparators — I computed the orders by hand and TestFixtureOrderIsAdversarial asserts each differs from byte order. The fourth separation, path-component order, is the one added after review, and it is load-bearing: with a component-wise comparator patched into assemble.go:551 the oracle fails at ALL FOUR positions while ac-1 stays green, which is exactly the 'consistent-but-not-lexicographic order fails' the criterion demands. Removing the assembler's sort entirely also fires it. On iss-2608311418202794 the scoping is right and I confirmed it: the criterion says 'item paths', the oracle compares paths, and a stable reversal of field order WITHIN one path passes both this lane and `go test ./internal/core/reading/` — that is the assembler's degree of freedom, spc-61's, and spc-65's Out of scope already assigns the projection there.
  evidence: evals/coldreading_order_test.go:295 — "want := sortedCopy(paths, func(x, y string) bool { return x < y }) — the eval's own sort, on a copy"
  evidence: evals/coldreading_test.go:172 — "func TestOracleImportsNothingFromTheAssembler(t *testing.T) { — bans internal/core/reading in any import path under evals/"
  evidence: evals/coldreading_order_test.go:309 — "func requireItemSequencesAgree(t *testing.T, a assembled) { — binds the manifest's order to the bundle's by item key before the oracle runs"
  evidence: evals/coldreading_order_test.go:256 — "name: \"directory-walk order\" — the fourth separation; MEASURED: a component-wise comparator patched into the assembler turns TestWalkOrderIsLexicographic red at all four positions"
  evidence: evals/coldreading_order_test.go:83 — "Path: \".abcd/development/brief/glossary/ordering.md\" — the record that separates byte order from component order, `.` before `/`"
  evidence: evals/coldreading_order_test.go:99 — "if got, want := len(orderRecords), 6; got != want { — the declared count, duplicated rather than derived"
  evidence: internal/core/reading/assemble.go:554 — "return out[i].fieldIdx < out[j].fieldIdx — MEASURED: reversing this comparison passes BOTH the cold-reading lane and the assembler package's own tests, which is iss-2608311418202794 reproduced; it is not this criterion's property"
- ac-3 — MET_WITH_CONCERNS: The scan is real and both-sided, and I watched it fire: catches-a-planted-timestamp turns every one of six synthetic shapes into a finding naming the right location (a timestamp key at top level and on an item, an ISO date, a clock time, a packed digit run, an epoch number), passes-a-clean-manifest proves it does not fire on the shapes a real manifest carries, and the manifestKeysScanned floor fatals if the walk never reached any of fourteen named keys — so a scanner that walked nothing cannot read as a clean manifest. The nested-run_id disclosure the ship commit added is accurate: I checked the Manifest struct and every type it reaches (ManifestItem: item_key/path/field/sha256; Exclusion: rule/signal/detail, with Positions json:"-"), and no nested field is spelled run_id, so the exemption's name-at-any-depth keying is unreachable rather than merely unobserved. CONCERN 1 — the criterion is narrower than its own text on the one instance that actually occurs. The real manifest's run_id IS a timestamp-shaped scalar by construction (rdg-<yymmddHHMMSS><rrrr>, adr-45), and the packed-digit exemption is exactly what stops ac-3 failing on it. That narrowing is disclosed in the eval's code, but the acceptance record asserts the opposite: itd-187's press release says the manifest 'carries content hashes and no timestamps', which manifest.go's own header says must not be said of it. CONCERN 2, named by nobody — spc-65's justification for confining the scan is unsound for paths. packedDigitPattern is an unanchored \d{8,} and item paths are scanned as ordinary scalars, so 'the manifest carries paths, field names, and hashes only, so a timestamp-shaped token there is unambiguously a defect' is false: I fed the scan a manifest naming a real capture-id path and it reported the path as 'a packed run of digits, which is how a moment travels without punctuation'. No admitted path family carries an 8+ digit run today, so this is latent rather than live — but the premise the decision rests on is already wrong, and this repository mints 19-digit ids.
  evidence: evals/coldreading_determinism_test.go:463 — "func TestManifestCarriesNoTimestamp(t *testing.T) { — measured green on the delivered tree; its six planted shapes each produce a finding at the expected location"
  evidence: evals/coldreading_determinism_test.go:482 — "for _, key := range manifestKeysScanned { if !seen[key] { — the anti-vacuity half: fourteen keys the walk must have reached"
  evidence: evals/coldreading_determinism_test.go:440 — "packedDigitPattern.MatchString(n) && key != runIDKey && !hexValuePattern.MatchString(n) — MEASURED: a manifest item path '.abcd/work/issues/open/iss-2608311331229368-...' is reported as a packed moment; paths are not exempt"
  evidence: evals/coldreading_determinism_test.go:499 — "if id := runIdentifier(t, a); !runIDPattern.MatchString(id) { — the shape assertion, top-level only, which bounds the exemption for the real artefact"
  evidence: internal/core/reading/manifest.go:64 — "type Manifest struct { — verified closed: no nested run_id on Manifest, ManifestItem or Exclusion, which is what makes the any-depth keying unreachable"
  evidence: internal/core/reading/manifest.go:57 — "It carries no timestamp FIELD, but it is not timestamp-free and must not be described as such: RunID embeds a mint stamp by construction (adr-45)"
  evidence: .abcd/development/intents/shipped/itd-187-amnesia-is-a-repository-property-proven-by-an-eval-the-same.md:20 — "the manifest sits outside the comparison, carries content hashes and no timestamps — the description manifest.go forbids"
  evidence: .abcd/development/specs/closed/spc-65-amnesia-is-a-repository-property-proven-by-an-eval-the-same.md:96 — "The manifest carries paths, field names, and hashes only, so a timestamp-shaped token there is unambiguously a defect. — false for run_id and false for paths"
- ac-4 — MET: Both pairs the criterion names are fed to the comparator and both report a difference naming the differing item, and I proved the naming clause is live rather than decorative: dropping item-key labelling from elementLabel turns the one-scalar-inside-one-item case red with 'the comparator's report names no itm-0002, so a failure would give a reader no way back to what differed'. The delivery goes beyond the criterion in three ways that all bear on vacuity — identical artefacts must report NOTHING (a comparator that reported everything would pass the criterion), equal structure with unequal bytes must still be a difference (so the relation stays byte-equality rather than degrading to a semantic one), and nonVacuous is asserted to actually refuse an empty assembly and an assembly that kept items but lost the order corpus, rather than promising to. The naming is real on live data too: under the map-rebuild mutation of the assembler the ac-1 failure message named items by key, e.g. 'bundle.json.items[0] (itm-0001) .kind'.
  evidence: evals/coldreading_determinism_test.go:609 — "func TestComparatorReportsADifference(t *testing.T) { — measured green; the reorder pair, the one-scalar-inside-an-item pair and a header-scalar pair"
  evidence: evals/coldreading_determinism_test.go:666 — "for _, name := range c.names { if !strings.Contains(report, name) { — MEASURED: gutting elementLabel turns this red naming itm-0002"
  evidence: evals/coldreading_determinism_test.go:623 — "t.Run(\"identical-artefacts-report-nothing\" — the other half, without which a comparator that reported everything would pass"
  evidence: evals/coldreading_determinism_test.go:675 — "t.Run(\"equal-structure-unequal-bytes-is-still-a-difference\" — keeps the relation byte-equality"
  evidence: evals/coldreading_determinism_test.go:685 — "t.Run(\"nonVacuous-refuses-an-empty-assembly\" — asserts the guard refuses, including an assembly that kept items but lost the order corpus"

Gap audit:
- honoured:
  - The eval is WIRED and demonstrably executes: it joins the always-run cold-reading CI lane by carrying `//go:build smoke || coldreading` and needs no Makefile or workflow edit, exactly as the harness documents. I ran the lane on an extracted copy of aa26e07a and it is green.
    evidence: evals/coldreading_determinism_test.go:1 — "//go:build smoke || coldreading"
    evidence: Makefile:58 — "go test -tags coldreading ./evals/... — measured: ok github.com/intentdriven/abcd/evals"
    evidence: .github/workflows/ci.yml:492 — "run: make evals-cold-reading — the always-run job, carrying no `inert` condition"
  - Declared deviation 1 is upward and real: spc-65 specifies identity at one position; the delivered ac-1 runs at all four, and I confirmed the loop is over everyPosition rather than a single one.
    evidence: evals/coldreading_determinism_test.go:50 — "for _, position := range everyPosition {"
    evidence: .abcd/development/specs/closed/spc-65-amnesia-is-a-repository-property-proven-by-an-eval-the-same.md:111 — "**One definition, not four.** ... the eval asserts identity at one position"
  - Declared deviation 2 is upward and has independent reach, which I measured rather than assumed: a manifest-only leak of the repository root fails the absolute-path subtest at all four positions while the byte comparison — which excludes the manifest — sees nothing. That is a leak class no other assertion in the lane covers.
    evidence: evals/coldreading_determinism_test.go:98 — "{bundleFile, side.a.BundleRaw}, {manifestFile, side.a.ManifestRaw} — MEASURED red on a manifest-only path leak, which the bundle byte comparison cannot see"
  - The iss-2608311418202794 scoping is correct. ac-2's text is about item PATHS, the assembler's field order within one path is spc-61's degree of freedom, and spc-65 already assigns the projection there. Confirmed by mutation: a stable field reversal passes both the cold-reading lane and `go test ./internal/core/reading/`.
    evidence: .abcd/development/specs/closed/spc-65-amnesia-is-a-repository-property-proven-by-an-eval-the-same.md:187 — "The assembler, its include table, its projection, and its manifest format: spc-61."
    evidence: .abcd/work/issues/open/iss-2608311418202794-a-stable-reversal-of-the-field-order-within-one-path-changes.md:14 — "This is the assembler's property rather than the amnesia eval's, so it belongs to spc-61 — reproduced: both lanes stay green"
  - The merge-commit body is honest about all three defects the build found — the false falsifier, the manifest gap and the field-order gap — and each carries a captured issue. It also states the substituted mutation (rebuilding the candidate slice from a map), which I independently confirmed does turn ac-1 red.
    evidence: .abcd/work/issues/open/iss-2608311331229368-spc-65-and-itd-187-both-prescribe-watching-testassembledinpu.md:13 — "The record's prescribed hand-run for ac-1 names a mutation that cannot falsify ac-1. — captured in 65baca09, the FIRST commit of the range"
    evidence: .abcd/work/issues/open/iss-2608311331273317-internal-core-reading-manifest-go-documents-that-two-assembl.md:13 — "a nondeterminism confined to the manifest ... is invisible to the amnesia eval"
  - The two nits settled in the ship commit rather than in a fix round are both accurate corrections, not hand-waves. The order corpus's count/comparator prose now matches the six records and four comparators, and the run-identifier exemption's comment now states the any-depth keying and names the closed Manifest struct as what makes it unreachable — which I verified against the struct itself.
    evidence: evals/coldreading_determinism_test.go:379 — "The exemption is keyed on the NAME at any depth, while the shape assertion reads the top-level field only ... The Manifest struct is closed and declares no such field"
    evidence: evals/coldreading_order_test.go:16 — "six records whose names sort one way by byte, another by case-folded comparison, a third by numeric suffix and a fourth by path component"
  - Scope Conditions: itd-187 states 'None stated', and none was invented, dispositioned or quietly added anywhere in the delivery.
    evidence: .abcd/development/intents/shipped/itd-187-amnesia-is-a-repository-property-proven-by-an-eval-the-same.md:35 — "## Scope Conditions  None stated."
- diverged:
  - THE HEADLINE. The acceptance record misdescribes its own falsifier, and shipped that way. itd-187's residue paragraph and spc-65's first Tests bullet both prescribe watching TestAssembledInputIsByteIdenticalAcrossRuns red by removing the assembler's walk sort. I removed it and the test PASSED at all four positions; only TestWalkOrderIsLexicographic fired. A removed sort produces a different order, not an unstable one, so two assemblies of one commit stay byte-identical. The prescribed hand-run would have proved nothing about ac-1 — it would have proved ac-2. The builder found this, captured it as iss-2608311331229368 in the first commit of the range, substituted a mutation that does falsify ac-1, and recorded the substitution in the merge body — but corrected NEITHER shipped record, so the intent and the spec still instruct a future reader to verify ac-1 by a run that cannot verify it. A commit message is not the record.
    evidence: .abcd/development/intents/shipped/itd-187-amnesia-is-a-repository-property-proven-by-an-eval-the-same.md:52 — "What remains is discharged by hand: the walk sort removed by a one-line local patch, the test watched red"
    evidence: .abcd/development/specs/closed/spc-65-amnesia-is-a-repository-property-proven-by-an-eval-the-same.md:167 — "`TestAssembledInputIsByteIdenticalAcrossRuns` is watched red against an assembler whose walk sort is temporarily removed by a one-line local patch, which is the run that proves the comparison can fail against real nondeterminism."
    evidence: internal/core/reading/assemble.go:551 — "sort.SliceStable(out, ... — MEASURED with it deleted: ac-1 PASS at widening/entailment/comparative/detection; only the order oracle red"
  - NOT ON THE DECLARED LIST — the absolute-path guard is a two-string containment check, not an absolute-path detector, and I found a leak it misses. The subtest looks only for the exact strings f.Root and f.Home. Both fixture roots live under one t.TempDir(), so an absolute local path equal to that shared parent rides into the BUNDLE and the entire lane stays green: the byte comparison cannot see it because both runs carry the same string, and the subtest is not looking for it. The run's own output directory is a second uncovered path (it happened to be caught in my probe only incidentally, by the timestamp scan firing on the digits in Go's temp-directory name). Under the repository's rule that no absolute local path enters an artefact, this is the class deviation 2 was added to close.
    evidence: evals/coldreading_determinism_test.go:99 — "{\"its repository root\", side.f.Root}, {\"the HOME it ran under\", side.f.Home} — the only two strings looked for"
    evidence: evals/coldreading_order_test.go:130 — "base := t.TempDir() ... Root: filepath.Join(base, \"first\", \"repo\") — MEASURED: `base` planted in the bundle leaves `go test -tags coldreading ./evals/...` green"
  - NOT ON THE DECLARED LIST — spc-65's stated justification for confining the timestamp scan to the manifest is unsound for paths. packedDigitPattern is an unanchored \d{8,} and item paths are scanned as ordinary scalars, so a manifest item path carrying eight or more consecutive digits is reported as a moment. This repository's own capture ids are nineteen packed digits; I fed the scan such a path and it fired. No path family the include table admits carries one today, so ac-3 is not currently false-positive — but the premise 'a timestamp-shaped token there is unambiguously a defect' is already wrong, and it is the whole argument for the scan's scope.
    evidence: .abcd/development/specs/closed/spc-65-amnesia-is-a-repository-property-proven-by-an-eval-the-same.md:98 — "The manifest carries paths, field names, and hashes only, so a timestamp-shaped token there is unambiguously a defect."
    evidence: evals/coldreading_determinism_test.go:367 — "packedDigitPattern = regexp.MustCompile(`\\d{8,}`) — unanchored; MEASURED firing on a manifest item path holding a real 19-digit capture id"
  - itd-187's press release describes the manifest as carrying 'content hashes and no timestamps'. The shipped assembler's own doc comment says the manifest is not timestamp-free and MUST NOT be described as such, because the run identifier embeds a mint stamp by construction. The eval implements the honest version — a declared, narrow packed-digit exemption for run_id, bounded by a shape assertion — so the code is right and the intent's prose is the thing that is wrong.
    evidence: .abcd/development/intents/shipped/itd-187-amnesia-is-a-repository-property-proven-by-an-eval-the-same.md:20 — "the manifest sits outside the comparison, carries content hashes and no timestamps"
    evidence: internal/core/reading/manifest.go:57 — "it is not timestamp-free and must not be described as such: RunID embeds a mint stamp by construction (adr-45)"
  - manifest.go's shipped prose asserts that two assemblies of one state differ 'in RunID and in nothing else', and this eval asserts no such thing — a per-run manifest content hash passes the entire cold-reading lane, which I confirmed. But iss-2608311331273317 states the gap more widely than it is: that same mutation turns the assembler package's own TestManifestHashIsStableAcrossRuns red ('two runs over one state differ in more than the run identifier'), so the claim IS held somewhere in the repository, just not in the lane itd-187 delivers. What this eval genuinely adds and spc-61's test cannot see is the two-path dimension, since spc-61 compares two runs in one directory.
    evidence: internal/core/reading/manifest.go:59 — "two assemblies of one repository state at one commit produce manifests that differ in RunID and in nothing else"
    evidence: internal/core/reading/manifest_test.go:162 — "t.Error(\"two runs over one state differ in more than the run identifier\") — MEASURED red under a per-run manifest hash, while the cold-reading lane stayed green"
  - spc-65's Wiring section describes a build that did not happen: it specifies `//go:build smoke` alone, says the eval runs under `make smoke` and CI's `smoke` job 'with no change to Makefile or .github/workflows/ci.yml', and then says the always-run workflow edit 'lands with this spec's build'. The delivered files carry `//go:build smoke || coldreading`, and the Makefile target and the always-run CI job landed with spc-64's build, not this one. The outcome is better than the spec (the eval joins an always-run lane with zero edits) and the spec was left describing the plan rather than the delivery.
    evidence: .abcd/development/specs/closed/spc-65-amnesia-is-a-repository-property-proven-by-an-eval-the-same.md:118 — "Go test files in package `evals` with `//go:build smoke`. ... neither needs an edit."
    evidence: .abcd/development/specs/closed/spc-65-amnesia-is-a-repository-property-proven-by-an-eval-the-same.md:138 — "The workflow edit lands with this spec's build, reviewed alongside the evals it serves."
    evidence: evals/coldreading_order_test.go:1 — "//go:build smoke || coldreading — and `git diff 446c607f..f462b92f` touches neither Makefile nor ci.yml"
  - NOT ON THE DECLARED LIST — the ship commit's run-identifier correction landed on one surface of two. evals/README.md still says the exemption's 'own shape is asserted', the exact wording the ship commit replaced in the Go comment because the shape assertion reads the top-level field while the exemption is keyed on the name at any depth. Not false for the real artefact, but it is the uncorrected half of a correction the same commit made.
    evidence: evals/README.md:101 — "it is exempt from the packed-digit rule alone and its own shape is asserted"
    evidence: evals/coldreading_determinism_test.go:379 — "the shape assertion reads the top-level field only, so a nested key spelled run_id would be exempt and unchecked — the corrected wording, applied to the Go comment only"
  - NOT ON THE DECLARED LIST — the 'one commit' premise is asserted over the ROOT commit, not over the target. materialiseOrderPair compares `git rev-list --max-parents=0 HEAD` between the two trees and reports that the assemblies 'must run over ONE commit', while the assemblies target HEAD. The fixture is committed exactly once, so root and HEAD coincide and the check is currently equivalent to what it claims; the message claims more than the assertion checks.
    evidence: evals/coldreading_order_test.go:161 — "if sha := rootCommit(t, second.Root); sha != first.RootSHA { ... the two assemblies must run over ONE commit"
    evidence: evals/coldreading_fixture_test.go:552 — "sha := gitFixture(t, root, \"rev-list\", \"--max-parents=0\", \"HEAD\") — the root commit, not HEAD"
  - spc-65's Approach still describes the order corpus as separating three comparators (byte, case-insensitive, numeric-suffix). The delivered corpus separates four; the fourth, path-component order, is the one f462b92f added after review found the corpus could not distinguish byte order from directory-walk order — and it is the likeliest wrong order, since it is what dropping the sort yields. The Go comments and evals/README.md were both updated; the spec was not.
    evidence: .abcd/development/specs/closed/spc-65-amnesia-is-a-repository-property-proven-by-an-eval-the-same.md:89 — "a set of records whose names sort differently under byte order, case-insensitive order, and numeric-suffix order"
    evidence: evals/README.md:110 — "six records whose names sort one way by byte, another by case-folded comparison, a third by numeric suffix and a fourth by path component"
- missing:
  - The hand-run itd-187's residue paragraph promises was not delivered as promised, and could not have been. The promise is specific — the walk sort removed by a one-line local patch, the test watched red, the patch reverted, the run recorded in the pull-request body — and the test named cannot go red that way. What was actually done is a different mutation, recorded only in the merge-commit body. So the residue's discharge exists, but not the discharge the record describes, and the record was never brought into line with it.
    evidence: .abcd/development/intents/shipped/itd-187-amnesia-is-a-repository-property-proven-by-an-eval-the-same.md:52 — "the patch reverted before the branch is pushed, and the run recorded in the pull-request body. That is a recorded hand-run, not a standing gate."
    evidence: .abcd/work/issues/open/iss-2608311331229368-spc-65-and-itd-187-both-prescribe-watching-testassembledinpu.md:13 — "Run on a copy of the tree, that mutation does not make it red — OPEN; neither record was corrected"
  - No assertion anywhere in this eval compares the two manifests modulo run_id, which is the cheap closure iss-2608311331273317 itself names. The manifest is held to two weaker properties and otherwise excluded, so within itd-187's own delivery a manifest-only nondeterminism is invisible; the backstop is spc-61's package test, which is a different lane, runs in one directory, and is not what itd-187 promised.
    evidence: evals/coldreading_determinism_test.go:113 — "compareArtefacts(bundleFile, a.BundleRaw, b.BundleRaw) — the only comparison; ManifestRaw is never compared between the two runs"
    evidence: .abcd/work/issues/open/iss-2608311331273317-internal-core-reading-manifest-go-documents-that-two-assembl.md:13 — "The cheap closure is a manifest comparison modulo run_id in the same eval. — OPEN"