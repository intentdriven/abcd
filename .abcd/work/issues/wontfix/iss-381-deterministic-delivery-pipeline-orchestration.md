---
schema_version: 1
id: "iss-381"
slug: "deterministic-delivery-pipeline-orchestration"
severity: "major"
category: "future-work-seed"
source: "review-followup"
found_during: "itd-130 end-to-end delivery"
wontfix_reason: "Overtaken (product thinker, 2026-09-23 run A interview, M34): the ideate gauntlet reframed the deterministic delivery pipeline (research/notes/2026-08-20-ideate-deterministic-delivery-pipeline.md, verdict reframed), and the reframed capability is the planned itd-2609201916151817 (one verb takes a single intent from ready to delivered). The routing's 'needs ideate' was stale."
---

The delivery pipeline abcd runs on itself is hand-orchestrated and should be deterministic. This session hand-ran the full sequence: SOTA research -> itd-84 decomposition -> draft intent -> TWO independent adversarial reviews (design/feasibility + record-discipline) -> planning interview -> plan+spec -> test-first implementation -> TWO more adversarial reviews of the diff (ruthless + security) -> apply/refute findings -> gates (preflight, record-lint) -> atomic commits + PR -> resolve the ledger issue with provenance -> seed the deferred items back into the ledger. Every hand-off, gate, and review-pair was driven by judgement, not a deterministic harness. The individual stages already have records — itd-83 (the review bar fires itself), itd-84 (decompose before filing), itd-82 (ledger drain), itd-78 (intent-dependency graph) — but ALL are drafts/staged and NONE composes the others into one orchestrated pipeline. Honest constraint (loud-staging): the pipeline cannot be MORE automated than its stages, so full automation waits on those disciplines shipping; but the deterministic SCAFFOLD — the fixed stage order, the mandatory review-pair gate before plan and before merge, the hand-offs to the host for the human-in-the-loop steps, the ledger-seed-the-deferrals close-out — can be built now as a workflow/routine (itd-107 versioned-routine-template is the nearest delivery vehicle). Routing note: this is a big, cross-record idea, not a decomposed intent — it likely needs /abcd:ideate then itd-84 decomposition before it becomes filable, and it may reshape rather than duplicate itd-83/82/78.

## Grounds

- declined: itd-2609201916151817 carries what this seed asked for; this would be wrong if that intent were superseded without a successor that orchestrates delivery
