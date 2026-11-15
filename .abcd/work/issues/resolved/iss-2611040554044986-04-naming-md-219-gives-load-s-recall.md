---
schema_version: 1
id: "iss-2611040554044986"
slug: "04-naming-md-219-gives-load-s-recall"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md"
remedy: "Rewrite the row from the bundled domain: its actual recall words and aliases, and all four rules."
resolution: "The LOAD row of 02-constraints/04-naming.md is rewritten from the bundled domain: it lists the domain's six recall words and 31 aliases verbatim, summarises all four rules including the load gate for preflights and race runs, and no longer gives 'pegging or saturating every core' as recalling it. A test in internal/core/rules holds the row's two lists to the bundled domain's."
impact: internal
resolved_by:
  commit: "12629b1764020444fe1150a6bc5acd3556a54021"
---

04-naming.md:219 gives LOAD's recall keywords as burners, stress runs, nohup, a load test or load average, and pegging or saturating every core, and its injection as the trust rule. abcd rules --prompt matches nothing for pegging every core or saturating every core; only peg every core and saturate every core are aliases. The row omits the recall words pgrep, preflight, go test and race run (internal/core/rules/defaults/rules.json) and the domain's fourth rule, the load gate for preflights and race runs.

## Grounds

- pursued: the row's recall words and aliases equal those of internal/core/rules/defaults/rules.json and every phrase it lists recalls LOAD, with all four rules summarised; TestLoadNamingRowIsWrittenFromTheBundledDomain failing, or `abcd rules --prompt` matching nothing for a phrase the row lists, would show it wrong
