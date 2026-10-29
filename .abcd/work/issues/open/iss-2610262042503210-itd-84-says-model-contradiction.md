---
schema_version: 1
id: "iss-2610262042503210"
slug: "itd-84-says-model-contradiction"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1096 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-84-intent-decomposition.md:33"
remedy: "Ruled 2026-10-27, the recommended answer accepted (dcn-2610271112330536): Decision: does itd-84's rule that model contradiction detection is advisory and never holds a gate shut or auto-files bind the consistency-pass pre-commit hook, whose stop and automatic modes can refuse a commit or file an issue on a model-pass finding? Answers: (A) yes, so stop and automatic apply only to the deterministic rule-seen codes and a model-pass finding only ever warns; (B) no, the owner who picks stop or automatic at setup accepts model false positives. Recommend A: itd-84 cites over-flagging as the dominant failure of contradiction detection, and the hook's own text already says it never refuses on a judgement it could not obtain; narrowing the modes keeps stop useful without letting a model hold a commit."
related_intents: [itd-84, itd-2609291923559186]
finding_ref: cfk-31e4a79e0a4969c3
---

itd-84 says model contradiction detection must never hold a gate shut on its own; itd-2609291923559186's hook can refuse a commit on one

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-84-intent-decomposition.md:33` — "Contradiction detection is unreliable even on frontier models and over-flags by default"
- End B: `.abcd/development/intents/planned/itd-2609291923559186-the-consistency-pass-runs-where-records.md:68` — "refuse the commit naming the finding"

itd-84 says contradiction detection over-flags even on frontier models, so it stays advisory, human-confirmed and never auto-filed. itd-2609291923559186's pre-commit hook can be set to stop, refusing the commit on a finding that includes model-pass contradictions, or to automatic, filing judgement contradictions as issues without a person. Both modes are what the discipline rules out.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1096.
