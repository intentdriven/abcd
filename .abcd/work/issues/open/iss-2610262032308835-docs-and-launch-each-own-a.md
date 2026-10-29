---
schema_version: 1
id: "iss-2610262032308835"
slug: "docs-and-launch-each-own-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 278 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/10-docs.md:140"
remedy: "In 10-docs.md:140 say the launch gate refuses the ambiguous pairs the lint only warns about, and in 04-launch.md:304 name the docs lint's advisory tier as the deliberately weaker CI check, so neither chapter reads as the sole authority on change narration."
finding_ref: cfk-8dbe1c46c7bdf182
---

docs and launch each own a change-narration check over the same doc bodies and give opposite verdicts on an ambiguous phrase: a warning in the docs lint, a refusal in launch

A scope leakage (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/10-docs.md:140` — "Unambiguous change-narration blocks; phrases that can also describe present state warn advisorily rather than block."
- End B: `.abcd/development/brief/04-surfaces/04-launch.md:304` — "A pair the words cannot tell apart is refused, not passed: the tier is a release gate"

10-docs.md gives change-narration to the currency lint, which it presents as the check that gates a release, and says phrases that can also describe present state only warn. 04-launch.md runs its own hard-fail change-narration gate over the shipped doc bodies and refuses any pair the words cannot tell apart. The same ground is covered twice with contradictory outcomes, so a sentence that passes CI's docs lint with a warning refuses the cut. Neither chapter names the other's rule as the authority.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 278.
