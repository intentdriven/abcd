---
schema_version: 1
id: "iss-235"
slug: "consolidate-review-lenses-under-one-verb"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "2026-08-16 routine-prompt session (reviewer-agent fallback discussion)"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (ruling owed: consolidate review lenses under one verb; naming and agent-backing open). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

The review lenses are scattered as separate agent TYPES (abcd:ruthless-reviewer, abcd:security-reviewer, and the persona-lens as an ad-hoc prompt) rather than a single surface. Consider consolidating them under one verb: bare /abcd:review lists the available lenses; /abcd:review --ruthless / --security / --persona (and future lenses) runs a named one. A single verb is easier to discover, keeps the lens set in one registry, gives cloud routines a stable fallback (the verb hand-loads the lens prompt when the plugin's agent types are unavailable — the 2026-08-16 canary found the abcd:* agent types do not resolve in a routine session), and matches how the other surfaces are shaped as verbs. Half-formed (maintainer thought): naming and whether lenses stay agent-backed under the hood are open.