---
schema_version: 1
id: "iss-2610262035592770"
slug: "itd-140-s-armed-detector-rung-lives-in"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 615 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-140-the-generic-specific-boundary-of-the.md:55"
remedy: "none (filed automatically)"
related_intents: [itd-140]
finding_ref: cfk-8fccf072420e0e93
---

itd-140's armed-detector rung lives in `abcd site check`; 16-lint says the site verb's former check sub-verb is an unknown command.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-140-the-generic-specific-boundary-of-the.md:55` — "site check\` refusing a repo-agnostic claim without the fixture receipt"
- End B: `.abcd/development/brief/04-surfaces/16-lint.md:44` — "The docs and site verbs' former check"

itd-140's build path places the armed-detector rung in `abcd site check` refusing a repo-agnostic claim without the fixture receipt. 16-lint says the docs and site verbs' former check sub-verbs are unknown commands, the site gate now being the site target of the one lint. The discipline names a verb that no longer exists as the home of its enforcement, so one gate carries two names and the one the discipline uses cannot be built on.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 615.
