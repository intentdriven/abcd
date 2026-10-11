---
schema_version: 1
id: "iss-2608280801427648"
slug: "whether-an-adopted-open-licensed-style"
severity: "nitpick"
category: "observation"
source: "user-observation"
found_during: "itd-141-planning-interview"
found_at: ".abcd/development/intents/superseded/itd-141-docs-lint-enforces-the-writing-style.md"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25; rulings-owed I): Adopt an external style guide named in rules.json (its own intent per adr-54), or close?"
remedy: "Waits on the external-style-guide ruling (adopt, or close): if adopted: file it as its own intent (adr-54) in which .abcd/rules.json gains one field naming the guide by its Vale package-library name and docs-lint re-implements only the chosen rules, adding no Vale binary without the product thinker's sign-off, proven by a docs-lint test on a fixture repository naming each supported guide; if not: wontfix the record, keeping the house writing guide as the one source."
---

whether an adopted open-licensed style guide (Google, Microsoft, GitLab, errata-ai/Vale packages) replaces parts of the house writing guide, and how a managed repo names its chosen guide in .abcd/rules.json: routed out of itd-141 at its closure (adr-54); files as its own intent if it goes live

## Remedy grounds (2026-09-29)

- Vale projects name a guide by its package-library name (Google, Microsoft, write-good) in a Packages key (https://docs.vale.sh/keys/packages, checked 2026-09-30), which gives rules.json a naming scheme managed repositories already recognise.
- Adversary filter (prefer-sota): running Vale itself is a new external tool, and a vendor guide's rules can collide with the house guide's host-agnostic and British-English rules, so the remedy borrows the naming, not the engine. Rejected: bundling the packages.
