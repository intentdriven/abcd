---
id: itd-2609020625400169
slug: an-intent-that-a-reading-occasioned-says-so-in-its-origin-wi
spec_id: spc-2609020626042168
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-178, itd-180, itd-185]
severity: minor
impact: fix
origin: researcher-authored
production_mode: dictated-and-formatted
---

# An intent that a reading occasioned says so in its origin, with the run and the item that occasioned it

Typed links: `builds_on` [itd-178](../shipped/itd-178-every-record-written-through-a-command-carries-its-origin-an.md) (the origin key, its parser and its lint), [itd-180](../shipped/itd-180-a-cold-reading-s-findings-land-as-reading-records-and-the-re.md) (the promote path for a dispositioned item), [itd-185](../shipped/itd-185-one-ingest-verb-validates-every-cold-reading-output-includin.md) (the ingest that mints items); `refines` [itd-178](../shipped/itd-178-every-record-written-through-a-command-carries-its-origin-an.md) (the third arrival path gets its writer).

## Press Release

> **A reading's contribution is traceable from both ends.** When an accepted reading item is promoted into an intent draft, the draft's `origin` reads `contributed-by-reading <rdg-N>/<rdi-N>`, naming the run and the item, and the item carries `promoted_to` pointing forward. The provenance lint already resolves that pair to the reading record; now something writes it. From a reading item, the record shows what it caused; from an intent, whether a reading occasioned it. Promotion of an issue keeps saying `extracted-from-record`, because an issue is something a person noticed and a reading item is something an instrument returned.

> "I need to be able to ask, for any intent, whether a cold reading put it on the table, and for any reading item, what became of it," said an AI/agent researcher who keeps the loop's genealogy. "Both directions, from the record, without reading the commit history."

## Why This Matters

[itd-178](../shipped/itd-178-every-record-written-through-a-command-carries-its-origin-an.md) names three values for `origin`: `researcher-authored`, `contributed-by-reading` carrying the run and item identifiers, and `extracted-from-record`. Its second acceptance criterion requires the pair to resolve to a reading record, and the shipped lint checks exactly that. Its fidelity verdict recorded the criterion as having no producer: the stamp primitive refuses the kind, and promoting a dispositioned reading item stamps `extracted-from-record` with `promoted_from`. The linkage the design wants is both directions: an accepted item stamps forward to whatever it produced, and the resulting intent carries the item identifier in `origin`, with the run identifier.

No reading has run, so no record is wrong today. The first accepted item promoted under the current path would be stamped as extracted from a record, which is the wrong claim about where it came from, and the join that the closing run's convergence and purpose-durability readings rest on would be lost at the first use.

## What's In Scope

- **The promote path for a reading item** stamps `origin: contributed-by-reading <rdg-N>/<rdi-N>` on the draft it mints. When it links an existing draft with `--intent`, it writes `promoted_from` and `promoted_to` and leaves the draft's `origin` untouched, because an origin is stamped at mint and never rewritten.
- **The stamp primitive** accepts the kind when, and only when, the caller supplies a well-formed run and item pair; resolution to the readings store is the promote path's, which reads the store before it mints, so no command can write the value without the join.
- **`promoted_from`** keeps naming the item, and `promoted_to` on the item keeps pointing forward, so the pair is redundant by design and the lint can check it both ways.
- **The promoted draft's seed carries no item identifier.** The press release seed is projected to the entailment reading, and a prior item's identifier in it would be revision history; the back-edge lives in `promoted_from`, which is not projected. The read-block eval plants a promoted seed to prove it.
- **The issue promote path is unchanged** and keeps `extracted-from-record`.
- The plugin surface pages for capture and intent say which path writes which value.

## What's Out of Scope

- Backfilling any record. Population is forward-only.
- A reading item promoted to anything other than an intent draft. Other landings (a discipline, an ADR, a brief passage) carry the join by their own means.

## Mechanism

We expect stamping the run and item at promotion to preserve the join because promotion is the only command that moves a reading item toward an intent, and a value written by the command that performs the act cannot drift from the act. It fails if a draft can reach the record with the value typed by hand, which the lint reports when the pair does not resolve and cannot report when it does; that residual is disclosed by itd-178 and stays.

## Scope Conditions

- The value carries exactly one run and one item. An intent occasioned by several items is promoted from one; linking a further item to a draft that already names another source writes that item's `promoted_to`, skips the back-edge and reports it. <!-- cond: cond-2609020727241828 -->
- Promoting a widening item requires its `accepted` disposition, which the admission intent's gate withholds until a comparative run names the item's run, so this path is transitively gated on the comparative channel for widening items. <!-- cond: cond-2609020626045842 -->
- The join resolves item to run directory in the readings store, as the shipped lint already does; no run record is required beyond the item's own directory. <!-- cond: cond-2609020626041091 -->
- **The impact is `fix`, and the reasoning is stated.** The path exists and writes a value the record itself calls the wrong claim; nothing usable changes for an issue promotion, and no reading item has yet been promoted, so there is no working invocation to break. <!-- cond: cond-2609020626044296 -->

## Acceptance Criteria

- **Given** an accepted reading item, **when** `capture promote <rdi-N>` mints a draft, **then** the draft's `origin` is `contributed-by-reading <rdg-N>/<rdi-N>` naming the item's run and id, and the provenance lint resolves it.
- **Given** an accepted reading item and an existing draft, **when** `capture promote <rdi-N> --intent <itd-N>` runs, **then** the draft's `promoted_from` names the item, the item's `promoted_to` names the draft, and the draft's `origin` is unchanged.
- **Given** an open issue, **when** `capture promote <iss-N> --grounds "pursued: <text>"` runs, **then** the draft's `origin` is `extracted-from-record`, unchanged.
- **Given** a call to the stamp primitive with the reading kind and no well-formed run and item pair, **when** it runs, **then** it refuses; resolution to the readings store is the promote path's, which reads the store before it mints.

## Prior Art

- [itd-178](../shipped/itd-178-every-record-written-through-a-command-carries-its-origin-an.md) and its spec; [itd-180](../shipped/itd-180-a-cold-reading-s-findings-land-as-reading-records-and-the-re.md) (the promote path for a dispositioned item).

## Open Questions

None.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-25ad875f8b53 -->
Fidelity review — receipt rcp-25ad875f8b53 (verifier abcd:intent-auditor claude-opus-5[1m]).

Provenance: abcd:intent-auditor@claude-opus-5[1m] · rubric_hash sha256:542ed2cd51ff938717a3f47b2b332e8d47910beec0ca7ecdfd238ae7edf5ced5 · prompt_hash sha256:5e1cf7cbbf12b737a521c1b15f6236d45fc70865f3327a2d4a4520629fff32dd
Input attestations: diff:4137a994~1..4137a994@sha256:a1c079e4c982617086c3fed2136525177ec82ffbb424fcb007913ba37ebc9248;

Acceptance rollup: MET 3 · MET_WITH_CONCERNS 1 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: the reading route mints with Origin{Kind: contributed-by-reading, Run: run, Item: req.ID} where run is read off the item's store path, and the test reads the frontmatter line back and then runs the shipped record_provenance rule over the fixture for zero findings, so 'the lint resolves it' is exercised rather than inferred (both passed on 4137a994)
  evidence: internal/core/capture/promote.go:439 — "Kind: provenance.KindContributedByReading, Run: run, Item: req.ID,"
  evidence: internal/core/capture/promote_reading_origin_test.go:84 — "want := "contributed-by-reading " + readingFixtureRun + "/" + item"
  evidence: internal/core/capture/promote_reading_origin_test.go:97 — "if fs := provenanceLintFindings(t, repo); len(fs) != 0 {"
- ac-2 — MET_WITH_CONCERNS: link mode writes promoted_from through SetPromotedFrom, which sets that one key and never reads or rewrites origin/production_mode, and the item's promoted_to is stamped under the ledger lock; the passing test asserts the two disclosure lines byte-identical before and after. CONCERN, disclosed by the implementer and confirmed here: SetPromotedFrom returns a POPULATED Intent beside a non-nil ErrBackEdgeTaken, against Go's zero-value-on-error convention, and the promote path depends on that value (backEdgeKept = it.PromotedFrom) while the primitive's own test discards the return, so the contract is asserted only indirectly through capture
  evidence: internal/core/intent/lifecycle.go:465 — "setFrontmatterFields(string(data), map[string]string{"promoted_from": source})"
  evidence: internal/core/capture/promote_reading_origin_test.go:143 — "func TestPromoteReadingItemLinkWritesBothEdgesAndLeavesOriginAlone(t *testing.T) {"
  evidence: internal/core/intent/lifecycle.go:456 — "return it, fmt.Errorf("%w: %s is promoted from %s, not %s", ErrBackEdgeTaken, intentID, existing, source)"
  evidence: internal/core/capture/promote.go:411 — "switch it, err := intent.SetPromotedFrom(repoRoot, req.LinkIntent, req.ID); {"
- ac-3 — MET: the issue route's mint still passes extracted-from-record, the change being only the field's type moving from Kind to Origin, and TestPromoteStampsExtractedFromRecord is re-run unchanged and passes
  evidence: internal/core/capture/promote.go:209 — "Origin: provenance.Origin{Kind: provenance.KindExtractedFromRecord},"
  evidence: internal/core/capture/promote.go:203 — "An issue is something a PERSON noticed, so promoting one keeps saying extracted-from-record"
- ac-4 — MET: both halves are delivered as the criterion splits them: NewStamp still refuses the kind outright with a message naming the constructor that takes the pair, NewReadingStamp refuses an unshaped run or item, and the resolution half is the promote path's store read — findReadingItem locates the item and the record's own mandatory run must agree with the directory it sits in before anything is minted; the four covering tests pass
  evidence: internal/core/provenance/provenance.go:182 — "origin %s carries a run and an item identifier, and this constructor takes neither; mint it through NewReadingStamp"
  evidence: internal/core/provenance/provenance.go:205 — "if !readingRunRe.MatchString(run) {"
  evidence: internal/core/capture/promote.go:350 — "src, err := findReadingItem(issuesRoot, req.ID)"
  evidence: internal/core/capture/promote.go:375 — "if declared := asString(fm["run"]); declared != run {"

Gap audit:
- honoured:
  - the origin value takes the form framework 11.3 and divergence-register entry 15 fix: contributed-by-reading < rdg-N>/< rdi-N>, naming the run and the item
    evidence: internal/core/provenance/provenance.go:204 — "func NewReadingStamp(run, item, mode string) (Stamp, error)"
    evidence: evals/testdata/cold-reading/baseline/.abcd/development/intents/drafts/itd-5-a-promoted-draft.md:4 — "origin: contributed-by-reading rdg-2609020000000007/rdi-2609020000000009"
  - the promoted seed carries no item id, and the read-block eval makes companion 8.3 falsifiable: the fixture item's id reaches no bundle at any assembling position
    evidence: internal/core/intent/create.go:440 — "return "_" + promotionSeedOpening + readingSeedSource + ". " + seedNoteTail + "_""
    evidence: evals/coldreading_test.go:50 — "if bytes.Contains(a.BundleRaw, []byte(promotedFixtureItem)) {"
  - framework 7.1 holds: origin stays warm — it is written at mint and never rewritten, and link mode leaves it alone
    evidence: internal/core/intent/lifecycle.go:434 — "It never reads or rewrites `origin` or `production_mode`"
    evidence: evals/coldreading_fixture_test.go:85 — "const promotedFixtureItem = "rdi-2609020000000009""
  - the issue promote path is unchanged and keeps extracted-from-record
    evidence: internal/core/capture/promote.go:209 — "Origin: provenance.Origin{Kind: provenance.KindExtractedFromRecord},"
  - the lint reads the join from both ends — a reading origin with no back-edge, one naming a different item, and an item whose promoted_to names some other record
    evidence: internal/core/lint/provenance.go:168 — "case !hasBack:"
    evidence: internal/core/lint/provenance.go:182 — "if forward, ok := promotedTo[o.Item]; ok && forward != r.handle() {"
  - the two plugin pages say which path writes which value
    evidence: commands/capture.md:67 — "`contributed-by-reading <rdg-N>/<rdi-N>`, which `capture promote <rdi-N>` mints"
    evidence: commands/intent.md:87 — "mints from an accepted reading item is `contributed-by-reading <rdg-N>/<rdi-N>`,"
  - the kept back-edge is reported in the result and in both renderings
    evidence: internal/core/capture/promote.go:60 — "BackEdgeKept string `json:"back_edge_kept,omitempty"`"
    evidence: internal/surface/cli/cli.go:2800 — "fmt.Fprintf(w, "back_edge: kept %s\n", termsafe.Sanitize(res.BackEdgeKept))"
- diverged:
  - the spec's Approach has the run come from capture.findReadingItem, 'which returns the run and the path'; the delivered route derives it as filepath.Base(filepath.Dir(src)) and findReadingItem still returns (string, error). Substantively the same join — readingItemPaths admits only directories passing recordid.ValidReadingRunID, so the derived run is well formed by construction, and the record's declared run must agree before anything is minted — so the delta is where the pair is assembled, not what is stamped
    evidence: internal/core/capture/reading.go:674 — "func findReadingItem(issuesRoot, item string) (string, error) {"
    evidence: internal/core/capture/promote.go:374 — "run := filepath.Base(filepath.Dir(src))"
    evidence: internal/core/capture/reading.go:711 — "if !recordid.ValidReadingRunID(run.Name()) {"
  - SetPromotedFrom returns a populated Intent beside a non-nil ErrBackEdgeTaken rather than the zero value Go's convention expects. It is deliberate and documented, and the value returned is truthful (the record as loaded), but it makes a shared lifecycle primitive's contract depend on a caller reading the doc comment
    evidence: internal/core/intent/lifecycle.go:434 — "The intent it returns beside that error carries the edge it kept"
    evidence: internal/core/intent/lifecycle.go:456 — "return it, fmt.Errorf("%w: %s is promoted from %s, not %s", ErrBackEdgeTaken, intentID, existing, source)"
- missing:
  - no test at the primitive asserts the returned-Intent-beside-ErrBackEdgeTaken contract the doc comment states: TestSetPromotedFromReportsATakenBackEdgeAndIsIdempotentOnTheSame discards the Intent, so the only coverage is indirect, through capture's res.BackEdgeKept
    evidence: internal/core/intent/intent_test.go:740 — "_, err = SetPromotedFrom(root, "itd-11", "rdi-18")"
    evidence: internal/core/capture/promote_reading_origin_test.go:203 — "if res.BackEdgeKept != firstItem {"

Scope-condition dispositions:
- cond-2609020727241828 — survived: the several-items case is delivered exactly as assumed: a taken back-edge is kept, the second item's promoted_to is still stamped, and the kept record is reported in the result and both renderings
  evidence: internal/core/capture/promote.go:411 — "switch it, err := intent.SetPromotedFrom(repoRoot, req.LinkIntent, req.ID); {"
  evidence: internal/core/capture/promote_reading_origin_test.go:187 — "func TestPromoteReadingItemLinkKeepsAnExistingBackEdge(t *testing.T) {"
- cond-2609020626045842 — untested: nothing in the delivered diff exercises or contradicts the widening position's disposition gate; the promote fixture sits at the detection position and the commit says outright that the gate is the comparative and admission specs'
- cond-2609020626041091 — survived: the join is item-to-run-directory and nothing more — promote reads the run off the path the item was found at and the lint keys the same map off the record's bucket; no rdg-N run record is opened on either side, and the fixture promote is clean under the shipped rule
  evidence: internal/core/capture/promote.go:374 — "run := filepath.Base(filepath.Dir(src))"
  evidence: internal/core/lint/provenance.go:92 — "runOf[r.handle()] = r.bucket"
- cond-2609020626044296 — survived: the impact stayed fix and the stated reasoning still holds against the delivered tree: the issue route's stamp is byte-unchanged, and the ledger holds no readings/ tree at all, so no promoted reading item exists whose record this would have changed
  evidence: .abcd/development/intents/shipped/itd-2609020625400169-an-intent-that-a-reading-occasioned-says-so-in-its-origin-wi.md:10 — "impact: fix"
  evidence: internal/core/capture/promote.go:209 — "Origin: provenance.Origin{Kind: provenance.KindExtractedFromRecord},"
## Grounds

- pursued: we expect stamping the run and item at promotion to preserve the detection-to-intent join the closing run rests on, because promotion is the only command that moves a reading item toward an intent; a promoted draft whose origin cannot be resolved back to the item that occasioned it would show it wrong
