---
id: itd-198
slug: an-assembly-reports-what-it-would-cost-before-a-reading-is
spec_id: spc-68
kind: standalone
suggested_kind: standalone
reclassification_history: []
builds_on: [itd-183]
severity: critical
impact: fix
origin: researcher-authored
production_mode: hand-written
---

# An assembly reports what it would cost before a reading is commissioned

## Press Release

> **The assembler says how large a reading would be, per material kind, before anyone dispatches it.** Every assembly reports bytes and an estimated token count for each kind of material it passed, and the total, whether or not it writes an artefact. Test files are reported apart from other source, because on a repository of any size they are the largest single class and nobody had counted them. No budget is enforced and none is invented: the assembler cannot know what a given reader accepts, and a number it made up would be wrong for someone. It reports; the operator decides.

> "I assembled a reading and got nine point eight megabytes," said an AI/agent researcher who runs cold readings against their own design record. "Every test passed. Every criterion was met. The artefact could not be handed to anything, and there was no way to learn that except by trying."

## Why This Matters

The instrument shipped correct and undeliverable, and the gap that let it was the absence of a number. Measured over this repository at one commit: about 9.8 MB of artefact, roughly 2.3 million estimated tokens of item text, at every position
([iss-2608311501186646](../../../work/issues/resolved/iss-2608311501186646-the-assembled-input-for-a-real-reading-of-this-repository-is.md)).
Source is 82 per cent of it and test files are 53 per cent of the source; the records a reading is meant to reason about are 9 per cent. Thirteen intents, six adversarial delta reviews and five fidelity audits passed over that, because the evals run against a fixture repository of about thirty files, which is the right corpus for asserting a firewall and the wrong one for learning what an artefact weighs.

This intent adds the number, and the four mechanism changes it needs to be
checkable rather than asserted. It does not make the reading fit — the measured selections are 2.3 million tokens for everything, 1.3 million without tests, and 210 thousand for the records alone, so no selection this intent could offer both fits a reader and preserves any position's stated object. Making a reading fit is [itd-199](itd-199-a-reading-is-about-something-narrower-than-everything-its.md)'s work, and it is a redesign. The number comes first because it is cheap, it is what made the problem visible, and every later decision about what a reading should contain is a decision nobody can take without it.

## What's In Scope

- **A per-kind size report on every assembly**, whether or not an artefact is written, reachable through the existing dry-run path that already renders a result and writes nothing.
- **Bytes and an estimated token count** per material kind and in total, with the estimate labelled as a byte-derived estimate rather than a tokenizer's answer.
- **A `test` material kind**, split from `source`, which requires a suffix form the include table's match grammar does not have: the grammar today reads an entry beginning with a dot as an extension and anything else as an exact basename, and `_test.go` is neither. **The suffix form is carried by its own row field rather than by a third convention inside the existing match list** (ruled, maintainer 2026-08-31), so no disambiguation rule against the two existing forms is needed and none is written: a form named by the field it sits in cannot be confused with a form inferred from a string's first character. The match is **case-sensitive**, because the Go toolchain recognises only a lowercase `_test.go` as a test file, and a report that called something a test which Go does not build as one would disagree with the thing it counts. The two existing forms disagree with each other on case for no stated reason; that asymmetry predates this intent, is not resolved by it, and is captured as [iss-2608311949421873](../../../work/issues/open/iss-2608311949421873-the-include-table-match-grammar-disagrees-with-itself-on-cas.md) so the fourth form does not rediscover it.
- **An assembler version bump — both versions move, and the intent says which and why.** `AssemblerVersion` moves because the include table's rendering changes twice over: a row is added, and the kind column joins the rendering. `SchemaVersion` moves from 1 to 2 because `ManifestItem` gains a field. `SchemaVersion` is **one constant shared by both artefacts an assembly writes**, so bumping it restamps the bundle as well, even though ac-8 holds the bundle's shape unchanged. That is a known consequence of the shared constant and is accepted here rather than fixed: splitting the two shape versions is a larger change than this intent, and it is not made silently by a change that only needed one of them.
- **The kind column added to the include table's rendering**, which fixes a LATENT defect rather than one this split creates. The rendering emits positions, source, matches, fields and the admitting rule, and no kind, so today a kind reassignment on an existing row changes every bundle while the version the manifests carry stands still. That is true before this intent and is closed by it.
- **The kind recorded per manifest item**, so the report is checkable against the manifest rather than asserted beside it. Brief invariant 16 requires an attestation to state no more than its examination establishes, and a report the manifest cannot corroborate is exactly that shape.
- **The version pin made mechanical rather than advisory**, which closes a second latent hole found while reading the gate this intent has to move. The gate compares the rendered table's digest to a standalone literal and never reads `AssemblerVersion` at all, so changing the table and updating only the literal is green with the version standing still: the gate asks a human to move the version, it does not make them ([iss-2608311949385350](../../../work/issues/resolved/iss-2608311949385350-the-assembler-version-pin-is-advisory-not-mechanical-testass.md)). That is the same shape as the hole in the bullet above, one layer out — an attestation whose examination cannot establish what it asserts, which brief invariant 16 forbids — and closing one while leaving the other would leave the version's own claim resting on a convention.

## What's Out of Scope

- **Making a reading fit.** No selection, budget or refusal. itd-199 carries that.
- **Enforcing a budget.** The assembler cannot know a reader's capacity, so a threshold here would be a guess with a gate attached.
- **A tokenizer.** The estimate is bytes divided by a constant, and says so.
- **Reclassifying test-support packages.** Helper packages that are not `_test.go` files stay `source`; measured, they are three items and about 2,600 estimated tokens, which is not worth a second rule.

## Scope Conditions

- The byte-derived estimate is accurate enough to decide whether something is plausible, not to plan against. If it ever changes a decision it should not have, it is replaced by a real tokenizer rather than tuned. **The divisor is fixed during the spec build by sampling this repository's material through a real tokenizer, and the sample is recorded in the spec** (ruled, maintainer 2026-08-31), so the one constant the estimate rests on is measured rather than assumed. A constant chosen from evidence cannot later be accused of the tuning the paragraph above forbids, and a constant back-fitted to the figures already written into this record would have read as a precision the method does not have. <!-- cond: cond-2608311949582375 -->
- Reporting test files apart from source does not narrow any reading. The detection position's own definition names the tests as part of its object: "The shipped tree read against the claim record: the source, the tests, the delivered documentation and the build configuration on one side, and the shipped intents, the specs, the disciplines, the glossary and the brief's current text on the other." So this intent separates them for counting and never for admission, and ac-4 below binds that rather than leaving it to prose. <!-- cond: cond-2608311949586552 -->
- The kind carried on a bundle item is a field a reading receives, so reassigning 407 of this repository's items from `source` to `test` DOES change the bundle's bytes. That is the whole of the bundle change, and ac-6 states it rather than claiming the bundle is untouched. <!-- cond: cond-2608311949589926 -->
- This intent assumes Go source stays admitted. itd-194 narrows admission to what the exclusion floor can parse and states its parseable set as markdown whose frontmatter it resolves; every percentage here rests on source remaining in the bundle, and if itd-194 lands first that assumption is the one to re-measure. <!-- cond: cond-2608311949589261 -->

## Acceptance Criteria

- **Given** an assembly, **when** its result is produced, **then** it carries bytes and an estimated token count for each material kind it passed, and a total.
- **Given** a dry run that writes no artefact, **when** its result is rendered on the CLI and reported through the plugin page, **then** the per-kind figures and the total appear on both surfaces.
- **Given** a report, **when** its token figures are read, **then** each is labelled as a byte-derived estimate rather than a tokenizer's count.
- **Given** each of the four positions, **when** an assembly runs before and after the kind split, **then** the set of admitted repository paths is identical at every position.
- **Given** a repository holding files whose basenames end in `_test.go`, matched case-sensitively as the Go toolchain matches it, **when** an assembly admits them, **then** those items carry the `test` kind, every other admitted Go file carries `source`, and a file whose basename ends in the suffix in any other case carries `source`.
- **Given** the include table's rendering, **when** the assembler version is derived from it, **then** the rendering includes each row's kind, so a kind reassignment on an existing row moves the version.
- **Given** a manifest, **when** it is decoded strictly, **then** every item carries a kind and round-trips it.
- **Given** an assembled bundle, **when** it is decoded, **then** it carries no size figure and no field the report introduced: the only bundle change this intent makes is the kind label on items reassigned to `test`.
- **Given** any change to the include table, **when** a manifest is written before and after it, **then** the assembler version the two manifests stamp differs — the version is derived from the table's rendering, so a manifest naming a version that does not describe its own table cannot be produced at all, rather than being caught by a gate after the fact.

## Grounds

- pursued: This conjecture is pursued now because the absence of a number is what let an unusable artefact pass every gate the workstream built, and the number is cheap where the fix is a redesign. What would show it wrong is a reader that accepts the current artefact, which would make the count uninteresting, or a token estimate so far from a tokenizer's answer that it misleads the decision it exists to inform.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-14b2ab720552 -->
Fidelity review — receipt rcp-14b2ab720552 (verifier intent-auditor claude-opus-5[1m]).

Provenance: intent-auditor@claude-opus-5[1m] · rubric_hash sha256:542ed2cd51ff938717a3f47b2b332e8d47910beec0ca7ecdfd238ae7edf5ced5 · prompt_hash sha256:154e5b0eaa502a3fed90b6bd89a4cfdbc0715671b00c2cb2234c3797340d6a9d
Input attestations: diff:0e0743e8..2f3106d25233a8b5c84966e1a7676b32bd960fdc@sha256:0798c660e101ff2da58139629c9f474be903c8b227efa033162a69a79feca25a; intent:.abcd/development/intents/shipped/itd-198-an-assembly-reports-what-it-would-cost-before-a-reading-is.md@-; spec:.abcd/development/specs/closed/spc-68-an-assembly-reports-what-it-would-cost-before-a-reading-is.md@-; recomputation:.abcd/development/readings/README.md@sha256:4ec7fdecea43c78746f7e306a9b123a7bf2dcd981f82d1567c8328a5b628de70;

Acceptance rollup: MET 5 · MET_WITH_CONCERNS 4 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: sizeReport totals every collected candidate into a per-kind row carrying Items/Bytes/TokensEst plus a report-level total, and the report rides on AssembleResult.Size on every assembly; the per-kind rows are proven to sum to the totals and to omit no kind that passed an item.
  evidence: internal/core/reading/assemble.go:111 — "type SizeReport struct { ByKind []KindSize; Items int; Bytes int; TokensEst int; Basis string }"
  evidence: internal/core/reading/assemble.go:130 — "func sizeReport(cands []candidate) SizeReport"
  evidence: internal/core/reading/assemble.go:341 — "Size:             sizeReport(cands),"
  evidence: internal/core/reading/size_test.go:112 — "func TestSizeReportSumsToTotal"
  evidence: internal/core/reading/size_test.go:148 — "func TestSizeReportOmitsKindsThatPassedNothing"
- ac-2 — MET: A dry run carries a populated report from the core, renderSizeReport is called from renderAssembleResult ahead of the dry-run early return so a run that writes nothing still prints the per-kind rows and the total, and commands/reading.md instructs the host to report size/by_kind/tokens_est on every run including a dry run, with a test asserting the page says so.
  evidence: internal/core/reading/size_test.go:199 — "func TestDryRunCarriesTheSizeReport"
  evidence: internal/surface/cli/reading.go:267 — "renderSizeReport(w, res.Size)"
  evidence: internal/surface/cli/reading.go:137 — "renderAssembleResult(w, res)"
  evidence: internal/surface/cli/reading_surface_test.go:693 — "func TestSizeReportRendersBeforeTheWrittenLine"
  evidence: commands/reading.md:84 — "Also report `size`, on every run including a dry run: the total `bytes` and `tokens_est`, and each row of `by_kind`"
  evidence: internal/surface/cli/reading_surface_test.go:738 — "func TestPluginPageReportsTheSizeAndScope"
- ac-3 — MET_WITH_CONCERNS: The basis string names the method and the divisor and travels in the artefact rather than only in the rendering, and every figure is a tokens_est field governed by it; CONCERN — the label is report-level, not per-figure: SizeReport carries ONE Basis for the whole report and the CLI prints it once on the total line, so each per-kind row renders as a bare tilde-prefixed count. That is a signed-off narrowing (spc-68 states 'with the basis stated once'), not an accident, but it is narrower than ac-3's 'each is labelled'.
  evidence: internal/core/reading/assemble.go:124 — "var sizeBasis = fmt.Sprintf(\"estimated: bytes / %.2f, byte-derived, not a tokenizer's count\", tokenBytesPerToken)"
  evidence: internal/core/reading/size_test.go:221 — "func TestSizeReportLabelsItsEstimate"
  evidence: internal/surface/cli/reading.go:313 — "fmt.Fprintf(w, \"  size (item text): %s, ~%s tokens (%s)\\n\", humanBytes(s.Bytes), thousands(s.TokensEst), s.Basis)"
  evidence: internal/surface/cli/reading.go:316 — "for _, k := range s.ByKind { fmt.Fprintf(w, \"    %-18s %6d item(s)  %9s  ~%s tokens\\n\", ...) }"
  evidence: .abcd/development/specs/closed/spc-68-an-assembly-reports-what-it-would-cost-before-a-reading-is.md:305 — "one row per kind that passed plus a total, with the basis stated once"
- ac-4 — MET_WITH_CONCERNS: The invariant is proved by MUTATION rather than assertion — the test row is removed from the include table and the admitted path set must be byte-identical, with a companion test refusing vacuity by requiring a test-kind item at every position exercised; CONCERN — the loop runs over AssemblingPositions(), which is Positions() MINUS comparative, so the proof covers THREE of the four positions ac-4 names. Comparative refuses assembly outright (a change delivered by itd-199 inside this same range), so the fourth position is not merely unproven but unprovable as written.
  evidence: internal/core/reading/size_test.go:50 — "func TestKindSplitDoesNotMoveAdmission"
  evidence: internal/core/reading/size_test.go:76 — "for _, p := range AssemblingPositions() {"
  evidence: internal/core/reading/size_test.go:88 — "func TestTestKindIsReachableAtEveryPosition"
  evidence: internal/core/reading/include.go:84 — "func AssemblingPositions() []Position { ... if p == PositionComparative { continue }"
  evidence: internal/core/reading/assemble.go:238 — "if position == PositionComparative {"
- ac-5 — MET: MatchSuffix is matched with strings.HasSuffix on the basename with no case folding, deliberately unlike the Match extension form's EqualFold beside it, and the test row is ordered above the .go row so it owns the path first; the fixture pins all three outcomes — widget_test.go carries test, widget.go carries source, and Gadget_TEST.go (the suffix in another case) carries source.
  evidence: internal/core/reading/deny.go:91 — "for _, s := range r.MatchSuffix { if strings.HasSuffix(base, s) {"
  evidence: internal/core/reading/deny.go:99 — "if strings.EqualFold(ext, m) {"
  evidence: internal/core/reading/include.go:309 — "MatchSuffix: []string{\"_test.go\"}, Kind: KindTest,"
  evidence: internal/core/reading/size_test.go:13 — "func TestTestFilesCarryTheTestKind"
  evidence: internal/core/reading/size_test.go:17 — "writeFile(t, root, \"Gadget_TEST.go\", ...)  // want KindSource"
- ac-6 — MET: Render() emits a Kind column (and a Suffixes column) for every row, AssemblerVersion() is the sha256 of exactly that rendering, and the coverage is proved by mutation rather than by assertion. Independently recomputed: the rendered table committed in the readings charter — which visibly carries the Kind column — hashes to 4ec7fdec…, the literal the digest pin holds, so the rendering that is digested demonstrably includes the kind.
  evidence: internal/core/reading/include.go:430 — "| Positions | Source | Matches | Suffixes | Fields | Store | Bucket | Kind | Admitting rule |"
  evidence: internal/core/reading/include.go:441 — "row.Kind,"
  evidence: internal/core/reading/include.go:47 — "sum := sha256.Sum256([]byte(Render()))"
  evidence: internal/core/reading/size_test.go:333 — "func TestRenderCoversKindAndSuffix"
  evidence: .abcd/development/readings/README.md:0 — "recomputed sha256 of the charter's rendered table = 4ec7fdecea43c78746f7e306a9b123a7bf2dcd981f82d1567c8328a5b628de70, equal to includeTableDigest"
- ac-7 — MET: ManifestItem.Kind is a non-omitempty field populated on every item at assembly time, DecodeManifest is strict (DisallowUnknownFields plus a trailing-content refusal), and three mutations — empty kind, absent kind, out-of-vocabulary kind — are each proved to be refused, with the round-trip asserted item by item.
  evidence: internal/core/reading/manifest.go:115 — "Kind Kind `json:\"kind\"`   // deliberately NOT omitempty"
  evidence: internal/core/reading/assemble.go:322 — "ItemKey: key, Path: c.path, Field: c.field, Kind: c.kind,"
  evidence: internal/core/reading/manifest.go:196 — "dec.DisallowUnknownFields()"
  evidence: internal/core/reading/size_test.go:285 — "func TestManifestItemRoundTripsKind"
  evidence: internal/core/reading/size_test.go:443 — "func TestDecodeManifestRefusesAnItemWithoutAKind"
- ac-8 — MET_WITH_CONCERNS: The first clause is realised and enforced structurally: the bundle carries no size figure, BundleItem is exactly item_key/kind/text, and the guard decodes the artefact and refuses any top-level or per-item key outside a closed allow-set (a shape check, not a substring scan). CONCERN — the second clause, 'the only bundle change this intent makes is the kind label', is not verifiable against the delivered artefact: the bundle ALSO gained a scope block and its SchemaVersion is 4, not the 1-to-2 move itd-198's What's In Scope promises. The test itself whitelists "scope" and attributes it to itd-199, which shipped inside this same range; the attribution is honest and itd-198's own contribution is clean, but the criterion as written cannot be separated from co-delivery by reading the artefact.
  evidence: internal/core/reading/manifest.go:35 — "type BundleItem struct { ItemKey string; Kind Kind; Text string }"
  evidence: internal/core/reading/size_test.go:241 — "func TestBundleGainsNoFieldFromTheReport"
  evidence: internal/core/reading/size_test.go:267 — "want := map[string]bool{\"_type\": true, \"schema_version\": true, \"position\": true, \"scope\": true, \"items\": true}"
  evidence: internal/core/reading/manifest.go:23 — "const SchemaVersion = 4"
  evidence: internal/core/reading/manifest.go:62 — "Scope BundleScope `json:\"scope\"`"
- ac-9 — MET_WITH_CONCERNS: The pin is mechanical, not advisory: AssemblerVersion() is the core semver composed with the WHOLE sha256 of Render(), and a manifest is stamped by calling that function at build time, so a manifest naming a version that does not describe its own table cannot be constructed; the mutation test performs a table change and demands the stamped version move, and the advisory literal pin was restored alongside as a second, human-facing signal (verified current by recomputation). CONCERN — the criterion's absolute ('cannot be produced at all, rather than being caught by a gate after the fact') still rests on a gate for one channel: Render() flattens the table into an UNESCAPED pipe-delimited markdown table over free-prose fields, so two structurally different tables can render, and therefore stamp, identically. TestRenderCannotForgeARowBoundary refuses pipes and newlines in the CURRENT table's field values — which is exactly a gate catching it after the fact, not a structural impossibility. The channel is closed in practice and open in principle.
  evidence: internal/core/reading/include.go:47 — "func AssemblerVersion() string { sum := sha256.Sum256([]byte(Render())); return AssemblerVersionCore + \"+\" + hex.EncodeToString(sum[:]) }"
  evidence: internal/core/reading/assemble.go:311 — "AssemblerVersion: AssemblerVersion(),"
  evidence: internal/core/reading/include_test.go:284 — "func TestATableChangeMovesTheStampedVersion"
  evidence: internal/core/reading/include_test.go:245 — "const includeTableDigest = \"4ec7fdecea43c78746f7e306a9b123a7bf2dcd981f82d1567c8328a5b628de70\""
  evidence: internal/core/reading/size_test.go:554 — "func TestRenderCannotForgeARowBoundary — iterates the current Table refusing values containing |, \\n, \\r"
  evidence: internal/core/reading/include.go:435 — "fmt.Fprintf(&b, \"| %s | `%s` | %s | %s | %s | %s | %s | `%s` | %s |\\n\", ... row.Rule)"

Gap audit:
- honoured:
  - A per-kind size report on every assembly, whether or not an artefact is written, reachable through the existing dry-run path.
    evidence: internal/core/reading/assemble.go:341 — "Size:             sizeReport(cands),"
    evidence: internal/core/reading/size_test.go:199 — "func TestDryRunCarriesTheSizeReport"
  - The suffix form is carried by its own row field rather than by a third convention inside the existing match list, so no disambiguation rule against the two existing forms is needed and none is written.
    evidence: internal/core/reading/include.go:167 — "MatchSuffix []string"
    evidence: internal/core/reading/deny.go:80 — "an empty Match beside a non-empty MatchSuffix means the extension/basename form contributes nothing"
  - The kind column added to the include table's rendering, closing the latent defect where a kind reassignment changed every bundle while the stamped version stood still.
    evidence: internal/core/reading/include.go:441 — "row.Kind,"
    evidence: internal/core/reading/size_test.go:333 — "func TestRenderCoversKindAndSuffix"
  - The version pin made mechanical rather than advisory (iss-2608311949385350), with the digest carried WHOLE rather than truncated so the absolute claim is not weakened by a grindable collision.
    evidence: internal/core/reading/include.go:40 — "The digest is carried WHOLE, not truncated."
    evidence: internal/core/reading/include_test.go:284 — "func TestATableChangeMovesTheStampedVersion"
  - The report is checkable against the manifest rather than asserted beside it — brief invariant 16, an attestation stating no more than its examination establishes.
    evidence: internal/core/reading/manifest.go:124 — "Bytes  int    `json:\"bytes\"`"
    evidence: internal/core/reading/size_test.go:499 — "func TestTheSizeReportIsCheckableAgainstTheManifest — rebuilds every per-kind row, the totals and the token estimate from the manifest alone and demands they match, then checks each item's bytes against the text the bundle actually carries"
  - The divisor is fixed by sampling this repository's material through a real tokenizer, and the sample is recorded in the spec rather than back-fitted.
    evidence: internal/core/reading/assemble.go:93 — "const tokenBytesPerToken = 3.85"
    evidence: .abcd/development/specs/closed/spc-68-an-assembly-reports-what-it-would-cost-before-a-reading-is.md:147 — "through `tiktoken` 0.14.0 ... 2,575 files, 17,119,789 bytes ... total 3.865"
  - Store and Bucket are rendered — the reasoning for leaving them out did not survive examination, since rowPaths filters on Bucket and selects a node type by Store.
    evidence: internal/core/reading/include.go:430 — "| Positions | Source | Matches | Suffixes | Fields | Store | Bucket | Kind | Admitting rule |"
    evidence: internal/core/reading/include.go:466 — "func routeField(s string) string"
- diverged:
  - 'SchemaVersion moves from 1 to 2 because ManifestItem gains a field' — the delivered constant is 4, moved twice more by the scope block and by ManifestItem.Bytes.
    evidence: internal/core/reading/manifest.go:23 — "const SchemaVersion = 4"
    evidence: internal/core/reading/manifest.go:14 — "At version 4 the manifest item gained its byte length, so the shared constant restamps the bundle again."
  - ac-4's 'each of the four positions' — delivered as three, because comparative no longer assembles at all.
    evidence: internal/core/reading/include.go:84 — "func AssemblingPositions() // Positions() minus comparative"
    evidence: internal/core/reading/size_test.go:76 — "for _, p := range AssemblingPositions() {"
  - ac-8's 'the only bundle change this intent makes' — the delivered bundle also gained a scope block, delivered by itd-199 in the same range and whitelisted by name in itd-198's own guard.
    evidence: internal/core/reading/size_test.go:267 — "\"scope\": true"
    evidence: internal/core/reading/manifest.go:62 — "Scope BundleScope `json:\"scope\"`"
  - spc-68's illustrative CLI snippet still shows 'size:' and a basis without 'byte-derived'; the shipped render is 'size (item text):' and the shipped basis carries 'byte-derived'. The substantive snippet-versus-prose contradiction on what Bytes counts is corrected; this residue is cosmetic.
    evidence: .abcd/development/specs/closed/spc-68-an-assembly-reports-what-it-would-cost-before-a-reading-is.md:310 — "size:          9.8 MB, ~2,295,107 tokens (estimated: bytes / 3.85, not a tokenizer's count)"
    evidence: internal/surface/cli/reading.go:313 — "\"  size (item text): %s, ~%s tokens (%s)\\n\""
- missing:
  - A structural guarantee that the rendering AssemblerVersion digests is unambiguous. The escaping of the digested table is held by a test over the current table's values, not by the renderer, so ac-9's 'cannot be produced at all, rather than being caught by a gate after the fact' is honoured for truncation and delegated to a gate for row-boundary forgery.
    evidence: internal/core/reading/include.go:435 — "fmt.Fprintf(&b, \"| %s | `%s` | %s | %s | %s | %s | %s | `%s` | %s |\\n\", ...) — no escaping of the free-prose fields"
    evidence: internal/core/reading/size_test.go:554 — "func TestRenderCannotForgeARowBoundary — a gate over Table's current values"
  - An end-to-end CLI proof of ac-2: the dry-run render is asserted against a hand-built AssembleResult rather than against a real assembly driven through the command, so the two halves are joined by composition rather than by one executed path.
    evidence: internal/surface/cli/reading_surface_test.go:663 — "func TestDryRunRendersTheSizeReport — res := reading.AssembleResult{ ... } (a fixture, not an assembly)"
    evidence: internal/surface/cli/reading.go:137 — "renderAssembleResult(w, res)"

Scope-condition dispositions:
- cond-2608311949582375 — survived: The divisor was fixed by sampling this repository's material through a real tokenizer during the spec build and the sample is recorded in spc-68 — 2,575 files, 17,119,789 bytes, cl100k_base and o200k_base agreeing within 0.3 per cent, byte-weighted 3.865 rounded to 3.85 — so the constant rests on evidence rather than back-fitting, and the known per-kind bias is disclosed rather than tuned away.
  evidence: .abcd/development/specs/closed/spc-68-an-assembly-reports-what-it-would-cost-before-a-reading-is.md:147 — "through `tiktoken` 0.14.0 in a throwaway virtual environment outside the repository, under both `cl100k_base` and `o200k_base`. 2,575 files, 17,119,789 bytes."
  evidence: internal/core/reading/assemble.go:79 — "the divisor of the byte-derived token estimate, measured rather than assumed"
  evidence: .abcd/development/specs/closed/spc-68-an-assembly-reports-what-it-would-cost-before-a-reading-is.md:171 — "A single divisor mis-states each kind, in known directions."
- cond-2608311949586552 — survived: The test row labels and never admits: it is ordered above the .go row solely so path.Ext does not let source claim the file first, and the condition is bound mechanically rather than left to prose — removing the row entirely leaves the admitted path set byte-identical at every assembling position, with a companion test refusing the vacuous case where no test file was admitted at all.
  evidence: internal/core/reading/include.go:302 — "Both rows admit — this row narrows nothing and widens nothing, it only labels."
  evidence: internal/core/reading/size_test.go:50 — "func TestKindSplitDoesNotMoveAdmission"
  evidence: internal/core/reading/size_test.go:88 — "func TestTestKindIsReachableAtEveryPosition"
- cond-2608311949589926 — survived: The kind is carried on the bundle item as the condition anticipated, so reassigning items from source to test does change the bundle's bytes; the guard on ac-8 holds the bundle's key set closed around exactly item_key/kind/text, confirming the kind label is the whole of the item-level change this intent makes.
  evidence: internal/core/reading/manifest.go:35 — "type BundleItem struct { ItemKey string; Kind Kind; Text string }"
  evidence: internal/core/reading/size_test.go:271 — "wantItem := map[string]bool{\"item_key\": true, \"kind\": true, \"text\": true}"
- cond-2608311949589261 — survived: The assumption held over the delivery: itd-194 has not landed — it remains in the drafts bucket — and the include table still admits Go source under its own row, so no percentage in this intent rests on a narrowing that occurred.
  evidence: .abcd/development/intents/drafts/itd-194-the-reading-include-table-admits-only-what-the-exclusion-flo.md:1 — "itd-194 is in the drafts bucket, not shipped"
  evidence: internal/core/reading/include.go:318 — "Match: []string{\".go\"}, Kind: KindSource,"