---
schema_version: 1
id: "iss-2609020716236446"
slug: "the-register-rule-the-grill-domain-in-abcd-rules-json-and-th"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/rules.json"
---

The register rule (the GRILL domain in .abcd/rules.json and the itd-200 mode) tells an agent to address a product thinker in outcomes and choices and a technical facilitator in mechanism and ids, but it says nothing about ASSUMED KNOWLEDGE, and that is where explanations still miss. Observed during the 2026-09-02 design interview: a fork about repository ownership was first explained in terms of git's safe.directory, dubious-ownership refusals and the isolated git environment; the product thinker cannot be assumed to know git at all, while the technical facilitator can, and the same split holds for a merge queue, a worktree, a hook, an environment variable, a checksum, a symlink. The rule should carry an explicit knowledge floor per register: the product thinker knows the product's outcomes and the ordinary vocabulary of software use, not version control, shells, file permissions or CI; the technical facilitator knows those and the repo's record ids. An explanation for the product thinker names no tool and no mechanism the floor excludes, and where a concept is unavoidable it is introduced in one sentence in product terms (a notebook of pages, a lock, a name) before it is used. Worth deciding whether the floor is a list in the rule text, a glossary the rule points at, or both.
