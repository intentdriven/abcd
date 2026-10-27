---
schema_version: 1
id: "iss-2610262036453716"
slug: "the-menu-intent-hides-the-guard-as-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 671 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-2610090831227812-a-person-who-types-abcd-sees-only-the.md:21"
remedy: "none (filed automatically)"
related_intents: [itd-2610090831227812]
finding_ref: cfk-0f6bb1a10932a662
---

The menu intent hides the guard as a command only agents run; the guard chapter says a person or a script asks the guard about a command line.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-2610090831227812-a-person-who-types-abcd-sees-only-the.md:21` — "Commands only agents run, such as the guard that judges a shell command, are hidden from the menu"
- End B: `.abcd/development/brief/04-surfaces/17-guard.md:11` — "A person or a script asks it about one command line."

The intent's press release gives the guard as its example of a command only agents run, and its ruling is that only commands no person types leave the person's menu. The guard chapter says two things use the guard: a person or a script asking about one command line, and an agent harness through the hook. It sets out the verdicts a person gets back. If a person asks it, the guard is a person's command by the intent's own rule.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 671.
