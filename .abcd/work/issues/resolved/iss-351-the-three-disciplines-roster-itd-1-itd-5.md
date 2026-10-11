---
schema_version: 1
id: "iss-351"
slug: "the-three-disciplines-roster-itd-1-itd-5"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "bughunt-round-2"
found_at: ".abcd/development/brief/01-product/03-mental-model.md"
resolution: "roster sites point at intents/disciplines/; phase-scoped mentions say so"
impact: internal
---

the three-disciplines roster itd-1 itd-5 itd-37 is hand-enumerated at six sites while intents/disciplines/ holds six active disciplines, dropping itd-79 itd-81 itd-84 from the mental model, scope, roadmap dashboard and phase-0 acceptance
## Evidence

- Hand-enumerated "three disciplines (itd-1, itd-5, itd-37)" at `brief/01-product/03-mental-model.md:71`, `:79`, `:100`, `roadmap/README.md:83`, `brief/01-product/04-scope.md:27-29`, `roadmap/phases/phase-0-substrate.md:46,57,62`. `intents/disciplines/` holds six active discipline intents (`itd-79` persona registry, `itd-81` judge calibration, `itd-84` intent decomposition are dropped by the roster).
- `roadmap/README.md:69` directly above the stale table row says counts are "derived from the filesystem, never hand-kept"; `03-mental-model.md:71` itself warns hand-kept counts re-drift. Prior art: routed into iss-38's corpus in the 2026-07-08 review as "four (itd-79)" but never corrected; drifted to six since. `index_drift` has no entry covering these files. Leave `adr-2:35` (illustrative "Examples:") alone.
- Refuter verdict: CONFIRMED substantive. Fix: pointer to `intents/disciplines/` instead of a roster, the same migration `:71`/04-scope already made for the phased-intent set.
