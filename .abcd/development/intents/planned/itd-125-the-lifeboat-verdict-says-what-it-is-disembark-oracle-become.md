---
id: itd-125
spec_id: spc-30
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: major
impact: breaking
slug: the-lifeboat-verdict-says-what-it-is-disembark-oracle-become
---

# The Lifeboat Verdict Says What It Is

## Press Release

The lifeboat's verdict says what it is. `abcd disembark review` weighs a
packed lifeboat and returns `SHIP`, `NEEDS_WORK`, or `MAJOR_RETHINK` — a
judgement over the pack, which is a review, and now the verb, the flag, the
artefact, and the prose all say so. "The file used to be called an audit,
produced by an oracle, holding a review verdict — three words for one thing.
Now it's one word," says Kira, maintainer.

## Why This Matters

The 2026-08-16 planning investigation found what adr-40 §5 missed: the binary
verb never invokes the oracle seam. `disembark oracle` is a compute-or-ingest
verdict endpoint — deterministic mode is a mechanical mapping over the
manifest seal and coverage summary, delegated mode validates a verdict the
host's agent produced elsewhere. Naming the verb for the seam claims a seam
this verb doesn't touch, would collide with the `/abcd:oracle ask` design
target (the real seam surface, adr-25), and leaves the artefact
(`audit/oracle-*.json`) contradicting the vocabulary at every level. The
maintainer therefore **reversed adr-40 §5** ("keep the verb, fix the prose")
in favour of the rename — the reversal is recorded as a dated amendment
inside adr-40 §5 and a `DECISIONS.md` line. The oracle seam itself is
untouched: adr-25 stands, and `oracle` remains the seam's name.

## Scope Conditions

None stated.

## Acceptance Criteria

> _BDD format, per the itd-1 discipline. Walked and confirmed by the
> maintainer, 2026-08-16._

- **Given** the rename, **then** `disembark review <lifeboat-dir>
  <source-repo> [--review-json <path>|-]` replaces the `oracle` spellings,
  and the old sub-verb and flag fail as unknown — no alias, no shim.
- **Given** behaviour, **then** both modes are preserved byte-for-byte: the
  deterministic manifest+coverage mapping and the delegated
  validate-and-ingest (enum-membership gate, cite-or-be-dropped findings,
  core-stamped attestation), including the exit contract.
- **Given** the artefacts, **then** the synthesis output moves to
  `review/review-<manifest12>.json` + `.md`, still excluded from
  `manifest_sha256`.
- **Given** an older lifeboat carrying `audit/oracle-<manifest12>.json`,
  **then** a re-run writes the new path and removes the superseded old-name
  file for the same manifest — the clean-replacement guarantee survives the
  rename; two verdicts for one manifest never coexist.
- **Given** the sweep, **then** the `lifeboat-oracle` agent becomes
  `lifeboat-reviewer`, `commands/disembark.md` and the brief's "oracle
  audit" prose move to "review", the historical-records boundary holds as in
  the sibling renames, and the `oracle_review` task-class token is
  re-checked in the same change.
- **Given** the decision record, **then** the adr-40 §5 amendment (dated,
  in-place, recording the compute-or-ingest finding and the reversal) and
  one dated `DECISIONS.md` line land with this intent's planning records.
- **Given** itd-122's extended `surface_coverage` armed, **then** the
  `disembark` sub-verb table row flips in the same change — `record-lint`
  exit 0 proves the migration; landing before the check is armed is
  forbidden.
- **Given** the release record, **then** the CHANGELOG entry is breaking.

## SOTA

Same family as itd-123/itd-124: pre-1.0.0 breaking rename, no aliases
(adr-40; iss-171 precedent). The naming pattern follows the session's ruled
convention — target-grain + role (`intent-auditor`, `lifeboat-reviewer`) and
self-describing artefact basenames. **Chosen path: bespoke sweep**, proved
complete by the armed itd-122 gate. No new dependency.

## Open Questions

_None gating. The adr-40 §5 reversal was investigated, confirmed, and homed
by the maintainer, 2026-08-16._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
