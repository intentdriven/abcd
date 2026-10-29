---
schema_version: 1
id: "iss-2610262042345947"
slug: "itd-1-gives-intent-auditor-four-verdict"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1078 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-1-acceptance-gates.md:46"
remedy: "Ruled 2026-10-27, the recommended answer accepted (dcn-2610271112330536): Does itd-81's binary-verdicts rule bind the intent-auditor's four-word per-criterion verdicts, or is it narrowed to no numeric scores? Recommended: narrow it to 'a closed categorical verdict, never a numeric score' and calibrate each criterion verdict against its label, because the stated reason (numeric scales drift to the middle) does not apply to a four-word enum and the four states are shipped."
related_intents: [itd-1, itd-81]
finding_ref: cfk-5856fa7997d50171
---

itd-1 gives intent-auditor four verdict states and rejects binary pass/fail; itd-81 requires binary verdicts from every judge, intent-auditor named first

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-1-acceptance-gates.md:46` — "Binary pass/fail loses information; four states preserve nuance without exploding."
- End B: `.abcd/development/intents/disciplines/itd-81-judge-calibration.md:102` — "Binary verdicts. Severity is structural (a FIX-FIRST list vs a NOTES list)"

itd-1 says binary pass/fail loses information and has intent-auditor emit four states. itd-81 binds every verdict-rendering agent, today intent-auditor, to binary verdicts with severity carried by structure. The calibration that itd-81 scores and the verdict shape that itd-1 mandates disagree for the same agent.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1078.
