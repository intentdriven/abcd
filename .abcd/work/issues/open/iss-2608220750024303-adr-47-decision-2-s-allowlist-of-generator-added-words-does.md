---
schema_version: 1
id: "iss-2608220750024303"
slug: "adr-47-decision-2-s-allowlist-of-generator-added-words-does"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "agent-observation"
found_at: "docs/explanation/rationale.md"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (renewed by run A 2026-09-29 after the v0.10.0 grant lapsed at the v0.11.0 anchor): Amend or supersede adr-47 decision 2 to admit image alt text as a generator-added word?"
---

adr-47 decision 2's allowlist of generator-added words does not name image alt text, yet the migrated docs pages carry alt text the site inherits; the carve-out wants an explicit ADR amendment