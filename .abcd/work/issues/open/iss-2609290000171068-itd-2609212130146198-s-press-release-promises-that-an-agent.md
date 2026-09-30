---
schema_version: 1
id: "iss-2609290000171068"
slug: "itd-2609212130146198-s-press-release-promises-that-an-agent"
severity: "minor"
category: "drift"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: fidelity audit itd-2609212130146198"
found_at: "internal/surface/cli/guard_question.go"
origin: researcher-authored
production_mode: hand-written
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (rulings-owed BJ, narrowing a shipped promise; run A 2026-09-29, lane drainDrift3): itd-2609212130146198's press release says an agent cannot ask the human anything until it has said whom it is asking, and the guard covers only the host's question tool, as the intent's scope and Decision 1 confine it. Should the press release's headline be narrowed to the question tool (its persona quote is already published on the v0.11.1 release page and would be left as quoted), or should a stop-time gate be built that also catches a question asked in prose at the end of a turn?"
remedy: "Waits on ruling BJ: if gated: a stop-time check refuses to end a turn whose final message asks the person a question while no role is set, proven by fixtures of a prose question and a plain statement; if narrowed: record the narrowing in the H10 shape, never rewriting the press-release headline: this issue resolved by the lane's commit plus an Audit Notes line on itd-2609212130146198 saying the guard covers the host's question tool, as the scope and Decision 1 confine it, leaving the published persona quote as quoted."
---

itd-2609212130146198's press release promises that an agent cannot ask the human anything until it has said whom it is asking, but the delivered guard (criterion 2) covers only the host's question tool: a stop that asks in prose at the end of a turn is not gated, so the badge can still read abcd-managed while an answer is owed, which is the Mechanism's own falsifier. The intent's scope confines the guard to the question tool, so the press release overclaims; either the promise narrows to the tool or a stop-time gate closes the prose path. Found by the fidelity audit at 52c2236a5 (internal/surface/cli/guard_question.go questionTools).

## Remedy grounds (2026-09-29)

Confirmed at this base: questionTools in internal/surface/cli/guard_question.go names the question tool alone. The gate's detector is heuristic (a question read from prose), so its false-positive cost is part of the ruling. Rejected: widening questionTools, which cannot see prose.
