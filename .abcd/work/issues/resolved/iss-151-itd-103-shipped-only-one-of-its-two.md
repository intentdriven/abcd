---
schema_version: 1
id: "iss-151"
slug: "itd-103-shipped-only-one-of-its-two"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25; rulings-owed J): Wire itd-103's teaching plane (a rules-loader safety domain generated from the hazard registry), or amend itd-103 to drop it?"
resolution: "itd-103's teaching plane is built: the rules loader bundles a SHELL domain generated from the bundled hazard registry the guard reads, one rule per entry and one recall term per command head (ruling J10)"
impact: additive
resolved_by:
  intent: "itd-103"
  commit: "c707d788909b08521ff0fccd392144257d967e04"
---

itd-103 shipped only ONE of its two promised planes: the execution-time guard plane is fully wired (abcd guard check/hook, ahoy health), but the TEACHING plane — the rules loader injecting matched shell-safety rules before shell-heavy work, per spc-16 'Two planes, one registry' — is not built; there is no guard/safety/hazard domain in internal/core/rules/. All four itd-103 ACs concern the guard plane only, so the fidelity review is 3 MET/1 MWC, but the intent headline ('two planes') is only half delivered. Wire the teaching plane as a rules-loader safety domain sourced from the same bundled hazard registry, or amend the intent's scope.

## Grounds

- pursued: a prompt about shell-heavy work is taught the registry's safe forms before any command runs, from the one registry the guard enforces; shown wrong if an entry added to or removed from the registry leaves the SHELL domain unchanged, or a shell-heavy prompt injects no SHELL rules
