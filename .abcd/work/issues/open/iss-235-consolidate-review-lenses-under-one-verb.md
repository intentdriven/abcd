---
schema_version: 1
id: "iss-235"
slug: "consolidate-review-lenses-under-one-verb"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "2026-08-16 routine-prompt session (reviewer-agent fallback discussion)"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker, carried unanswered from the v0.10.0 deferral: should the review lenses become one verb, and under what name?"
remedy: "Waits on the review-lenses ruling (one verb, and its name): if consolidated: add one review verb (CLI and commands/<verb>.md) whose bare form lists the lenses from one registry of the reviewer agents under agents/ and whose lens flag runs one, hand-loading the agent's prompt where the host cannot resolve the agent type, proven by a test that every reviewer agent appears in the listing; if kept as agent types: wontfix the record naming the agent types as the surface."
---

The review lenses are scattered as separate agent TYPES (abcd:ruthless-reviewer, abcd:security-reviewer, and the persona-lens as an ad-hoc prompt) rather than a single surface. Consider consolidating them under one verb: bare /abcd:review lists the available lenses; /abcd:review --ruthless / --security / --persona (and future lenses) runs a named one. A single verb is easier to discover, keeps the lens set in one registry, gives cloud routines a stable fallback (the verb hand-loads the lens prompt when the plugin's agent types are unavailable — the 2026-08-16 canary found the abcd:* agent types do not resolve in a routine session), and matches how the other surfaces are shaped as verbs. Half-formed (maintainer thought): naming and whether lenses stay agent-backed under the hood are open.

## Remedy grounds (2026-09-29)

- The remedy keeps the lens definitions where they already live (agents/ruthless-reviewer.md, agents/security-reviewer.md) so a verb, if ruled, is a front door onto them rather than a second copy.
- Rejected: minting a verb name here, which is the open half of the ruling.
