---
schema_version: 1
id: "iss-2609231931006041"
slug: "itd-147-ac-3-fidelity-audit-the-generated-appendix-of-the"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "autonomous run 2026-09-23 fidelity audit"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/13-consult.md"
---

itd-147 ac-3 (fidelity audit): the generated appendix of the three host-delegated commands says 'There is no shipped surface' while their register rows read shipped. UnbuiltSentence (internal/core/surface/appendix.go) is emitted for any chapter whose command the Go tree does not register, and /abcd:consult, /abcd:ingest and /abcd:prepare-this-repo are shipped host-delegated commands with a command page and no Go verb (04-surfaces/README.md, No skills). The block therefore states a false claim about the surface — the class the intent exists to remove — in 13-consult.md, 14-ingest.md and 15-prepare-this-repo.md. A host-delegated chapter needs its own sentence: shipped as a command page, with no Go verb and so no flags or sub-verbs to list; the generator can tell the cases apart from the register's Status column, which it already parses.
