---
id: itd-2609212137116617
slug: a-new-capture-or-draft-is-matched
spec_id: spc-2609212141417782
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-84, itd-42]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-87, itd-48]
related_adrs: [adr-2609212115255771]
---

# A new capture or draft is matched against the record before it is written

## Press Release

> **A new issue or draft is matched against the record at filing; a likely double is linked, named, and never dropped.**
>
> "I filed the same finding twice a month apart and nobody noticed until a consistency pass," said a technical facilitator. "Now the capture tells me at filing that it looks like iss-N, writes the link, and leaves it to me to confirm. Nothing is refused: a wrong match is a link I remove, not a finding I lost."

## Why This Matters

Three places look for a match today, none at filing: the pre-pass at planning (itd-42), the drain before it captures, and itd-87's recurrence rule after closure. The itd-84 discipline names a capture-time validator as its next rung. The research of 2026-09-21 on unattended triage is plain about the failure to avoid: a matcher that refuses is the one that loses findings silently. Ruled: file it, link it, say so.

## Mechanism

We expect a typed link written at filing to make doubles visible where a later triage never finds them, because the moment of filing is the only moment both records are in one hand; shown wrong if the next consistency pass still finds unlinked doubles filed after this ships.

## Scope Conditions

- Holds for text long enough to compare; a one-line capture below the declared minimum is filed without matching and says so. <!-- cond: cond-2609212141415326 -->

## What's In Scope

- **The match**: `capture` and the quoted-text `intent` create compare the new text with every open and resolved issue and every intent's title and press release, by the term-overlap heuristic the embark ranking uses, declared a heuristic.
- **The link**: a likely match is written onto the new record as `duplicates:` (near-identical) or `refines:` (narrower), naming the candidate; the verb prints the match and the link; nothing is refused or dropped.
- **The confirmation**: a person or a later pass confirms or removes the link; a record whose link is removed is ordinary.
- **The configuration**: threshold and compared fields declared with a bundled default; below the threshold nothing is written and `--json` lists the near misses.
- **The rung**: the itd-84 discipline's capture-time validator is marked delivered by this record.

## What's Out of Scope

- Refusing or merging records.
- Matching across repositories.
- Semantic matching by a model (the heuristic is lexical; a host pass is a later rung).

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that filed and planned this intent (adr-2609212115255771 records the vocabulary rulings it rests on):

1. File it, link it, say so; never refuse, never drop (ruled 2026-09-21).
2. A lexical heuristic, declared as one; the threshold is configuration.

## Open Questions

_None open._

## Acceptance Criteria

- **Given** a new capture whose text overlaps an existing issue above the threshold, **when** it is filed, **then** the record is written with a `duplicates:` or `refines:` link naming the candidate, and the verb prints the match.
- **Given** a new draft intent whose text overlaps an existing intent's title or press release, **when** it is created, **then** the same link is written and printed.
- **Given** any match, **when** filing completes, **then** no record was refused or dropped; removing the link leaves an ordinary record.
- **Given** overlap below the threshold, **when** filing completes, **then** nothing is written for it and `--json` lists the near misses with their scores.
- **Given** the itd-84 discipline record, **when** it is read after this ships, **then** the capture-time validator rung reads delivered by this record.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-1887e5f574ef -->
Fidelity review — receipt rcp-1887e5f574ef (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:55cacd764222cf6126e4455ac627a86ccad5b272fd5e58b31e6d99d69a11c824
Input attestations: tree:ceb4b6dbb97622bddf2401c91f9f808ed05060a0 (origin/main; internal/core/record/match, internal/core/capture/match.go, internal/core/intent/match.go, internal/surface/cli/match.go)@-;

Acceptance rollup: MET 5 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: the capture workflow runs matchAndLink under the ledger lock before the write, writes duplicates:/refines: into the record's frontmatter, and the CLI prints each link written; TestCaptureLinksAPlantedDouble and TestCaptureVerbLinksAndPrintsTheMatch exercise both ends and pass at BASE
  evidence: internal/core/capture/workflow.go:267 — "content, matched = matchAndLink(repoRoot, issuesRoot, *req.Match, req.Text, content, fm)"
  evidence: internal/core/capture/match.go:78 — "func matchAndLink(repoRoot, issuesRoot string, cfg match.Config, text, content string, fm map[string]any) (string, *match.Outcome)"
  evidence: internal/surface/cli/match.go:36 — "func renderMatch(w io.Writer, o *match.Outcome)"
  evidence: internal/core/capture/match_test.go:66 — "func TestCaptureLinksAPlantedDouble"
  evidence: internal/surface/cli/match_cli_test.go:67 — "func TestCaptureVerbLinksAndPrintsTheMatch"
- ac-2 — MET: the quoted-text intent create matches title plus press release under the mint lock against every intent's H1 and press release (and the ledger), seeds the typed links into the draft, and the CLI prints the outcome; TestCreateFromTextMatchedLinksADouble and TestIntentCreateVerbLinksAndPrintsTheMatch pass at BASE
  evidence: internal/core/intent/create.go:324 — "outcome = runMatch(opts.Match, opts.Title+"\n"+opts.PressRelease)"
  evidence: internal/core/intent/create.go:468 — "for _, rel := range []match.Relation{match.Duplicates, match.Refines} {"
  evidence: internal/core/intent/match.go:35 — "func MatchTexts(repoRoot string) ([]MatchText, error)"
  evidence: internal/surface/cli/cli.go:2686 — "it, err := intent.CreateFromTextMatched(repoRoot, text, opts, m)"
  evidence: internal/surface/cli/match_cli_test.go:120 — "func TestIntentCreateVerbLinksAndPrintsTheMatch"
- ac-3 — MET: an unreadable candidate set, a refused configuration or a link the schema refuses all come back as an outcome with the record filed unlinked, never an error; a record whose link line is removed is listed back as ordinary; TestCaptureIsNeverRefusedByTheMatch, TestCreateFromTextMatchedNeverRefuses, TestCaptureVerbFilesThroughARefusedConfiguration and TestRemovingTheLinkLeavesAnOrdinaryRecord pass at BASE
  evidence: internal/core/capture/match.go:113 — "o.Skipped = fmt.Sprintf("the links could not be written (%v), so the record is filed unlinked", err)"
  evidence: internal/surface/cli/match.go:26 — "so the record is filed without matching; fix or remove the key"
  evidence: internal/core/capture/match_test.go:176 — "func TestRemovingTheLinkLeavesAnOrdinaryRecord"
  evidence: internal/core/intent/match_test.go:71 — "func TestCreateFromTextMatchedNeverRefuses"
- ac-4 — MET: a score below the threshold takes no Relation and is never linked, up to five near misses travel on the outcome with score and reverse, the threshold and compared fields come from match.threshold / match.fields through the layered reader with bundled defaults, and --json carries near_misses; TestCaptureBelowTheThresholdWritesNothingAndListsNearMisses, TestCaptureVerbJSONListsNearMisses and TestTheThresholdIsHonoured pass at BASE
  evidence: internal/core/record/match/match.go:317 — "if len(o.NearMisses) < NearMissLimit {"
  evidence: internal/core/record/match/config.go:67 — "th, err := layered.Get(s, "match.threshold", DefaultThreshold,"
  evidence: internal/core/capture/match_test.go:120 — "func TestCaptureBelowTheThresholdWritesNothingAndListsNearMisses"
  evidence: internal/surface/cli/match_cli_test.go:89 — "func TestCaptureVerbJSONListsNearMisses"
- ac-5 — MET: the itd-84 discipline record carries a Delivered rung paragraph naming this intent as what delivers the capture-time candidate pass, and says what it does not deliver
  evidence: .abcd/development/intents/disciplines/itd-84-intent-decomposition.md:63 — "**Delivered rung: the capture-time candidate pass.** The lexical shortlist of rule 2 runs at filing, delivered by"
  evidence: .abcd/development/intents/disciplines/itd-84-intent-decomposition.md:65 — "[itd-2609212137116617] (../shipped/itd-2609212137116617-a-new-capture-or-draft-is-matched.md)"

Gap audit:
- honoured:
  - file it, link it, say so; never refuse, never drop (decision 1)
    evidence: internal/core/intent/match.go:105 — "The match never refuses the create"
    evidence: internal/core/capture/match.go:76 — "It never fails the capture"
  - a lexical heuristic declared as one on every outcome, threshold as configuration (decision 2)
    evidence: internal/core/record/match/match.go:31 — "const Heuristic = "lexical term overlap, weighted by how rare each term is across the candidates; a heuristic, not a judgement""
    evidence: internal/core/record/match/config.go:54 — "LoadConfig resolves match.threshold and match.fields through the one layered"
  - candidates are every open and resolved issue and every intent's title and press release; wontfix is excluded
    evidence: internal/core/capture/match.go:43 — "for _, st := range []State{StateOpen, StateResolved} {"
  - only duplicates and refines are ever proposed; reverses and supersedes stay human (itd-84 rule 3)
    evidence: internal/core/record/match/match.go:34 — "a lexical // score can only ever propose the two below"
  - the plugin pages document the links and near_misses
    evidence: commands/capture.md:117 — "`duplicates: [<id>]` when the two hold each other's terms"
    evidence: commands/intent.md:80 — "`duplicates:` or `refines:` link for each likely double (itd-2609212137116617)"
- diverged:
  - a new issue is matched at filing (press release): the ledger's other filing paths file without matching — inbox promote builds a CaptureRequest with no Match, and the consistency and reading ingests write issue records outside matchAndLink; only the capture verb and the quoted-text intent create match
    evidence: internal/core/report/inbox.go:683 — "return capture.CaptureRequest{"
    evidence: internal/core/capture/workflow.go:267 — "content, matched = matchAndLink(repoRoot, issuesRoot, *req.Match"
    evidence: internal/core/capture/consistency.go:22 — "func IngestConsistency(repoRoot string, payload []byte, date string)"
  - the overlap function is the one the embark ranking uses, moved to the record package (spec Scope 1 / Approach): no such function existed, the primitive shipped new as internal/core/record/match; recorded by the spec amendment and iss-2609261631134401
    evidence: .abcd/development/specs/closed/spc-2609212141417782-a-new-capture-or-draft-is-matched.md:51 — "The Scope, Approach and Footprint above name a delivery that did not happen"
- missing: (none)

Scope-condition dispositions:
- cond-2609212141415326 — survived: a text with fewer than MinTerms (8) distinct terms is filed with no candidate gathered and the outcome's Skipped names the declared minimum, which the CLI prints as not matched; TestAShortTextIsFiledWithoutMatchingAndSaysSo passes at BASE
  evidence: internal/core/record/match/match.go:283 — "o.Skipped = "the text carries fewer distinct terms than the declared minimum, so it is filed without matching""
  evidence: internal/core/record/match/match.go:263 — "func Short(text string) bool { return len(Terms(text)) < MinTerms }"
  evidence: internal/core/record/match/match_test.go:131 — "func TestAShortTextIsFiledWithoutMatchingAndSaysSo"
<!-- abcd-review-end receipt=rcp-1887e5f574ef -->

## Grounds

- pursued: the ledger holds 480 open issues and the run and the drain will file more unattended; we expect a link at filing to make doubles visible; shown wrong if the next consistency pass still finds unlinked doubles filed after this ships
