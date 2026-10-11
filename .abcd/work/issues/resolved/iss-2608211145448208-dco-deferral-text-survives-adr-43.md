---
schema_version: 1
id: "iss-2608211145448208"
slug: "dco-deferral-text-survives-adr-43"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "bughunt-b-round-4"
found_at: "AGENTS.md"
resolution: "Retired the DCO deferral text from AGENTS.md and scripts/check-attribution.sh (comment + runtime note), replacing it with adr-43's decision (no DCO; inbound = outbound MIT; no Signed-off-by required), consistent with CONTRIBUTING.md. The blanket co-authorship refusal is preserved — with no Signed-off-by it costs nothing. check-attribution-cases.sh: 74/74 pass."
impact: internal
---

adr-43 (accepted 2026-08-19) decided 'No CLA, and no DCO' and named as its own consequence 'the DCO deferral text retires from every surface that carried it', but the deferral survived in two live surfaces the ADR's landing commit did not touch: AGENTS.md (symlinked as CLAUDE.md/GEMINI.md, injected into every agent session) said 'A human-only Signed-off-by: (DCO) is deferred to the public flip or the first outside contribution', and scripts/check-attribution.sh carried it twice — a comment and the runtime error text shown to a contributor whose commit trips COAUTHOR_RE — stating the false premise that the repo defers DCO until it goes public (it is already public). CONTRIBUTING.md already says 'no CLA and no Signed-off-by requirement', so the surfaces contradicted each other.