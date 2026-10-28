---
id: itd-2610141646454347
slug: an-ideate-verdict-proves-its-own-legs
spec_id: spc-2610142138047122
kind: standalone
bundle: null
suggested_kind: null
reclassification_history:
  - { date: 2026-10-14, from: bundle-member, to: standalone, reason: "The product thinker ruled in the planning interview that this fix ships as soon as it is ready, outside bundle ideate-librarian: it needs nothing from the librarian, and its major issue blocks every release cut while it is open." }
builds_on: []
related_issues: [iss-2610141647304982, iss-2610141647307807, iss-2610141647308358, iss-2610141647315880, iss-2610141647313018]
severity: major
impact: breaking
origin: researcher-authored
production_mode: hand-written
---

# An ideate verdict proves its own legs

## Press Release

> **An ideate verdict proves every leg it records, not just the record grill.**
> Every record id in every leg resolves, and a primary source must be a web
> address, a filed source or a record. The grill says what it searched and
> where, so an empty grill can be told from a broken search. A reframing has its
> own field. Two reviewers are recorded reviewer by reviewer, and a split
> between them carries the ruling that settled it. A claim checked only against
> the idea's own source reads "quoted", never "verified".
>
> "I read a verdict that killed an idea," said Bob, a staff engineer. "I could
> see which searches came back empty, that both reviewers agreed, and that the
> claim it leaned on was only checked as quoted. I trusted the kill because I
> could check it."

## Why This Matters

`abcd ideate record` checks citations only in the grill leg. A lab hand-run of
2026-10-14 recorded, with exit 0, a verdict that cited a non-existent intent in
leg 3 and another in leg 1 and named "a remark heard in passing" as a verified
primary source (iss-2610141647304982). The same lab found that a broken grill
and an empty one write the same record (iss-2610141647307807); that two
reviewers and a disagreement between them have no place in the record
(iss-2610141647308358, iss-2610141647315880); and that "verified" cannot say
"verified as quoted" (iss-2610141647313018). Each is a missing field or a
missing check, and the binary already checks the grill leg's ids, so the same
kind of check extends to the rest.

Breaking, for anything that records a verdict: a payload with an unresolved id
in any leg, a free-text primary source, a reframed verdict without its
reframing, or schema version 1 is refused where it was accepted.

## What's In Scope

- Schema version 2 of the ideate verdict, its validation, and its rendering.
- The command page `commands/ideate.md` updated to the new shape in the same
  change.

## What's Out of Scope

- Private verdicts and citing them (itd-2610141646456128).
- Ideate starting from a source, and citing pack items (itd-2610141646452213).
- Judging any claim: the binary checks shapes and citations, never the truth of
  a claim.

## Mechanism

We expect no verdict recorded after this change to carry an unresolved id, an
unlabelled self-quotation or an unrecorded split, because each is refused at
record time by a check on the payload's shape and citations. Shown wrong if a
version 2 verdict on record carries an id that does not resolve, or a claim
marked verified whose only primary source is one of the idea's own origin
sources.

## Scope Conditions

- The writer accepts version 2 only; a version 1 payload is refused with a <!-- cond: cond-2610142138048878 -->
  message naming version 2. Verdicts already on record are rendered Markdown
  and are not parsed or backfilled.
- The command page and the binary ship together; a host reading an older page <!-- cond: cond-2610142138047607 -->
  gets the version-1 refusal message.
- The binary checks shapes and citations. Whether a claim is true, or how a <!-- cond: cond-2610142138040936 -->
  reviewer reached a verdict, is the host's and the person's judgement.

## Acceptance Criteria

- **Given** a verdict citing a record id that does not resolve, in any leg,
  **when** it is recorded, **then** it is refused, naming the leg and the id.
- **Given** a claim whose primary source is not a web address, a filed source
  key or a record id, **when** the verdict is recorded, **then** it is refused.
- **Given** a grill leg with no hits and no search record (the terms searched,
  the places searched, and the count each returned), **when** the verdict is
  recorded, **then** it is refused.
- **Given** a verdict marked reframed with no reframing field, **when** it is
  recorded, **then** it is refused.
- **Given** two reviewers, **when** the verdict is recorded, **then** each
  reviewer's kill attempts, verdict and consulted sources are kept under that
  reviewer.
- **Given** two reviewers whose verdicts differ, **when** the verdict is
  recorded without a ruling (who ruled, and the ruling), **then** it is
  refused.
- **Given** a claim marked verified whose only primary source is listed among
  the verdict's origin sources, **when** it is recorded, **then** it is
  refused; marked quoted, it is accepted.
- **Given** a version 1 payload, **when** it is recorded, **then** it is
  refused with a message naming version 2.
- **Given** the updated command page, **when** it is read, **then** it tells the
  runner to give each reviewer one idea per context.

## Prior Art

- itd-104 (shipped): the ideate admission gauntlet and `ideate record`, whose
  grill-leg citation check this extends to every leg.
- itd-1 (discipline): every audit verifies acceptance criteria; the verdict's
  own legs get the same treatment.
- iss-2608230748418054 (open): the grill's structural blind spot over the
  research notes, which the search record makes visible.

## SOTA

- Structured-output validation of model-produced records (JSON Schema
  validators, Pydantic-style models): mature; validates shape, not truth.
- Citation resolution in research tooling (DOI and URL resolvers): mature for
  external references; abcd's record ids need abcd's own resolver.

**Path:** 2. A native floor: abcd's own resolver and schema check, since the ids
are abcd's own; an external validator adds nothing the Go check lacks.

## Open Questions

_None._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-dbe525f06247 -->
Fidelity review — receipt rcp-dbe525f06247 (verifier intent-auditor claude-opus-5-5).

Provenance: intent-auditor@claude-opus-5-5 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:4685a5a288619851380647e707ec06360e84954083144001c915d57ba4776be2
Input attestations: diff:c66f9959f..9c77aae4f@sha256:4b98c145c87cfc93bf26e7c5a45b7d7c43d64d5cb7016abb7a79d01464953c07; diff:9c77aae4f..a5629f871@sha256:dc5d3e380f731baeca83aea1099ffd186462719a598cd33558337d9eb1a0b0fb;

Acceptance rollup: MET 9 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: resolveCitations scans claims, primary sources, grill notes, kill attempts, reviewer sources, rejected alternatives, origins and the reframing, and the fix 9c77aae4f..a5629f871 adds reviewer labels and the ruling's 'by' beside its reason; CitationError names each part and id, tested per field including the two new cases. The grill's search terms and places stay unchecked by documented design: a search term records what was searched for, not a citation, so it falls outside 'citing a record id'
  evidence: ac1-fix.diff:35 — "+ add(adversarial, r.Label)"
  evidence: ac1-fix.diff:43 — "+ add(adversarial, v.ruling.By, v.ruling.Reason)"
  evidence: ac1-fix.diff:55 — ""reviewer label": {func(p map[string]any) { reviewer(p, 0)["label"] = "the reviewer of an unfiled id" }, "adversarial-review", "an unfiled id"},"
  evidence: ac1-fix.diff:57 — "leg(p, 2)["ruling"] = map[string]any{"by": "the advisor named in an unfiled id", "verdict": "survives", "reason": "both agree"}"
  evidence: internal/core/ideate/record.go:1006 — "for _, c := range v.claims { add(research, c.Claim, c.PrimarySource)"
  evidence: internal/core/ideate/record.go:100 — "parts[i] = c.Where + " cites " + c.ID"
  evidence: internal/core/ideate/record.go:990 — "nor are the grill's search // terms and places, where a searched-for id that names nothing is the finding."
  evidence: internal/core/ideate/v2_test.go:38 — "func TestV2RefusesAnUnresolvedIDInAnyLeg(t *testing.T) {"
- ac-2 — MET: validateResearch refuses a primary source that sourceShaped rejects; sourceShaped accepts only an https:// address with a host, source:< key> in the corpus key grammar, or a record id, and a key is resolved against the default corpus when one exists; tested with 'a remark heard in passing', file://, http:// and malformed keys
  evidence: internal/core/ideate/record.go:572 — "if !sourceShaped(src) {"
  evidence: internal/core/ideate/record.go:600 — "func sourceShaped(src string) bool {"
  evidence: internal/core/ideate/record.go:1194 — "the verdict cites sources the sources corpus does not carry — nothing was written"
  evidence: internal/core/ideate/v2_test.go:96 — "func TestV2RefusesAPrimarySourceOfNoKnownShape(t *testing.T) {"
  evidence: internal/surface/cli/ideate.go:190 — "if dir, err := source.DefaultDir(); err == nil {"
- ac-3 — MET: validateGrill refuses no hits together with no searches, and validateSearches requires each search's terms, places and a non-nil, non-negative count; tested, including the accepted empty grill with a recorded search
  evidence: internal/core/ideate/record.go:685 — "if len(leg.Hits) == 0 && len(leg.Searches) == 0 {"
  evidence: internal/core/ideate/record.go:738 — "if s.Count == nil {"
  evidence: internal/core/ideate/v2_test.go:188 — "func TestV2RefusesAGrillWithNoHitsAndNoSearches(t *testing.T) {"
- ac-4 — MET: validateReframing refuses a reframed verdict whose cleaned reframing is empty (absent or blank), and refuses a reframing beside any other verdict; tested both ways
  evidence: internal/core/ideate/record.go:970 — "case p.Verdict == VerdictReframed && reframing == "":"
  evidence: internal/core/ideate/v2_test.go:239 — "func TestV2RefusesAReframedVerdictWithoutAReframing(t *testing.T) {"
- ac-5 — MET: the Reviewer type holds label, kill_attempts, verdict and sources; Leg carries no top-level kill attempts; the renderer writes one '### Reviewer < label> — < verdict>' section holding that reviewer's attempts and sources; the test asserts each reviewer's attempt and source appear only in its own section
  evidence: internal/core/ideate/ideate.go:296 — "type Reviewer struct {"
  evidence: internal/core/ideate/render.go:134 — "fmt.Fprintf(b, "### Reviewer %s — %s\n\n", r.Label, r.Verdict)"
  evidence: internal/core/ideate/v2_test.go:277 — "func TestV2KeepsEachReviewerUnderItsLabel(t *testing.T) {"
- ac-6 — MET: validateRuling refuses differing reviewer verdicts with no ruling, and a given ruling must name who ruled, a valid verdict and a reason; tested, including the rendered ruling
  evidence: internal/core/ideate/record.go:880 — "if in == nil { for _, r := range reviewers[1:] { if r.Verdict != reviewers[0].Verdict {"
  evidence: internal/core/ideate/record.go:898 — "case out.By == "":"
  evidence: internal/core/ideate/v2_test.go:321 — "func TestV2RefusesASplitWithoutARuling(t *testing.T) {"
- ac-7 — MET: validateResearch refuses a verified claim whose normalised primary source is among the origin keys, and quoted is a registered status; tests cover alternate spellings, an origin with surrounding words, an omitted origin_sources list (refused) and an accepted quoted claim
  evidence: internal/core/ideate/record.go:580 — "if c.Status == ClaimVerified && origin[sourceKey(src)] {"
  evidence: internal/core/ideate/ideate.go:200 — "ClaimQuoted ClaimStatus = "quoted""
  evidence: internal/core/ideate/v2_test.go:367 — "func TestV2RefusesAVerifiedSelfQuotation(t *testing.T) {"
- ac-8 — MET: SchemaVersion is 2; decodePayload peeks schema_version before the strict decode and refuses an older version with olderSchemaError, which names version 2; tested in the core on a real v1-shaped document and at the CLI (exit 2, 'version 2')
  evidence: internal/core/ideate/ideate.go:34 — "const SchemaVersion = 2"
  evidence: internal/core/ideate/record.go:377 — "if json.Unmarshal(raw, &peek) == nil && peek.SchemaVersion > 0 && peek.SchemaVersion < SchemaVersion {"
  evidence: internal/core/ideate/record.go:413 — "compose it as version %d, which adds origin_sources"
  evidence: internal/core/ideate/v2_test.go:447 — "func TestV2RefusesAVersionOnePayload(t *testing.T) {"
  evidence: internal/surface/cli/ideate_surface_test.go:158 — ""version-1-payload","
- ac-9 — MET: the command page's leg 3 lists 'One idea per context' among the leg's properties, and a test reads the leg 3 section of the page for the phrase
  evidence: commands/ideate.md:128 — "- **One idea per context.** Give each reviewer exactly one idea in its context,"
  evidence: internal/core/ideate/v2_test.go:488 — "func TestCommandPageGivesEachReviewerOneIdeaPerContext(t *testing.T) {"

Gap audit:
- honoured:
  - a primary source must be a web address, a filed source or a record
    evidence: internal/core/ideate/record.go:600 — "func sourceShaped(src string) bool {"
  - the grill says what it searched and where, so an empty grill can be told from a broken search
    evidence: internal/core/ideate/record.go:685 — "if len(leg.Hits) == 0 && len(leg.Searches) == 0 {"
    evidence: internal/core/ideate/render.go:115 — "b.WriteString("| Terms | Places | Count |\n|---|---|---|\n")"
  - a reframing has its own field
    evidence: internal/core/ideate/ideate.go:349 — "Reframing string `json:"reframing,omitempty"`"
    evidence: internal/core/ideate/render.go:186 — "b.WriteString("## The reframing\n\n")"
  - two reviewers are recorded reviewer by reviewer, and a split carries the ruling that settled it
    evidence: internal/core/ideate/render.go:134 — "fmt.Fprintf(b, "### Reviewer %s — %s\n\n", r.Label, r.Verdict)"
    evidence: internal/core/ideate/record.go:883 — "the reviewers' verdicts differ (%s) and the adversarial-review leg records no ruling — nothing was written"
  - a claim checked only against the idea's own source reads quoted, never verified
    evidence: internal/core/ideate/record.go:580 — "if c.Status == ClaimVerified && origin[sourceKey(src)] {"
  - the command page is updated to the version 2 shape in the same change, and its example decodes as this binary's payload
    evidence: commands/ideate.md:154 — ""schema_version": 2,"
    evidence: internal/core/ideate/v2_test.go:510 — "if err := dec.Decode(&p); err != nil {"
- diverged:
  - every record id in every leg resolves: delivered over every prose, label and source field (reviewer labels and the ruling's 'by' added in a5629f871), with the grill's search terms and places exempt by design, as the brief, the command page and the code comment all state
    evidence: internal/core/ideate/record.go:994 — "func citationPlaces(v verdictDoc) []citedText {"
    evidence: ac1-fix.diff:25 — "part of the address). A search term naming an id is not checked, because"
    evidence: .abcd/development/brief/04-surfaces/18-ideate.md:151 — "grill's search terms and places are not checked, because a searched-for id"
  - a primary source may be a web address: delivered as https:// only, so an http:// address is refused
    evidence: internal/core/ideate/record.go:602 — "case strings.HasPrefix(src, "https://"):"
    evidence: internal/core/ideate/v2_test.go:100 — ""http://example.invalid/plain","
  - a filed source key is proved filed: only where a sources corpus exists; with none, a well-formed key is accepted on its shape alone
    evidence: internal/core/ideate/record.go:1174 — "case errors.Is(err, source.ErrNoCorpus): return nil"
- missing: (none)

Scope-condition dispositions:
- cond-2610142138048878 — survived: the writer refuses every version below 2 with a message naming version 2, and the ideate package has no non-test file read: existing verdict records are only ever created exclusively, never parsed or backfilled
  evidence: internal/core/ideate/record.go:397 — "case p.SchemaVersion != SchemaVersion: return Payload{}, olderSchemaError(p.SchemaVersion)"
  evidence: internal/core/ideate/record.go:243 — "if err := fsutil.CreateExclusiveIn(root, rel, []byte(body), 0o644); err != nil {"
- cond-2610142138047607 — survived: the command page and the binary change in the same delivered range, a test ties the page's example to the binary's payload type and version, and a v1 payload from an older page gets the refusal naming version 2 and pointing at the command page, at the core and at the CLI
  evidence: internal/core/ideate/v2_test.go:513 — "if p.SchemaVersion != SchemaVersion {"
  evidence: internal/core/ideate/record.go:414 — "and a reframing for a reframed verdict (the ideate command page carries the shape)"
  evidence: internal/surface/cli/ideate_surface_test.go:158 — ""version-1-payload","
- cond-2610142138040936 — survived: every new check is a shape, enum, presence, string-comparison or citation-resolution check; no source is fetched, a ruling is never checked against the overall verdict, and how a reviewer was run is left to the command page
  evidence: internal/core/ideate/record.go:596 — "The // address is not fetched and the record id is resolved with every other // citation (resolveCitations); this is the shape alone"
  evidence: internal/core/ideate/record.go:876 — "Whether the ruling was // right is the person's judgement, and the binary never checks it against the // verdict the whole gauntlet reached."
  evidence: internal/core/ideate/record.go:787 — "conduct is the orchestrating prompt's obligation — the binary cannot observe // how an agent was run"
<!-- abcd-review-end receipt=rcp-dbe525f06247 -->

## Grounds

- pursued: an ideate verdict on record today can cite things that do not exist and still pass; we expect that after this change no verdict on record carries an unresolved citation or an unlabelled self-quotation, and the fix also unblocks release cuts, which refuse while its serious issue is open. Shown wrong if a verdict recorded after it still carries one.
