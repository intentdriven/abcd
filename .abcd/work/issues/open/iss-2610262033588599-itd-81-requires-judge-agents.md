---
schema_version: 1
id: "iss-2610262033588599"
slug: "itd-81-requires-judge-agents"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 458 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-81-judge-calibration.md:102"
remedy: "Ruled 2026-10-27, the recommended answer accepted (dcn-2610271112330536): Decision: does itd-81's 'Binary verdicts' requirement bind the closed categorical verdict sets already wired (itd-1's four per-criterion verdicts for intent-auditor, the SHIP / NEEDS_WORK / MAJOR_RETHINK review enum, itd-84's three outcomes), or does it only forbid model-emitted numeric or graded severity scores? Answers: (A) it binds them, so those enums must be collapsed to binary; (B) it forbids numeric scores only, and calibration scores each enum collapsed to flagged versus clean. Recommend B: the enums are load-bearing in code, the audit surface and the matrix, and the evidence itd-81 cites (drift of numeric scales, position bias) concerns scales, not small categorical sets; then reword itd-81 line 102."
related_intents: [itd-81]
finding_ref: cfk-55a654c7f2118da9
deferred_after: v0.15.1
deferral_reason: "Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes."
---

itd-81 requires judge agents, intent-auditor named first, to emit binary verdicts; the intent surface has intent-auditor emit four-valued per-criterion verdicts

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-81-judge-calibration.md:102` — "Binary verdicts. Severity is structural (a FIX-FIRST list vs a NOTES list)"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:326` — "Per-criterion verdicts (`MET` / `MET_WITH_CONCERNS` / `NOT_MET` / `INCONCLUSIVE`) appended to the intent's `## Audit Notes`."

itd-81 names intent-auditor as the verdict-rendering agent of today and puts binary verdicts, with severity carried structurally rather than graded, into the output contract its calibration scores. The intent surface's audit records each criterion as one of four verdicts, MET, MET_WITH_CONCERNS, NOT_MET or INCONCLUSIVE, including a graded middle. The same agent's output cannot be both binary and four-valued, so a calibration built to itd-81 cannot score the audit as the intent surface specifies it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 458.

## Deferral 2026-10-27

Deferred past v0.15.1: Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes.
