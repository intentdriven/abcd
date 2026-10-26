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
remedy: "none (filed automatically)"
related_intents: [itd-81]
finding_ref: cfk-55a654c7f2118da9
---

itd-81 requires judge agents, intent-auditor named first, to emit binary verdicts; the intent surface has intent-auditor emit four-valued per-criterion verdicts

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-81-judge-calibration.md:102` — "Binary verdicts. Severity is structural (a FIX-FIRST list vs a NOTES list)"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:326` — "Per-criterion verdicts (`MET` / `MET_WITH_CONCERNS` / `NOT_MET` / `INCONCLUSIVE`) appended to the intent's `## Audit Notes`."

itd-81 names intent-auditor as the verdict-rendering agent of today and puts binary verdicts, with severity carried structurally rather than graded, into the output contract its calibration scores. The intent surface's audit records each criterion as one of four verdicts, MET, MET_WITH_CONCERNS, NOT_MET or INCONCLUSIVE, including a graded middle. The same agent's output cannot be both binary and four-valued, so a calibration built to itd-81 cannot score the audit as the intent surface specifies it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 458.
