---
schema_version: 1
id: "iss-2611060809343029"
slug: "abcd-development-brief-05-internals"
severity: "nitpick"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/05-prompt-quality.md"
remedy: "Reword line 101 to the later statement: no shipped prompt has a research file it can find by name."
---

`.abcd/development/brief/05-internals/05-prompt-quality.md:101` says research files do not exist for any shipped agent. `.abcd/development/research/prompting/agents/intent-fidelity-reviewer.md` is research for the shipped intent auditor (`.abcd/development/research/prompting/agents/README.md:44`), and :151-155 of the same chapter says it is stranded under the old name, so that no shipped prompt has a research file it can find.
