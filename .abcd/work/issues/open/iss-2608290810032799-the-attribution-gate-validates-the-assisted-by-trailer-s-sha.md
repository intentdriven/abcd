---
schema_version: 1
id: "iss-2608290810032799"
slug: "the-attribution-gate-validates-the-assisted-by-trailer-s-sha"
severity: "minor"
category: "process"
source: "impl-review"
found_during: "intent-implementation-run"
found_at: "scripts/check-attribution.sh"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Keep a known-models list for Assisted-by truthfulness, or trust the harness-supplied model id?"
---

The attribution gate validates the Assisted-by trailer's SHAPE but cannot tell whether the model id names the model that actually did the work, so a wrong but well-formed id passes green. In the itd-150..162 run four implementation agents were briefed with an incorrect model id: two used it as instructed and their commits passed the gate, two recognised it as a false disclosure and stamped their real id instead. The convention exists precisely to prevent a false disclosure, and the gate is blind to the only failure that matters. A full mechanical fix may be impossible because the runner cannot attest which model produced a commit, but two partial rungs exist: refuse a vendor and model pair absent from a maintained known-models list, so a typo or a stale id is caught; and have the dispatching harness supply its own model id to the agent, so a hand-written briefing cannot introduce a wrong one. Distinct from iss-211 (making the attribution preference portable) and iss-220 (composing the trailer at creation time): this is about the truthfulness of a trailer that is present and well-formed.