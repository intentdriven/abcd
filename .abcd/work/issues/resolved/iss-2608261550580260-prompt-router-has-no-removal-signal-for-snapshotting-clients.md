---
schema_version: 1
id: "iss-2608261550580260"
slug: "prompt-router-has-no-removal-signal-for-snapshotting-clients"
severity: "minor"
category: "future-work-seed"
source: "impl-review"
found_during: "second-harness adaptor lab review (2026-08-24/26)"
found_at: "internal/core/rules/inject.go"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25; rulings-owed J): Removal signal for snapshotting clients in the prompt-router protocol: the full active-domain set, or tombstones?"
resolution: "The prompt router names the full active-domain set on every prompt (ruling J15): abcd hook prompt-router --json writes {text, injected, active}, a name absent from active has stopped, the kill switch is an empty list, and an unevaluable prompt carries error and no active. The hook manifest keeps the plain form, so the model's context is unchanged; the session ledger forgets a domain that leaves the set so its return re-injects."
impact: additive
resolved_by:
  commit: "ec4e96c7f"
---

The prompt-router emits only domains whose content changed this turn, so a client that snapshots injected rules — a host-side adaptor staging them into a system prompt, or a future MCP consumer — keeps a domain that was deleted or renamed mid-session until session end: there is no removal signal in the protocol. Emit the full active-domain-name set, or tombstones for removals, so a snapshotting client can prune. A local second-harness adaptor lab hit this and had to accept the lingering snapshot as a recorded gap.

## Grounds

- pursued: a client that saw WIDGETS on turn 1 sees it absent from active on turn 2 when it is deleted, renamed, made dormant or killed (TestInjectActiveLetsAClientSeeADomainStop, TestHookPromptRouterJSONNamesStoppedDomain); shown wrong by any envelope naming a stopped domain, by a plain-form turn with nothing new writing a byte to stdout, or by a domain back in force being deduped against its old signature
