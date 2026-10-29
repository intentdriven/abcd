---
schema_version: 1
id: "iss-2610230734298145"
slug: "the-command-surface-needs-another"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "PT request during the 2026-10-23 morning interview"
origin: researcher-authored
production_mode: hand-written
found_at: "commands"
remedy: "run a surface review that lists every top-level command and sub-verb with its audience (person or agent), its part of speech and every flag with its meaning; propose one naming and flag convention as an ADR for the product thinker to rule on; then rename what breaks it behind deprecation aliases, enforced by a surface lint so new commands follow the convention"
---

The command surface needs another consistency review on three axes. (1) Human verbs versus agent verbs: which commands a person types and which only an agent drives (the plugin pages' user-invocable split), and whether each is named and documented for its real audience. (2) Verbs versus nouns: some commands are actions and some are things, for example abcd drain (a verb) beside abcd lab (a noun), with no stated rule for which form a command takes. (3) Consistent flags: the same idea should be spelt the same way on every command, for example a select-everything flag (drain takes --max for a cap; is there an --all?), --dry-run against a bare read-only form, --apply, --json, --release and --take-over. Raised by the product thinker on 2026-10-23.
