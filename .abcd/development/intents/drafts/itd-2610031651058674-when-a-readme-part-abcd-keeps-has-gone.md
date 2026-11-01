---
id: itd-2610031651058674
slug: when-a-readme-part-abcd-keeps-has-gone
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610031348087517]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# abcd's check reports README parts that have gone stale, and writes nothing

## Press Release

> When a README part abcd keeps has gone stale or been edited by hand since the last setup run, abcd's check says which part and why, and writes nothing; the owner decides whether to run setup again or keep their edit.

_Proposed by the facilitator on filing (2026-10-03): the product thinker chose, at itd-2610031348087517's interview (its decision 1), that keeping README parts current also means a separate check that reports a stale part and writes nothing; to be confirmed at this draft's own planning interview._

## Why This Matters

_Facilitator-written._

Setup rewrites the README parts an owner chose only when it runs; between runs a licence can change, a workflow can be renamed, or the owner can edit inside a marked part. This check says so without touching the file, so a person or a continuous-integration run sees the drift and the owner decides what happens. The research for the parent draft found no tool that reports drift outside its own markers; the cog tool's checksum guard is the model for telling a hand edit from a stale derivation.

Typed links: builds on itd-2610031348087517 (the README parts and their marked places).

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [dashboard-sites SOTA note](../../research/notes/2026-10-23-dashboard-sites-sota.md).

- **The marker already carries a checksum:** each part is wrapped in `<!-- BEGIN ABCD:<name> sha256=… -->` over a normalised body (`internal/core/ahoy/readme_marker.go`). That is cog's design: `-c` will "checksum the output to protect it against accidental change" (`[[[end]]] (sum: …)`), and on a hand edit "the checksum won't match, and cog will stop to avoid overwriting the edited output" (cog docs).
- **"Would change if run again" is the staleness check the field uses:** cog's `--check` ("Check that the files would not change if run again", with `--diff` to show why); embedme's `--verify` ("Verify that running embedme would result in no changes. Useful for CI"); terraform-docs injects between `BEGIN_TF_DOCS` and `END_TF_DOCS` markers and leaves the rest alone. Only cog tells a hand edit from a stale derivation, and only at write time; the two-finding report is abcd's addition.
- **Exit codes separate drift from breakage:** Prettier's `--check` exits 0 (formatted), 1 (not formatted), 2 (something wrong with the tool); abcd's lint already exits 2 on an error and 1 on warnings alone. A drifted part is the warning class; a duplicated or unclosed marker the error class.
- **No colour in a pipe** (clig.dev; adr-49): the check runs in CI.

**Path 2 — native, shaped to cog's checksum-plus-check pair and Prettier's exit codes: two findings per part from the marker's own data, hand-edited (body hash differs from the marker's) and stale (hashes agree but the regenerated body differs), exit 1 on either and 2 on a broken marker, with an optional diff.** The check reads the marker and never writes. The independent fit-challenge runs at plan time.

## Open Questions

- (SOTA 2026-10-23) Whether regenerating a part for comparison may make the badge-address or outbound checks the setup path makes, or must read the disk alone (adr-38).
- (SOTA 2026-10-23) Whether a hand-edited part fails CI (exit 1) by default, given the owner may have meant the edit; and what the finding tells them to do (remove the marker, or run setup).

## Typed Links

- **built on by and refined by `itd-2611011151512956`** (hand-written text caught and offered back to its source): Widens this check from the README parts to every rendered section and adds the offer back.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
