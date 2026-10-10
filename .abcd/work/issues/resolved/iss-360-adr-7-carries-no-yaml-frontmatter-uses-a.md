---
schema_version: 1
id: "iss-360"
slug: "adr-7-carries-no-yaml-frontmatter-uses-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "bughunt-round-3"
found_at: ".abcd/development/decisions/adrs/0007-grill-skill-and-glossary.md"
resolution: "Prepended contract-conforming YAML frontmatter to adr-7 (id/slug/status/date/supersedes/superseded_by/related_intents/related_rfcs/related_adrs) and dropped the redundant prose status line; date 2026-05-11 from the index."
impact: internal
---

adr-7 carries no YAML frontmatter (uses a prose Status line) while the ADR format contract requires machine-readable frontmatter on every ADR; it is the sole outlier of 36 and passes record-lint ungated
## Evidence

- `.abcd/development/decisions/adrs/0007-grill-skill-and-glossary.md:1-3` — line 1 `# ADR-7:`, line 3 `**Status:** Accepted`, no leading `---`.
- `.abcd/development/decisions/adrs/README.md:47` — "Every ADR has frontmatter (machine-readable) …".

All 36 ADRs begin with `---` except adr-7. record-lint's `record_schema` (iss-39) does not assert frontmatter *presence*, so adr-7 passes ungated. adr-7 is in no supersession chain; adding the block breaks nothing.

## Adversarial verdict

CONFIRMED (substantive, minor). Sole outlier; contract requires frontmatter; no documented exemption. Fix: prepend a frontmatter block mirroring the contract, date taken from the original commit.
