---
schema_version: 1
id: "iss-23"
slug: "itd-36-adr-28-reconcile"
severity: "minor"
impact: internal
category: "inconsistency"
source: "agent-finding"
found_during: "intent-dependency-sweep"
found_at: ".abcd/development/intents/shipped/itd-36-memory-unification.md"
resolution: "Body reconciled with adr-28 ahead of spec planning: every launch-gate / '/abcd:launch refuses' reference (Why This Matters, both scope bullets, two GWT criteria, the mixed-licence test scenario) now names the gate's real consumer — the lifeboat restrictive-licence gate run by /abcd:disembark, with .abcd/launch-allowlist.json re-including files into the gate's own evaluation input only. The read-every-X-as-Y banner shrank to a pure framing pointer since the body no longer needs translation."
---

itd-36 is partially superseded by adr-28 (launch-gate framing rewritten); reconcile body with the ADR before its spec plans