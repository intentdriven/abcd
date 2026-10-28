---
schema_version: 1
id: "iss-2610262030439309"
slug: "the-mental-model-gives-intent-auditor-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 37 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/03-mental-model.md:85"
remedy: "none (filed automatically)"
finding_ref: cfk-31cda9ef21e46978
---

The mental model gives intent-auditor a discipline role that checks every spec; the intent chapter gives it three roles, none of which does

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/03-mental-model.md:85` — "the same agent's discipline role checks every spec against the active disciplines"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:532` — "`intent-auditor` is a single agent in the catalog (per `05-internals/01-agents.md`) that owns three roles"

03-mental-model says the same agent's discipline role checks every spec against the active disciplines, while 05-intent says intent-auditor owns three roles (single-document fidelity, cross-document consistency, kind classification) and that no gate reads a discipline's body. The discipline role the mental model relies on is not one of the roles the surface defines.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 37.
