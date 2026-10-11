---
schema_version: 1
id: "iss-295"
slug: "readme-md-s-it-starts-with-the-brief"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "bughunt-round-1"
found_at: "README.md"
resolution: "README brief-onboarding rewritten to describe the shipped facilitator flow, with the ingest/Socratic automation marked as a not-yet-shipped design target."
impact: fix
---

README.md's 'It starts with the brief' section claims in present tense that abcd 'has a skill that ingests that material and produces a plain-language draft of your project's brief' and that you sharpen it with 'a Socratic interview the framework provides' — but abcd ships zero skills (surface is commands-only) and the grill/Socratic interview is an unshipped planned intent (itd-27), an unmarked not-yet-real claim the README's own honesty rule forbids
## Evidence

- `README.md` "It starts with the brief" — present-tense "`abcd` has a skill that ingests
  that material and produces a plain-language draft of your project's brief" and "a Socratic
  interview the framework provides".
- `docs/reference/terminology.md` and `.abcd/development/brief/05-internals/08-skills.md`:
  "abcd ships zero skills" (commands-only surface); no `skills/` dir or `SKILL.md` exists, and
  no command/agent drafts a brief.
- The grill / Socratic interview is planned intent `itd-27`
  (`.abcd/development/intents/planned/`); `.abcd/development/brief/04-surfaces/README.md` says
  the `intent` parent "ships no `grill` sub-verb yet".
- `README.md` states its own rule earlier: "every ambitious passage is visibly marked as not
  yet real" — this passage is unmarked.

## Adversarial review

CONFIRMED (substantive) by two independent refuters (one per claim half): both halves name
unshipped capability in unmarked present tense, breaking the README's own honesty rule. Fix:
mark the passage as a not-yet-real design target in the README's own idiom.
