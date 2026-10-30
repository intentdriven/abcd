---
schema_version: 1
id: "iss-2610262041521807"
slug: "the-verification-matrix-marks-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1031 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/02-verification-matrix.md:80"
remedy: "none (filed automatically)"
related_intents: [itd-34]
finding_ref: cfk-1c312f79c3f43a9a
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The matrix is accurate about the code (no suggested_kind classifier, no shape role) and the shipped intent's press release is a promise whose delivered part its own audit records; rewriting a shipped record would not add anything."
---

The verification matrix marks the capture-time kind classifier and the shape role as staged; shipped itd-34 claims both.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/06-delivery/02-verification-matrix.md:80` — "the capture-time `suggested_kind` classifier, and the shape-classification role that would propose a reclassification"
- End B: `.abcd/development/intents/shipped/itd-34-three-intent-kinds.md:21` — "with an `oracle` classifier writing an advisory hint at capture time"

The matrix's three-intent-kinds row marks as staged the capture-time suggested_kind classifier and the shape-classification role that would propose a reclassification. It also says the committed pre-commit hook runs no record lint. The shipped itd-34's press release says an oracle classifier writes an advisory hint at capture time, and its scope runs the shape role continuously from the pre-commit hook. The intent claims as delivered what the matrix says is not built.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1031.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The matrix is accurate about the code (no suggested_kind classifier, no shape role) and the shipped intent's press release is a promise whose delivered part its own audit records; rewriting a shipped record would not add anything.
