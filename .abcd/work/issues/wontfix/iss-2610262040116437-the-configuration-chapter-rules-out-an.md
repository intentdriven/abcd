---
schema_version: 1
id: "iss-2610262040116437"
slug: "the-configuration-chapter-rules-out-an"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 915 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md:411"
remedy: "none (filed automatically)"
related_intents: [itd-53]
finding_ref: cfk-0d5c9ce273cdbb56
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The shipped intent says its scope is read through the re-scope paragraph, which removes the autodrain configuration the quoted scope line still lists."
---

The configuration chapter rules out an autodrain config knob on purpose; shipped itd-53 scopes an opt-in `review.autodrain` key

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/03-configuration.md:411` — "There is deliberately no autodrain config knob — receipt gating is part of the seam contract"
- End B: `.abcd/development/intents/shipped/itd-53-review-queue-auto-drain-fidelity-gate.md:34` — "An opt-in `review.autodrain` configuration (default off)"

03-configuration says owed-review draining is receipt gating in the run seam, and that there is on purpose no autodrain config knob. itd-53, in shipped/, puts an opt-in `review.autodrain` configuration in scope, off by default, to drain pending fidelity reviews. Someone building from itd-53 would add the key that the configuration model rules out.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 915.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The shipped intent says its scope is read through the re-scope paragraph, which removes the autodrain configuration the quoted scope line still lists.
