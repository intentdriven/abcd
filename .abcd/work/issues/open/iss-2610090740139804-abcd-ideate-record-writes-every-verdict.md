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
remedy: "Make ideate record write the verdict to the person's home store first, ~/.abcd.noindex/ideas/<root-sha>/<date>-ideate-<slug>.md (owner-only, keyed on the repository's root commit like the history and transcript stores), with no DECISIONS.md line; add an explicit publish step (for example 'abcd ideate publish <slug>') that moves a chosen verdict into the research notes and writes the decision-log pointer, and say in commands/ideate.md that nothing reaches the repository until then."
---

abcd ideate record writes every verdict straight into the repository: a research note under .abcd/development/research/notes/ and a pointer line in .abcd/work/DECISIONS.md, both committed and public once pushed. The product thinker treats every idea put through ideate as secret: an idea and the sources it came from are kept private first and enter the repository only when the person decides it should.
