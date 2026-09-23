---
schema_version: 1
id: "iss-390"
slug: "the-pre-decided-promotion-of-examples-use-reserved-identifie"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "bughunt-round-3"
found_at: ".abcd/development/principles/README.md"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (governance act owed: mint the discipline-kind intent for examples-use-reserved-identifiers per principles README). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

the pre-decided promotion of examples-use-reserved-identifiers to a discipline-kind intent never happened — the iss-154 lint shipped, the principles README contract says a mechanical gate promotes the principle, and no discipline intent exists
## Evidence

- `.abcd/development/principles/README.md:24-25` — "The moment a principle gains a mechanical gate (a lint code, a hook, a CI check), it is promoted to a discipline-kind intent … this directory is the not-yet-enforced layer."
- `.abcd/work/DECISIONS.md:881` — the iss-154 allowlist-inversion lint's shipping "promotes the principle to a discipline".
- The trigger fired (the lint is armed in `repolint`), yet `.abcd/development/intents/disciplines/` holds only itd-1, itd-5, itd-37, itd-79, itd-81, itd-84 — no intent for examples-use-reserved-identifiers.
- Refuter verdict: CONFIRMED (minor, process) — the promotion is pre-decided by two records, not an open design question. Minting a discipline-kind intent is a maintainer ceremony (an intent record, not an issue fix), so this record stays open as the tracker rather than being resolved by the autonomous round; iss-389 fixed the principle file's false prose in the same round.
