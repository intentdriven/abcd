---
schema_version: 1
id: "iss-2610090740139804"
slug: "abcd-ideate-record-writes-every-verdict"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "2026-10-09 product thinker capture"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/ideate.md"
remedy: "Product thinker ruling 2026-10-14: when ideate runs, ask the person whether the idea is private or public before writing anything, then write the verdict to ~/.abcd.noindex/ideas/<root-sha>/<date>-ideate-<slug>.md (private: owner-only home store keyed on the repository's root commit, no DECISIONS.md line) or to .abcd/development/research/ideas/<root-sha>/<date>-ideate-<slug>.md (public, with its decision-log pointer); never write without the answer, and say in commands/ideate.md which store each answer reaches. Supersedes the earlier remedy (private-only write plus a later publish step)."
resolution: "ideate record now takes the product thinker's answer as exactly one of --private or --public and refuses without it: a private verdict lands owner-only in ~/.abcd.noindex/ideas/<root-sha>/ with no decision-log line, a public one under .abcd/development/research/ideas/<root-sha>/ with its pointer; commands/ideate.md asks the question before the first leg."
impact: breaking
---

abcd ideate record writes every verdict straight into the repository: a research note under .abcd/development/research/notes/ and a pointer line in .abcd/work/DECISIONS.md, both committed and public once pushed. The product thinker treats every idea put through ideate as secret: an idea and the sources it came from are kept private first and enter the repository only when the person decides it should.
