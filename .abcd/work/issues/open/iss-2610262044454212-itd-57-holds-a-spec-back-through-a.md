---
schema_version: 1
id: "iss-2610262044454212"
slug: "itd-57-holds-a-spec-back-through-a"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1228 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-57-manual-hold-sentinel.md:32"
remedy: "Supersede or re-scope itd-57 together with the hold record: the native spec store (itd-80) is a minimal floor of link, status-by-directory and reconcile with no spec-to-spec dependency, no `depends_on`, no `spec rm-dep` and no selector that skips on one; the pick reads intent `blocked_by` and `held:` instead."
related_intents: [itd-57, itd-80]
finding_ref: cfk-6435ed7e7d7edffc
deferred_after: v0.15.1
deferral_reason: "Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes."
---

itd-57 holds a spec back through a native spec dependency that the selector already honours; the native spec store shipped by itd-80 has no dependencies between specs

A sequencing impossibility (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-57-manual-hold-sentinel.md:32` — "a spec is held by adding a native spec dependency on the sentinel (`depends_on: [<sentinel>]`); the run seam's selector already skips specs with a non-done dependency"
- End B: `.abcd/development/intents/shipped/itd-80-intent-lifecycle-automation.md:41` — "the store here is the minimal floor: link, status-by-directory, reconcile."

itd-57's whole mechanism is a `depends_on` edge on a sentinel spec, lifted with a native `spec rm-dep`. It relies on a run-seam selector that it says already skips specs with an unfinished dependency. itd-80, which delivered the native spec store, puts the dependency graph between specs out of scope. It describes the store as a minimal floor: link, status-by-directory, reconcile. As written, that store has none of the three things itd-57 builds on: the dependency edge, the `rm-dep` verb and the skipping selector.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1228.

## Deferral 2026-10-27

Deferred past v0.15.1: Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes.
