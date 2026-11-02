---
id: itd-171
slug: a-decision-recorded-in-an-intent-an-adr
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2608290819228175]
---

# A decision points back to the conversation where it was reached

## Press Release

> _Seeded by promotion from iss-2608290819228175. Expand into the full press-release narrative before planning._

## Why This Matters

A decision recorded in an intent, an architecture decision record, or the decision log should point at the passage of the redacted transcript where it was actually reached, so the reasoning behind a record is recoverable rather than lost with the session that produced it. The record is the floor of recoverable theory, not the theory itself.

This is the trace the facilitator needs when the product thinker reports that the product does not match what they expected. Because the facilitator is a machine by default, the trace has to be machine-followable rather than merely readable by a person who was there.

The store is closer to this than it looks. A transcript record already carries a session identifier, a content hash, and a capture timestamp, so the session is already addressable. What is missing is granularity inside one transcript: there is no per-turn anchor, so a link can name the session but not the passage.

Two constraints rule out the obvious approach. A line number is the wrong anchor, and this repository already refuses that shape elsewhere. And redaction rewrites spans in place, so any offset computed against the raw transcript is invalid against the stored redacted one: an anchor must be derived from the redacted artefact or be content-addressed rather than positional.

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [sources-provenance SOTA note](../../research/notes/2026-10-23-sources-provenance-sota.md).

- **The anchor standard exists.** The W3C Web Annotation Data Model's `TextQuoteSelector` "MUST have exactly 1 `exact` property" and SHOULD carry one `prefix` and one `suffix`, text normalised and counted in code points; a `TextPositionSelector` is "very brittle with regards to changes to the resource"; several selectors may describe one segment and a consumer "MUST pick one".
- **Re-anchoring has a settled recipe.** Hypothesis stores range, position and quote selectors (32 characters of context) and tries them in order: range, position, context-first fuzzy match near the expected offset, whole-document fuzzy match, with an acceptance threshold (Hypothesis fuzzy anchoring). Browser text fragments (`#:~:text=prefix-,start,end,-suffix`) are the same idea built to "degrade gracefully into an ordinary link".
- **The host transcript already has a per-entry id.** Community documentation of the JSONL describes a `uuid` and `parentUuid` per line; the vendor's hooks page confirms `session_id` and `transcript_path` (which "may lag the in-memory conversation") and a per-prompt `prompt_id`, but not the per-entry id, so its stability is not vendor-documented.
- **Commit-side linking is settled as a trailer the minting tool resolves**: Gerrit's `Change-Id` footer inserted by a commit-msg hook and kept across amend and rebase; Entire's `Entire-Checkpoint: <id>` with sessions under `refs/entire/checkpoints/...`, never on the active branch, secrets redacted best-effort; GitHub's `Agent-Logs-Url` trailer on cloud-agent commits. The 2026-07-30 note already placed abcd on the transcripts-out-of-repo side; this intent is the record-side link.
- **Path 2**: a typed cross-reference minted when the decision is recorded, holding `session_id`, `source_sha256` of the stored record, the host entry id when present, and a `TextQuoteSelector` computed from the redacted body (never the raw one) with a position selector as a hint only; stored on the decision record, never in the transcript; resolved by `abcd history anchor <ref>` in Hypothesis's order, reporting which stage matched so a stale anchor degrades to "session found, passage moved". Any line or byte form is the hint, honouring the existing refusal of positional references. No Go dependency. The independent fit-challenge runs at plan time.

## Open Questions

- (SOTA 2026-10-23) The quoted `exact`, `prefix` and `suffix` are transcript prose entering a committed record; do they pass the banlist and identity scans at mint time, and how long may the span be?
- (SOTA 2026-10-23) When a re-redaction rewrites inside a quoted span, is the anchor re-minted, marked suspect in the manner of a fingerprinted link, or left to fail at resolve time?
- (SOTA 2026-10-23) Is a span of a few hundred characters the unit the facilitator needs, or the whole turn (the entry id) with the span as the highlight?

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
